package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/harshaweb/Queue/pkg/queue"
)

// BatchEnqueue adds multiple messages to a queue
func (sq *StreamQueue) BatchEnqueue(ctx context.Context, queueName string, messages []*queue.Message, opts *queue.MessageOptions) error {
	if len(messages) == 0 {
		return nil
	}

	streamKey := sq.client.StreamKey(queueName)
	pipe := sq.client.Get().Pipeline()

	for _, message := range messages {
		message.QueueName = queueName

		// Handle scheduled messages separately
		if opts != nil && (opts.ScheduleAt != nil || opts.Delay != nil) {
			if err := sq.enqueueScheduled(ctx, queueName, message, opts); err != nil {
				return err
			}
			continue
		}

		msgData, err := message.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to serialize message %s: %w", message.ID, err)
		}

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

		pipe.XAdd(ctx, &args)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to batch enqueue messages: %w", err)
	}

	// Update stats
	sq.updateStats(ctx, queueName, "enqueued", int64(len(messages)))

	return nil
}

// BatchDequeue retrieves multiple messages from a queue
func (sq *StreamQueue) BatchDequeue(ctx context.Context, queueName string, count int, opts *queue.ConsumeOptions) ([]*queue.Message, error) {
	if count <= 0 {
		return nil, nil
	}

	if opts == nil {
		opts = &queue.ConsumeOptions{
			VisibilityTimeout: sq.config.DefaultVisibilityTimeout,
			MaxRetries:        sq.config.DefaultMaxRetries,
			BatchSize:         count,
		}
	} else {
		opts.BatchSize = count
	}

	// Ensure consumer group exists
	if err := sq.ensureConsumerGroup(ctx, queueName); err != nil {
		return nil, err
	}

	streamKey := sq.client.StreamKey(queueName)
	consumerGroup := sq.client.ConsumerGroupKey(queueName)
	consumerName := sq.generateConsumerName()

	// Read messages in batch
	timeout := time.Duration(0)
	if opts.LongPoll {
		timeout = opts.LongPollTimeout
	}

	streams, err := sq.client.Get().XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    consumerGroup,
		Consumer: consumerName,
		Streams:  []string{streamKey, ">"},
		Count:    int64(count),
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

	// Parse all messages
	var messages []*queue.Message
	for _, xmsg := range streams[0].Messages {
		message, err := sq.parseStreamMessage(xmsg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse message %s: %w", xmsg.ID, err)
		}
		messages = append(messages, message)
	}

	// Update stats
	sq.updateStats(ctx, queueName, "dequeued", int64(len(messages)))

	return messages, nil
}

// ProcessScheduledMessages moves scheduled messages that are ready to the main queue
func (sq *StreamQueue) ProcessScheduledMessages(ctx context.Context) error {
	queues, err := sq.ListQueues(ctx)
	if err != nil {
		return fmt.Errorf("failed to list queues: %w", err)
	}

	now := time.Now().Unix()

	for _, queueName := range queues {
		if err := sq.processScheduledMessagesForQueue(ctx, queueName, now); err != nil {
			// Log error but continue with other queues
			continue
		}
	}

	return nil
}

func (sq *StreamQueue) processScheduledMessagesForQueue(ctx context.Context, queueName string, now int64) error {
	scheduledKey := sq.client.ScheduledKey(queueName)
	streamKey := sq.client.StreamKey(queueName)

	// Get messages that are ready to be processed
	readyMessages, err := sq.client.Get().ZRangeByScoreWithScores(ctx, scheduledKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("%d", now),
		Count: 100, // Process in batches
	}).Result()

	if err != nil || len(readyMessages) == 0 {
		return err
	}

	pipe := sq.client.Get().Pipeline()

	for _, zmsg := range readyMessages {
		msgData := zmsg.Member.(string)

		// Parse the message
		var message queue.Message
		if err := json.Unmarshal([]byte(msgData), &message); err != nil {
			continue // Skip invalid messages
		}

		// Add to main stream
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: streamKey,
			MaxLen: sq.config.MaxStreamLength,
			Approx: true,
			Values: map[string]interface{}{
				"message": msgData,
				"id":      message.ID,
			},
		})

		// Remove from scheduled set
		pipe.ZRem(ctx, scheduledKey, msgData)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to process scheduled messages for queue %s: %w", queueName, err)
	}

	return nil
}

