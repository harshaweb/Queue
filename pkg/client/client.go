package client

import (
	"context"
	"fmt"
	"time"

	"github.com/queue-system/redis-queue/pkg/queue"
	"github.com/queue-system/redis-queue/pkg/redis"
)

// Client is the main client for the queue system
type Client struct {
	queue         queue.QueueManager
	config        *Config
	middleware    []Middleware
	serializer    Serializer
	traceProvider TraceProvider
}

// Config contains client configuration
type Config struct {
	RedisConfig    *redis.Config
	DefaultTimeout time.Duration
	DefaultRetries int
	DefaultBackoff queue.BackoffPolicy
	EnableTracing  bool
	EnableMetrics  bool
	CircuitBreaker *CircuitBreakerConfig
}

// CircuitBreakerConfig contains circuit breaker settings
type CircuitBreakerConfig struct {
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts map[string]uint64) bool
	OnStateChange func(name string, from string, to string)
}

// DefaultConfig returns a default client configuration
func DefaultConfig() *Config {
	return &Config{
		RedisConfig:    redis.DefaultConfig(),
		DefaultTimeout: 30 * time.Second,
		DefaultRetries: 3,
		DefaultBackoff: queue.BackoffPolicy{
			Type:        queue.BackoffExponential,
			InitialWait: 1 * time.Second,
			MaxWait:     30 * time.Second,
			Multiplier:  2.0,
		},
		EnableTracing: true,
		EnableMetrics: true,
	}
}

// NewClient creates a new queue client
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create Redis client
	redisClient, err := redis.NewClient(config.RedisConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %w", err)
	}

	// Create queue manager
	queueManager := redis.NewStreamQueue(redisClient, redis.DefaultStreamQueueConfig())

	client := &Client{
		queue:      queueManager,
		config:     config,
		serializer: &JSONSerializer{},
	}

	// Add default middleware
	if config.EnableMetrics {
		client.middleware = append(client.middleware, NewMetricsMiddleware())
	}

	if config.EnableTracing {
		client.middleware = append(client.middleware, NewTracingMiddleware())
	}

	if config.CircuitBreaker != nil {
		client.middleware = append(client.middleware, NewCircuitBreakerMiddleware(config.CircuitBreaker))
	}

	return client, nil
}

// Enqueue adds a message to a queue
func (c *Client) Enqueue(ctx context.Context, queueName string, payload interface{}, opts ...EnqueueOption) (*EnqueueResult, error) {
	// Apply middleware
	ctx = c.applyMiddleware(ctx, "enqueue", queueName)

	// Build options
	options := &EnqueueOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Serialize payload
	data, err := c.serializer.Serialize(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize payload: %w", err)
	}

	// Create message
	message := queue.NewMessage(data, options.Headers)

	// Convert options
	msgOpts := &queue.MessageOptions{
		Priority:   options.Priority,
		ScheduleAt: options.ScheduleAt,
		Delay:      options.Delay,
		TTL:        options.TTL,
		Headers:    options.Headers,
	}

	// Enqueue message
	err = c.queue.Enqueue(ctx, queueName, message, msgOpts)
	if err != nil {
		return nil, err
	}

	return &EnqueueResult{
		MessageID: message.ID,
		QueueName: queueName,
	}, nil
}

// Consume starts consuming messages from a queue
func (c *Client) Consume(ctx context.Context, queueName string, handler MessageHandler, opts ...ConsumeOption) error {
	// Apply middleware
	ctx = c.applyMiddleware(ctx, "consume", queueName)

	// Build options
	options := &ConsumeOptions{
		BatchSize:         1,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        c.config.DefaultRetries,
		BackoffPolicy:     c.config.DefaultBackoff,
		LongPoll:          true,
		LongPollTimeout:   30 * time.Second,
	}

	for _, opt := range opts {
		opt(options)
	}

	// Convert options
	queueOpts := &queue.ConsumeOptions{
		VisibilityTimeout: options.VisibilityTimeout,
		MaxRetries:        options.MaxRetries,
		BackoffPolicy:     options.BackoffPolicy,
		BatchSize:         options.BatchSize,
		LongPoll:          options.LongPoll,
		LongPollTimeout:   options.LongPollTimeout,
	}

	// Start consuming
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := c.processMessages(ctx, queueName, handler, queueOpts); err != nil {
				if options.ErrorHandler != nil {
					if !options.ErrorHandler(err) {
						return err
					}
				} else {
					return err
				}
			}
		}
	}
}

