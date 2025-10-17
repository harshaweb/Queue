package pkg

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// Consumer represents a queue consumer with advanced capabilities
type Consumer struct {
	queue    *Queue
	id       string
	isActive bool
	metrics  *ConsumerMetrics
	mu       sync.RWMutex
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// ConsumerMetrics tracks consumer performance
type ConsumerMetrics struct {
	ProcessedCount    int64         `json:"processed_count"`
	ErrorCount        int64         `json:"error_count"`
	AverageProcessing time.Duration `json:"average_processing"`
	LastActivity      time.Time     `json:"last_activity"`
	StartTime         time.Time     `json:"start_time"`
	mu                sync.RWMutex
}

// NewConsumer creates a new consumer
func (q *Queue) NewConsumer(id string) *Consumer {
	consumer := &Consumer{
		queue:   q,
		id:      id,
		metrics: &ConsumerMetrics{StartTime: time.Now()},
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}

	q.mu.Lock()
	q.consumers[id] = consumer
	q.mu.Unlock()

	return consumer
}

// Start begins consuming messages
func (c *Consumer) Start(handler HandlerFunc, options ...*ConsumeOptions) error {
	c.mu.Lock()
	if c.isActive {
		c.mu.Unlock()
		return fmt.Errorf("consumer %s is already active", c.id)
	}
	c.isActive = true
	c.mu.Unlock()

	go c.consumeLoop(handler, options...)
	return nil
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isActive {
		return fmt.Errorf("consumer %s is not active", c.id)
	}

	close(c.stopCh)
	<-c.doneCh
	c.isActive = false

	// Remove from queue's consumer list
	c.queue.mu.Lock()
	delete(c.queue.consumers, c.id)
	c.queue.mu.Unlock()

	return nil
}

// IsActive returns whether the consumer is running
func (c *Consumer) IsActive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isActive
}

// GetMetrics returns consumer metrics
func (c *Consumer) GetMetrics() *ConsumerMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	return &ConsumerMetrics{
		ProcessedCount:    c.metrics.ProcessedCount,
		ErrorCount:        c.metrics.ErrorCount,
		AverageProcessing: c.metrics.AverageProcessing,
		LastActivity:      c.metrics.LastActivity,
		StartTime:         c.metrics.StartTime,
	}
}

// consumeLoop is the main consumption loop
func (c *Consumer) consumeLoop(handler HandlerFunc, options ...*ConsumeOptions) {
	defer close(c.doneCh)

	opts := &ConsumeOptions{
		BatchSize:         c.queue.config.BatchSize,
		MaxConcurrency:    1,
		VisibilityTimeout: c.queue.config.VisibilityTimeout,
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

	ctx := context.Background()
	sem := make(chan struct{}, opts.MaxConcurrency)

	for {
		select {
		case <-c.stopCh:
			return
		default:
			// Read messages from stream
			result := c.queue.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    c.queue.config.ConsumerGroup,
				Consumer: c.id,
				Streams:  []string{c.queue.name, ">"},
				Count:    int64(opts.BatchSize),
				Block:    100 * time.Millisecond,
			})

			if result.Err() != nil {
				if result.Err() == redis.Nil {
					continue
				}
				// Log error and continue
				time.Sleep(time.Second)
				continue
			}

			streams := result.Val()
			if len(streams) == 0 || len(streams[0].Messages) == 0 {
				continue
			}

			// Process messages concurrently
			var wg sync.WaitGroup
			for _, streamMsg := range streams[0].Messages {
				select {
				case <-c.stopCh:
					return
				case sem <- struct{}{}:
					wg.Add(1)

					go func(sm redis.XMessage) {
						defer wg.Done()
						defer func() { <-sem }()

						start := time.Now()

						msg, err := c.queue.parseMessage(sm)
						if err != nil {
							c.recordError()
							return
						}

						// Apply filter if provided
						if opts.FilterFunc != nil && !opts.FilterFunc(msg) {
							return
						}

						// Check if message should be processed
						if !c.queue.shouldProcessMessage(msg) {
							return
						}

						// Process message
						err = c.processMessage(ctx, msg, handler, opts)

						// Record metrics
						c.recordProcessing(time.Since(start), err == nil)

					}(streamMsg)
				}
			}

			wg.Wait()
		}
	}
}

