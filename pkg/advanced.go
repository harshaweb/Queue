package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// PriorityQueue implements priority-based message processing
type PriorityQueue struct {
	*Queue
	priorityLevels []string
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue(name string, config *Config, priorities []int) (*PriorityQueue, error) {
	baseQueue, err := NewQueue(name, config)
	if err != nil {
		return nil, err
	}

	// Sort priorities in descending order (highest first)
	sort.Sort(sort.Reverse(sort.IntSlice(priorities)))

	var priorityLevels []string
	for _, p := range priorities {
		priorityLevels = append(priorityLevels, fmt.Sprintf("%s:p%d", name, p))
	}

	pq := &PriorityQueue{
		Queue:          baseQueue,
		priorityLevels: priorityLevels,
	}

	// Initialize priority level streams
	ctx := context.Background()
	for _, level := range priorityLevels {
		err := baseQueue.rdb.XGroupCreateMkStream(ctx, level, config.ConsumerGroup, "$").Err()
		if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
			return nil, fmt.Errorf("failed to create priority level %s: %v", level, err)
		}
	}

	return pq, nil
}

// Send sends a message to the appropriate priority level
func (pq *PriorityQueue) Send(payload map[string]interface{}, options ...*SendOptions) (string, error) {
	msg := &Message{Payload: payload}
	return pq.SendMessage(msg, options...)
}

// SendMessage sends a message with priority routing
func (pq *PriorityQueue) SendMessage(msg *Message, options ...*SendOptions) (string, error) {
	if len(options) > 0 && options[0] != nil && options[0].Priority > 0 {
		msg.Priority = options[0].Priority
	}

	// Determine target stream based on priority
	targetStream := pq.getStreamByPriority(msg.Priority)

	// Temporarily change queue name for sending
	originalName := pq.name
	pq.name = targetStream

	id, err := pq.Queue.SendMessage(msg, options...)

	// Restore original name
	pq.name = originalName

	return id, err
}

// getStreamByPriority returns the appropriate stream for a priority level
func (pq *PriorityQueue) getStreamByPriority(priority int) string {
	if priority <= 0 {
		priority = 1
	}

	targetLevel := fmt.Sprintf("%s:p%d", pq.name, priority)

	// Check if this priority level exists
	for _, level := range pq.priorityLevels {
		if level == targetLevel {
			return level
		}
	}

	// If exact priority doesn't exist, use the closest lower priority
	for _, level := range pq.priorityLevels {
		return level // Return the highest priority level as fallback
	}

	return pq.name // Fallback to main stream
}

// ConsumeByPriority processes messages in priority order
func (pq *PriorityQueue) ConsumeByPriority(handler HandlerFunc, options ...*ConsumeOptions) error {
	ctx := context.Background()

	opts := &ConsumeOptions{
		BatchSize:         pq.config.BatchSize,
		MaxConcurrency:    1,
		VisibilityTimeout: pq.config.VisibilityTimeout,
		AutoAck:           true,
	}

	if len(options) > 0 && options[0] != nil {
		if options[0].BatchSize > 0 {
			opts.BatchSize = options[0].BatchSize
		}
		if options[0].MaxConcurrency > 0 {
			opts.MaxConcurrency = options[0].MaxConcurrency
		}
		if options[0].VisibilityTimeout > 0 {
			opts.VisibilityTimeout = options[0].VisibilityTimeout
		}
		opts.AutoAck = options[0].AutoAck
		opts.RetryPolicy = options[0].RetryPolicy
		opts.FilterFunc = options[0].FilterFunc
	}

	for {
		// Try to read from each priority level in order
		for _, level := range pq.priorityLevels {
			result := pq.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    pq.config.ConsumerGroup,
				Consumer: pq.config.ConsumerName,
				Streams:  []string{level, ">"},
				Count:    int64(opts.BatchSize),
				Block:    10 * time.Millisecond, // Short block to check next priority
			})

			if result.Err() == nil && len(result.Val()) > 0 && len(result.Val()[0].Messages) > 0 {
				// Process messages from this priority level
				streams := result.Val()
				for _, streamMsg := range streams[0].Messages {
					msg, err := pq.parseMessage(streamMsg)
					if err != nil {
						continue
					}

					// Apply filter if provided
					if opts.FilterFunc != nil && !opts.FilterFunc(msg) {
						continue
					}

					// Check if message should be processed
					if !pq.shouldProcessMessage(msg) {
						continue
					}

					// Process message
					err = pq.processMessage(ctx, msg, handler, opts)
					if err != nil && pq.metrics != nil {
						pq.metrics.IncrementErrors("process")
					}
				}
				break // Found messages at this priority, don't check lower priorities
			}
		}

		// Small delay to prevent busy waiting
		time.Sleep(50 * time.Millisecond)
	}
}

// DelayedQueue implements delayed message processing
type DelayedQueue struct {
	*Queue
	delayedStream string
	scheduler     *DelayScheduler
}

