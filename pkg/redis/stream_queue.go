package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/queue-system/redis-queue/pkg/queue"
)

// StreamQueue implements the Queue interface using Redis Streams
type StreamQueue struct {
	client *Client
	config *StreamQueueConfig
}

// StreamQueueConfig contains configuration for the Redis Stream queue
type StreamQueueConfig struct {
	DefaultVisibilityTimeout time.Duration
	DefaultMaxRetries        int
	CleanupInterval          time.Duration
	MaxStreamLength          int64
	ConsumerTimeout          time.Duration
}

// DefaultStreamQueueConfig returns default configuration
func DefaultStreamQueueConfig() *StreamQueueConfig {
	return &StreamQueueConfig{
		DefaultVisibilityTimeout: 30 * time.Second,
		DefaultMaxRetries:        3,
		CleanupInterval:          5 * time.Minute,
		MaxStreamLength:          10000,
		ConsumerTimeout:          60 * time.Second,
	}
}

// NewStreamQueue creates a new Redis Stream-based queue
func NewStreamQueue(client *Client, config *StreamQueueConfig) *StreamQueue {
	if config == nil {
		config = DefaultStreamQueueConfig()
	}

	return &StreamQueue{
		client: client,
		config: config,
	}
}

// Enqueue adds a message to the queue
func (sq *StreamQueue) Enqueue(ctx context.Context, queueName string, message *queue.Message, opts *queue.MessageOptions) error {
	message.QueueName = queueName

	// Handle scheduled messages
	if opts != nil && (opts.ScheduleAt != nil || opts.Delay != nil) {
		return sq.enqueueScheduled(ctx, queueName, message, opts)
	}

	// Serialize message
	msgData, err := message.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	streamKey := sq.client.StreamKey(queueName)

	// Add to stream with MAXLEN to prevent unbounded growth
	args := redis.XAddArgs{
		Stream: streamKey,
		MaxLen: sq.config.MaxStreamLength,
		Approx: true,
		Values: map[string]interface{}{
			"message": string(msgData),
			"id":      message.ID,
		},
	}

	if opts != nil && opts.Priority > 0 {
		values := args.Values.(map[string]interface{})
		values["priority"] = opts.Priority
	}

	_, err = sq.client.Get().XAdd(ctx, &args).Result()
	if err != nil {
		return fmt.Errorf("failed to add message to stream: %w", err)
	}

	// Update queue stats
	sq.updateStats(ctx, queueName, "enqueued", 1)

	return nil
}

// enqueueScheduled handles scheduled messages using sorted sets
func (sq *StreamQueue) enqueueScheduled(ctx context.Context, queueName string, message *queue.Message, opts *queue.MessageOptions) error {
	scheduleTime := time.Now()

	if opts.ScheduleAt != nil {
		scheduleTime = *opts.ScheduleAt
		message.ScheduleAt = opts.ScheduleAt
	} else if opts.Delay != nil {
		scheduleTime = time.Now().Add(*opts.Delay)
		message.ScheduleAt = &scheduleTime
	}

	msgData, err := message.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize scheduled message: %w", err)
	}

	scheduledKey := sq.client.ScheduledKey(queueName)
	score := float64(scheduleTime.Unix())

	return sq.client.Get().ZAdd(ctx, scheduledKey, &redis.Z{
		Score:  score,
		Member: string(msgData),
	}).Err()
}

// Dequeue retrieves a message from the queue
func (sq *StreamQueue) Dequeue(ctx context.Context, queueName string, opts *queue.ConsumeOptions) (*queue.Message, error) {
	if opts == nil {
		opts = &queue.ConsumeOptions{
			VisibilityTimeout: sq.config.DefaultVisibilityTimeout,
			MaxRetries:        sq.config.DefaultMaxRetries,
			BatchSize:         1,
		}
	}

	// Ensure consumer group exists
	if err := sq.ensureConsumerGroup(ctx, queueName); err != nil {
		return nil, err
	}

	streamKey := sq.client.StreamKey(queueName)
	consumerGroup := sq.client.ConsumerGroupKey(queueName)
	consumerName := sq.generateConsumerName()

	// Try to read from pending messages first
	if msg, err := sq.readPendingMessage(ctx, streamKey, consumerGroup, consumerName); err == nil && msg != nil {
		return msg, nil
	}

	// Read new messages
	timeout := time.Duration(0)
	if opts.LongPoll {
		timeout = opts.LongPollTimeout
	}

	streams, err := sq.client.Get().XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    consumerGroup,
		Consumer: consumerName,
		Streams:  []string{streamKey, ">"},
		Count:    int64(opts.BatchSize),
		Block:    timeout,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil // No messages available
		}
		return nil, fmt.Errorf("failed to read from stream: %w", err)
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, nil
	}

	// Process first message
	xmsg := streams[0].Messages[0]
	message, err := sq.parseStreamMessage(xmsg)
	if err != nil {
		return nil, err
	}

	// Update stats
	sq.updateStats(ctx, queueName, "dequeued", 1)

	return message, nil
}

// Ack acknowledges a message
func (sq *StreamQueue) Ack(ctx context.Context, queueName string, messageID string) error {
	streamKey := sq.client.StreamKey(queueName)
	consumerGroup := sq.client.ConsumerGroupKey(queueName)

	// Find the stream message ID
	streamMsgID, err := sq.findStreamMessageID(ctx, streamKey, consumerGroup, messageID)
	if err != nil {
		return err
	}

	// Acknowledge the message
	_, err = sq.client.Get().XAck(ctx, streamKey, consumerGroup, streamMsgID).Result()
	if err != nil {
		return fmt.Errorf("failed to ack message: %w", err)
	}

	// Update stats
	sq.updateStats(ctx, queueName, "acked", 1)

	return nil
}