// processMessage handles individual message processing
func (c *Consumer) processMessage(ctx context.Context, msg *Message, handler HandlerFunc, opts *ConsumeOptions) error {
	msg.Attempts++

	err := handler(ctx, msg)

	if opts.AutoAck {
		if err != nil {
			return c.queue.handleError(msg, err, opts)
		} else {
			return c.queue.Ack(msg.StreamID)
		}
	}

	return err
}

// recordProcessing updates metrics
func (c *Consumer) recordProcessing(duration time.Duration, success bool) {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()

	c.metrics.LastActivity = time.Now()

	if success {
		c.metrics.ProcessedCount++
		// Update average processing time
		if c.metrics.ProcessedCount == 1 {
			c.metrics.AverageProcessing = duration
		} else {
			c.metrics.AverageProcessing = time.Duration(
				(int64(c.metrics.AverageProcessing)*c.metrics.ProcessedCount + int64(duration)) /
					(c.metrics.ProcessedCount + 1),
			)
		}
	}
}

// recordError updates error metrics
func (c *Consumer) recordError() {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()

	c.metrics.ErrorCount++
	c.metrics.LastActivity = time.Now()
}

// ConsumerGroup manages multiple consumers
type ConsumerGroup struct {
	queue     *Queue
	consumers map[string]*Consumer
	mu        sync.RWMutex
}

// NewConsumerGroup creates a new consumer group
func (q *Queue) NewConsumerGroup() *ConsumerGroup {
	return &ConsumerGroup{
		queue:     q,
		consumers: make(map[string]*Consumer),
	}
}

// AddConsumer adds a consumer to the group
func (cg *ConsumerGroup) AddConsumer(id string) *Consumer {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	consumer := cg.queue.NewConsumer(id)
	cg.consumers[id] = consumer
	return consumer
}

// RemoveConsumer removes a consumer from the group
func (cg *ConsumerGroup) RemoveConsumer(id string) error {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	consumer, exists := cg.consumers[id]
	if !exists {
		return fmt.Errorf("consumer %s not found", id)
	}

	if consumer.IsActive() {
		if err := consumer.Stop(); err != nil {
			return err
		}
	}

	delete(cg.consumers, id)
	return nil
}

// StartAll starts all consumers in the group
func (cg *ConsumerGroup) StartAll(handler HandlerFunc, options ...*ConsumeOptions) error {
	cg.mu.RLock()
	defer cg.mu.RUnlock()

	for _, consumer := range cg.consumers {
		if err := consumer.Start(handler, options...); err != nil {
			return err
		}
	}

	return nil
}

// StopAll stops all consumers in the group
func (cg *ConsumerGroup) StopAll() error {
	cg.mu.RLock()
	defer cg.mu.RUnlock()

	for _, consumer := range cg.consumers {
		if consumer.IsActive() {
			if err := consumer.Stop(); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetActiveConsumers returns the number of active consumers
func (cg *ConsumerGroup) GetActiveConsumers() int {
	cg.mu.RLock()
	defer cg.mu.RUnlock()

	count := 0
	for _, consumer := range cg.consumers {
		if consumer.IsActive() {
			count++
		}
	}

	return count
}

// GetGroupMetrics returns aggregated metrics for the group
func (cg *ConsumerGroup) GetGroupMetrics() map[string]*ConsumerMetrics {
	cg.mu.RLock()
	defer cg.mu.RUnlock()

	metrics := make(map[string]*ConsumerMetrics)
	for id, consumer := range cg.consumers {
		metrics[id] = consumer.GetMetrics()
	}

	return metrics
}