func (c *Client) processMessages(ctx context.Context, queueName string, handler MessageHandler, opts *queue.ConsumeOptions) error {
	if opts.BatchSize == 1 {
		// Single message processing
		message, err := c.queue.Dequeue(ctx, queueName, opts)
		if err != nil {
			return err
		}

		if message == nil {
			return nil // No messages available
		}

		return c.handleMessage(ctx, queueName, message, handler)
	} else {
		// Batch processing
		messages, err := c.queue.BatchDequeue(ctx, queueName, opts.BatchSize, opts)
		if err != nil {
			return err
		}

		for _, message := range messages {
			if err := c.handleMessage(ctx, queueName, message, handler); err != nil {
				return err
			}
		}

		return nil
	}
}

func (c *Client) handleMessage(ctx context.Context, queueName string, message *queue.Message, handler MessageHandler) error {
	// Deserialize payload
	var payload interface{}
	if err := c.serializer.Deserialize(message.Payload, &payload); err != nil {
		return fmt.Errorf("failed to deserialize message: %w", err)
	}

	// Create message context
	msgCtx := &MessageContext{
		MessageID:  message.ID,
		QueueName:  queueName,
		Headers:    message.Headers,
		CreatedAt:  message.CreatedAt,
		RetryCount: message.RetryCount,
		client:     c,
	}

	// Handle message
	result := handler(ctx, payload, msgCtx)

	// Process result
	switch result.Action {
	case ActionAck:
		return c.queue.Ack(ctx, queueName, message.ID)
	case ActionNack:
		return c.queue.Nack(ctx, queueName, message.ID, result.Requeue)
	case ActionRequeue:
		return c.queue.Requeue(ctx, queueName, message.ID, result.Delay)
	case ActionMoveToLast:
		return c.queue.MoveToLast(ctx, queueName, message.ID)
	case ActionSkip:
		return c.queue.Skip(ctx, queueName, message.ID, result.SkipDuration)
	case ActionDeadLetter:
		return c.queue.MoveToDeadLetter(ctx, queueName, message.ID)
	default:
		return c.queue.Ack(ctx, queueName, message.ID)
	}
}

// BatchEnqueue adds multiple messages to a queue
func (c *Client) BatchEnqueue(ctx context.Context, queueName string, payloads []interface{}, opts ...EnqueueOption) (*BatchEnqueueResult, error) {
	// Apply middleware
	ctx = c.applyMiddleware(ctx, "batch_enqueue", queueName)

	// Build options
	options := &EnqueueOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Create messages
	var messages []*queue.Message
	var messageIDs []string

	for _, payload := range payloads {
		data, err := c.serializer.Serialize(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize payload: %w", err)
		}

		message := queue.NewMessage(data, options.Headers)
		messages = append(messages, message)
		messageIDs = append(messageIDs, message.ID)
	}

	// Convert options
	msgOpts := &queue.MessageOptions{
		Priority:   options.Priority,
		ScheduleAt: options.ScheduleAt,
		Delay:      options.Delay,
		TTL:        options.TTL,
		Headers:    options.Headers,
	}

	// Batch enqueue
	err := c.queue.BatchEnqueue(ctx, queueName, messages, msgOpts)
	if err != nil {
		return nil, err
	}

	return &BatchEnqueueResult{
		MessageIDs: messageIDs,
		QueueName:  queueName,
		Count:      len(messageIDs),
	}, nil
}

// CreateQueue creates a new queue
func (c *Client) CreateQueue(ctx context.Context, config *QueueConfig) error {
	queueConfig := &queue.QueueConfig{
		Name:              config.Name,
		VisibilityTimeout: config.VisibilityTimeout,
		MaxRetries:        config.MaxRetries,
		DeadLetterEnabled: config.DeadLetterEnabled,
		DeadLetterQueue:   config.DeadLetterQueue,
		BackoffPolicy:     config.BackoffPolicy,
		MaxConsumers:      config.MaxConsumers,
		MessageRetention:  config.MessageRetention,
	}

	return c.queue.CreateQueue(ctx, queueConfig)
}