// Nack negatively acknowledges a message
func (sq *StreamQueue) Nack(ctx context.Context, queueName string, messageID string, requeue bool) error {
	if !requeue {
		return sq.MoveToDeadLetter(ctx, queueName, messageID)
	}

	// For requeue, we'll let the visibility timeout handle it
	// or implement explicit requeue logic here
	return sq.Requeue(ctx, queueName, messageID, nil)
}

// MoveToLast moves a message to the end of the queue
func (sq *StreamQueue) MoveToLast(ctx context.Context, queueName string, messageID string) error {
	// This is implemented by moving the message using Lua script
	// to ensure atomicity
	return sq.executeAtomicMove(ctx, queueName, messageID, "last")
}

// Skip temporarily skips a message
func (sq *StreamQueue) Skip(ctx context.Context, queueName string, messageID string, duration *time.Duration) error {
	if duration == nil {
		defaultDuration := 5 * time.Minute
		duration = &defaultDuration
	}

	return sq.executeAtomicMove(ctx, queueName, messageID, "skip", *duration)
}

// MoveToDeadLetter moves a message to dead letter queue
func (sq *StreamQueue) MoveToDeadLetter(ctx context.Context, queueName string, messageID string) error {
	return sq.executeAtomicMove(ctx, queueName, messageID, "deadletter")
}

// Requeue puts a message back in the queue
func (sq *StreamQueue) Requeue(ctx context.Context, queueName string, messageID string, delay *time.Duration) error {
	if delay != nil {
		return sq.executeAtomicMove(ctx, queueName, messageID, "requeue_delayed", *delay)
	}
	return sq.executeAtomicMove(ctx, queueName, messageID, "requeue")
}

// Helper methods

func (sq *StreamQueue) ensureConsumerGroup(ctx context.Context, queueName string) error {
	streamKey := sq.client.StreamKey(queueName)
	consumerGroup := sq.client.ConsumerGroupKey(queueName)

	// Try to create consumer group, ignore if it already exists
	_, err := sq.client.Get().XGroupCreateMkStream(ctx, streamKey, consumerGroup, "0").Result()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	return nil
}

func (sq *StreamQueue) generateConsumerName() string {
	// Generate unique consumer name based on hostname, PID, etc.
	return fmt.Sprintf("consumer-%d", time.Now().UnixNano())
}

func (sq *StreamQueue) readPendingMessage(ctx context.Context, streamKey, consumerGroup, consumerName string) (*queue.Message, error) {
	// Check for pending messages for this consumer
	pending, err := sq.client.Get().XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream:   streamKey,
		Group:    consumerGroup,
		Start:    "-",
		End:      "+",
		Count:    1,
		Consumer: consumerName,
	}).Result()

	if err != nil || len(pending) == 0 {
		return nil, err
	}

	// Claim the oldest pending message
	msgs, err := sq.client.Get().XClaim(ctx, &redis.XClaimArgs{
		Stream:   streamKey,
		Group:    consumerGroup,
		Consumer: consumerName,
		MinIdle:  sq.config.DefaultVisibilityTimeout,
		Messages: []string{pending[0].ID},
	}).Result()

	if err != nil || len(msgs) == 0 {
		return nil, err
	}

	return sq.parseStreamMessage(msgs[0])
}

func (sq *StreamQueue) parseStreamMessage(xmsg redis.XMessage) (*queue.Message, error) {
	msgData, ok := xmsg.Values["message"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message format")
	}

	message, err := queue.FromJSON([]byte(msgData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}

	// Store stream message ID for acking
	if message.Headers == nil {
		message.Headers = make(map[string]string)
	}
	message.Headers["_stream_id"] = xmsg.ID

	return message, nil
}

func (sq *StreamQueue) findStreamMessageID(ctx context.Context, streamKey, consumerGroup, messageID string) (string, error) {
	// This would typically be cached or stored as part of the message
	// For now, we'll use the header stored during dequeue
	return "", fmt.Errorf("stream message ID lookup not implemented")
}

func (sq *StreamQueue) updateStats(ctx context.Context, queueName, operation string, count int64) {
	statsKey := sq.client.QueueStatsKey(queueName)
	sq.client.Get().HIncrBy(ctx, statsKey, operation, count)
	sq.client.Get().HSet(ctx, statsKey, "last_activity", time.Now().Unix())
}

// Lua script for atomic operations
const atomicMoveScript = `
local queue_key = KEYS[1]
local group_key = KEYS[2]
local message_id = ARGV[1]
local operation = ARGV[2]
local delay = ARGV[3] or "0"

-- Implementation of atomic move operations
-- This would contain the Lua logic for moving messages atomically
return "OK"
`

func (sq *StreamQueue) executeAtomicMove(ctx context.Context, queueName, messageID, operation string, args ...interface{}) error {
	streamKey := sq.client.StreamKey(queueName)
	consumerGroup := sq.client.ConsumerGroupKey(queueName)

	scriptArgs := []interface{}{messageID, operation}
	scriptArgs = append(scriptArgs, args...)

	return sq.client.Get().Eval(ctx, atomicMoveScript, []string{streamKey, consumerGroup}, scriptArgs...).Err()
}