// ExportQueue exports queue data for backup/migration
func (sq *StreamQueue) ExportQueue(ctx context.Context, queueName string) ([]byte, error) {
	export := struct {
		QueueName string            `json:"queue_name"`
		Config    json.RawMessage   `json:"config"`
		Messages  []string          `json:"messages"`
		Stats     map[string]string `json:"stats"`
		Timestamp time.Time         `json:"timestamp"`
	}{
		QueueName: queueName,
		Timestamp: time.Now(),
	}

	// Get queue config
	configKey := sq.client.QueueConfigKey(queueName)
	configData, err := sq.client.Get().Get(ctx, configKey).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get queue config: %w", err)
	}
	if configData != "" {
		export.Config = json.RawMessage(configData)
	}

	// Get queue stats
	statsKey := sq.client.QueueStatsKey(queueName)
	stats, err := sq.client.Get().HGetAll(ctx, statsKey).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}
	export.Stats = stats

	// Get all messages from stream
	streamKey := sq.client.StreamKey(queueName)
	messages, err := sq.client.Get().XRange(ctx, streamKey, "-", "+").Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	for _, msg := range messages {
		if msgData, ok := msg.Values["message"].(string); ok {
			export.Messages = append(export.Messages, msgData)
		}
	}

	// Get scheduled messages
	scheduledKey := sq.client.ScheduledKey(queueName)
	scheduledMsgs, err := sq.client.Get().ZRangeWithScores(ctx, scheduledKey, 0, -1).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get scheduled messages: %w", err)
	}

	for _, zmsg := range scheduledMsgs {
		export.Messages = append(export.Messages, zmsg.Member.(string))
	}

	return json.Marshal(export)
}

// ImportQueue imports queue data from backup
func (sq *StreamQueue) ImportQueue(ctx context.Context, queueName string, data []byte) error {
	var importData struct {
		QueueName string            `json:"queue_name"`
		Config    json.RawMessage   `json:"config"`
		Messages  []string          `json:"messages"`
		Stats     map[string]string `json:"stats"`
		Timestamp time.Time         `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &importData); err != nil {
		return fmt.Errorf("failed to parse import data: %w", err)
	}

	// Import queue config
	if len(importData.Config) > 0 {
		var config queue.QueueConfig
		if err := json.Unmarshal(importData.Config, &config); err != nil {
			return fmt.Errorf("failed to parse queue config: %w", err)
		}
		config.Name = queueName
		if err := sq.CreateQueue(ctx, &config); err != nil && err != queue.ErrQueueExists {
			return fmt.Errorf("failed to create queue: %w", err)
		}
	}

	// Import messages
	if len(importData.Messages) > 0 {
		var messages []*queue.Message
		for _, msgData := range importData.Messages {
			msg, err := queue.FromJSON([]byte(msgData))
			if err != nil {
				continue // Skip invalid messages
			}
			msg.QueueName = queueName
			messages = append(messages, msg)
		}

		if err := sq.BatchEnqueue(ctx, queueName, messages, nil); err != nil {
			return fmt.Errorf("failed to import messages: %w", err)
		}
	}

	// Import stats
	if len(importData.Stats) > 0 {
		statsKey := sq.client.QueueStatsKey(queueName)
		if err := sq.client.Get().HMSet(ctx, statsKey, importData.Stats).Err(); err != nil {
			return fmt.Errorf("failed to import stats: %w", err)
		}
	}

	return nil
}