// GetQueueStats returns queue statistics
func (c *Client) GetQueueStats(ctx context.Context, queueName string) (*QueueStats, error) {
	stats, err := c.queue.GetQueueStats(ctx, queueName)
	if err != nil {
		return nil, err
	}

	return &QueueStats{
		Name:           stats.Name,
		Length:         stats.Length,
		InFlight:       stats.InFlight,
		Processed:      stats.Processed,
		Failed:         stats.Failed,
		ConsumerCount:  stats.ConsumerCount,
		LastActivity:   stats.LastActivity,
		DeadLetterSize: stats.DeadLetterSize,
	}, nil
}

// ListQueues returns all queue names
func (c *Client) ListQueues(ctx context.Context) ([]string, error) {
	return c.queue.ListQueues(ctx)
}

// Close closes the client and cleans up resources
func (c *Client) Close() error {
	return c.queue.Close()
}

func (c *Client) applyMiddleware(ctx context.Context, operation, queueName string) context.Context {
	for _, mw := range c.middleware {
		ctx = mw.Before(ctx, operation, queueName)
	}
	return ctx
}

// Admin operations

// PauseQueue pauses message consumption
func (c *Client) PauseQueue(ctx context.Context, queueName string) error {
	return c.queue.PauseQueue(ctx, queueName)
}

// ResumeQueue resumes message consumption
func (c *Client) ResumeQueue(ctx context.Context, queueName string) error {
	return c.queue.ResumeQueue(ctx, queueName)
}

// MoveMessage moves a message to another queue or position
func (c *Client) MoveMessage(ctx context.Context, queueName, messageID string, action MoveAction, target string) error {
	switch action {
	case MoveActionToLast:
		return c.queue.MoveToLast(ctx, queueName, messageID)
	case MoveActionToDeadLetter:
		return c.queue.MoveToDeadLetter(ctx, queueName, messageID)
	default:
		return fmt.Errorf("unsupported move action: %v", action)
	}
}

// SkipMessage temporarily skips a message
func (c *Client) SkipMessage(ctx context.Context, queueName, messageID string, duration time.Duration) error {
	return c.queue.Skip(ctx, queueName, messageID, &duration)
}

// Use adds middleware to the client
func (c *Client) Use(middleware Middleware) {
	c.middleware = append(c.middleware, middleware)
}

// Ack returns a message result indicating successful acknowledgment
func (c *Client) Ack(ctx context.Context, queueName, messageID string) error {
	return c.queue.Ack(ctx, queueName, messageID)
}

// Nack returns a message result indicating negative acknowledgment
func (c *Client) Nack(ctx context.Context, queueName, messageID string, requeue bool) error {
	return c.queue.Nack(ctx, queueName, messageID, requeue)
}

// Dequeue retrieves messages from a queue without consuming them permanently
func (c *Client) Dequeue(ctx context.Context, queueName string, batchSize int) ([]*queue.Message, error) {
	opts := &queue.ConsumeOptions{
		BatchSize:         batchSize,
		VisibilityTimeout: c.config.DefaultTimeout,
	}

	if batchSize == 1 {
		message, err := c.queue.Dequeue(ctx, queueName, opts)
		if err != nil {
			return nil, err
		}
		if message == nil {
			return []*queue.Message{}, nil
		}
		return []*queue.Message{message}, nil
	}

	// For batch dequeue, we'll need to call multiple times or implement batch in interface
	messages := make([]*queue.Message, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		message, err := c.queue.Dequeue(ctx, queueName, opts)
		if err != nil {
			return messages, err
		}
		if message == nil {
			break // No more messages
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// Peek retrieves messages without removing them from the queue
func (c *Client) Peek(ctx context.Context, queueName string, count int) ([]*queue.Message, error) {
	// Since Peek is not in the base interface, we'll implement it by dequeuing and re-enqueuing
	// This is not ideal but provides the functionality
	messages, err := c.Dequeue(ctx, queueName, count)
	if err != nil {
		return nil, err
	}

	// Re-enqueue the messages to "peek" without consuming
	for _, msg := range messages {
		err = c.queue.Nack(ctx, queueName, msg.ID, true) // Requeue
		if err != nil {
			// Log error but continue
			continue
		}
	}

	return messages, nil
}