// NewDelayedQueue creates a new delayed queue
func NewDelayedQueue(name string, config *Config) (*DelayedQueue, error) {
	baseQueue, err := NewQueue(name, config)
	if err != nil {
		return nil, err
	}

	delayedStream := name + ":delayed"

	dq := &DelayedQueue{
		Queue:         baseQueue,
		delayedStream: delayedStream,
	}

	// Initialize delayed stream
	ctx := context.Background()
	err = baseQueue.rdb.XGroupCreateMkStream(ctx, delayedStream, config.ConsumerGroup, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return nil, fmt.Errorf("failed to create delayed stream: %v", err)
	}

	// Initialize scheduler - get Redis client from base queue
	redisClient := baseQueue.GetRedisClient()
	if redisClient == nil {
		return nil, fmt.Errorf("failed to get Redis client for delay scheduler")
	}
	dq.scheduler = NewDelayScheduler(redisClient, delayedStream, name)

	return dq, nil
}

// SendDelayed sends a message with a delay
func (dq *DelayedQueue) SendDelayed(payload map[string]interface{}, delay time.Duration, options ...*SendOptions) (string, error) {
	msg := &Message{
		Payload: payload,
		Delay:   delay,
	}

	return dq.SendDelayedMessage(msg, options...)
}

// SendDelayedMessage sends a delayed message
func (dq *DelayedQueue) SendDelayedMessage(msg *Message, options ...*SendOptions) (string, error) {
	// Apply options
	if len(options) > 0 && options[0] != nil {
		opts := options[0]
		if opts.Priority > 0 {
			msg.Priority = opts.Priority
		}
		if opts.Delay > 0 {
			msg.Delay = opts.Delay
		}
		if opts.Deadline != nil {
			msg.Deadline = opts.Deadline
		}
		if opts.Headers != nil {
			msg.Headers = opts.Headers
		}
		if opts.MaxRetries > 0 {
			msg.MaxRetries = opts.MaxRetries
		}
	}

	// Set defaults
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.MaxRetries == 0 {
		msg.MaxRetries = dq.config.MaxRetries
	}
	msg.CreatedAt = time.Now()

	// Calculate execution time
	executeAt := time.Now().Add(msg.Delay)

	// Store in delayed stream with execution time as score
	return dq.scheduler.ScheduleMessage(msg, executeAt)
}

// Start begins processing delayed messages
func (dq *DelayedQueue) Start() error {
	return dq.scheduler.Start()
}

// Stop stops processing delayed messages
func (dq *DelayedQueue) Stop() error {
	return dq.scheduler.Stop()
}

// DelayScheduler manages delayed message scheduling
type DelayScheduler struct {
	rdb           *redis.Client
	delayedStream string
	targetStream  string
	stopCh        chan struct{}
	isRunning     bool
}

// NewDelayScheduler creates a new delay scheduler
func NewDelayScheduler(rdb *redis.Client, delayedStream, targetStream string) *DelayScheduler {
	return &DelayScheduler{
		rdb:           rdb,
		delayedStream: delayedStream,
		targetStream:  targetStream,
		stopCh:        make(chan struct{}),
	}
}

// ScheduleMessage schedules a message for future execution
func (ds *DelayScheduler) ScheduleMessage(msg *Message, executeAt time.Time) (string, error) {
	ctx := context.Background()

	// Serialize message
	msgData, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to serialize message: %v", err)
	}

	// Store in sorted set with execution time as score
	result := ds.rdb.ZAdd(ctx, ds.delayedStream, &redis.Z{
		Score:  float64(executeAt.Unix()),
		Member: msgData,
	})

	if result.Err() != nil {
		return "", fmt.Errorf("failed to schedule message: %v", result.Err())
	}

	return msg.ID, nil
}

// Start begins processing scheduled messages
func (ds *DelayScheduler) Start() error {
	if ds.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	ds.isRunning = true

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ds.stopCh:
				ds.isRunning = false
				return
			case <-ticker.C:
				ds.processScheduledMessages()
			}
		}
	}()

	return nil
}

// Stop stops the scheduler
func (ds *DelayScheduler) Stop() error {
	if !ds.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	close(ds.stopCh)
	return nil
}

// processScheduledMessages moves ready messages to the main queue
func (ds *DelayScheduler) processScheduledMessages() {
	ctx := context.Background()
	now := float64(time.Now().Unix())

	// Get messages ready for execution
	result := ds.rdb.ZRangeByScore(ctx, ds.delayedStream, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("%.0f", now),
		Count: 100, // Process up to 100 messages at a time
	})

	if result.Err() != nil || len(result.Val()) == 0 {
		return
	}

	// Process each ready message
	for _, msgData := range result.Val() {
		var msg Message
		if err := json.Unmarshal([]byte(msgData), &msg); err != nil {
			continue
		}

		// Move to main queue
		streamData := map[string]interface{}{
			"id":          msg.ID,
			"payload":     ds.serializePayload(msg.Payload),
			"priority":    msg.Priority,
			"max_retries": msg.MaxRetries,
			"created_at":  msg.CreatedAt.Unix(),
			"retry_count": msg.RetryCount,
		}

		if msg.Headers != nil {
			streamData["headers"] = ds.serializeHeaders(msg.Headers)
		}

		if msg.Deadline != nil {
			streamData["deadline"] = msg.Deadline.Unix()
		}

		// Add to target stream
		addResult := ds.rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: ds.targetStream,
			Values: streamData,
		})

		if addResult.Err() == nil {
			// Remove from delayed queue
			ds.rdb.ZRem(ctx, ds.delayedStream, msgData)
		}
	}
}

// Helper methods
func (ds *DelayScheduler) serializePayload(payload map[string]interface{}) string {
	if payload == nil {
		return "{}"
	}
	data, _ := json.Marshal(payload)
	return string(data)
}

func (ds *DelayScheduler) serializeHeaders(headers map[string]string) string {
	if headers == nil {
		return "{}"
	}
	data, _ := json.Marshal(headers)
	return string(data)
}

// BatchQueue provides high-throughput batch processing
type BatchQueue struct {
	*Queue
	batchProcessor *BatchProcessor
}

// NewBatchQueue creates a new batch queue
func NewBatchQueue(name string, config *Config) (*BatchQueue, error) {
	baseQueue, err := NewQueue(name, config)
	if err != nil {
		return nil, err
	}

	bq := &BatchQueue{
		Queue:          baseQueue,
		batchProcessor: NewBatchProcessor(baseQueue, config.BatchSize),
	}

	return bq, nil
}

// ConsumeBatch processes messages in batches
func (bq *BatchQueue) ConsumeBatch(handler BatchHandlerFunc, options ...*ConsumeOptions) error {
	return bq.batchProcessor.Start(handler, options...)
}

// BatchProcessor handles batch message processing
type BatchProcessor struct {
	queue     *Queue
	batchSize int
	stopCh    chan struct{}
	isRunning bool
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(queue *Queue, batchSize int) *BatchProcessor {
	return &BatchProcessor{
		queue:     queue,
		batchSize: batchSize,
		stopCh:    make(chan struct{}),
	}
}

// Start begins batch processing
func (bp *BatchProcessor) Start(handler BatchHandlerFunc, options ...*ConsumeOptions) error {
	if bp.isRunning {
		return fmt.Errorf("batch processor is already running")
	}

	bp.isRunning = true

	opts := &ConsumeOptions{
		BatchSize:         bp.batchSize,
		MaxConcurrency:    1,
		VisibilityTimeout: bp.queue.config.VisibilityTimeout,
		AutoAck:           true,
	}

	if len(options) > 0 && options[0] != nil {
		if options[0].BatchSize > 0 {
			opts.BatchSize = options[0].BatchSize
		}
		if options[0].MaxConcurrency > 0 {
			opts.MaxConcurrency = options[0].MaxConcurrency
		}
		if options[0].VisibilityTimeout > 0 {
			opts.VisibilityTimeout = options[0].VisibilityTimeout
		}
		opts.AutoAck = options[0].AutoAck
		opts.RetryPolicy = options[0].RetryPolicy
		opts.FilterFunc = options[0].FilterFunc
	}

	go func() {
		defer func() { bp.isRunning = false }()

		ctx := context.Background()

		for {
			select {
			case <-bp.stopCh:
				return
			default:
				// Read batch of messages
				result := bp.queue.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:    bp.queue.config.ConsumerGroup,
					Consumer: bp.queue.config.ConsumerName,
					Streams:  []string{bp.queue.name, ">"},
					Count:    int64(opts.BatchSize),
					Block:    100 * time.Millisecond,
				})

				if result.Err() != nil {
					if result.Err() == redis.Nil {
						continue
					}
					time.Sleep(time.Second)
					continue
				}

				streams := result.Val()
				if len(streams) == 0 || len(streams[0].Messages) == 0 {
					continue
				}

				// Parse messages
				var messages []*Message
				var streamIDs []string

				for _, streamMsg := range streams[0].Messages {
					msg, err := bp.queue.parseMessage(streamMsg)
					if err != nil {
						continue
					}

					// Apply filter if provided
					if opts.FilterFunc != nil && !opts.FilterFunc(msg) {
						continue
					}

					// Check if message should be processed
					if !bp.queue.shouldProcessMessage(msg) {
						continue
					}

					messages = append(messages, msg)
					streamIDs = append(streamIDs, streamMsg.ID)
				}

				if len(messages) > 0 {
					// Process batch
					err := handler(ctx, messages)

					if opts.AutoAck {
						if err != nil {
							// Handle batch error (could retry individual messages)
							if bp.queue.metrics != nil {
								bp.queue.metrics.IncrementErrors("batch_process")
							}
						} else {
							// Acknowledge all messages in the batch
							for _, streamID := range streamIDs {
								bp.queue.Ack(streamID)
							}
						}
					}
				}
			}
		}
	}()

	return nil
}

// Stop stops batch processing
func (bp *BatchProcessor) Stop() error {
	if !bp.isRunning {
		return fmt.Errorf("batch processor is not running")
	}

	close(bp.stopCh)
	return nil
}
