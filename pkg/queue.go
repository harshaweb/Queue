package pkg

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// Queue represents a high-performance, feature-rich queue
type Queue struct {
	rdb    redis.Cmdable // Support both single and cluster clients
	client *redis.Client // Keep reference for single client operations
	name   string
	config *Config
	mu     sync.RWMutex

	// Internal state
	consumers map[string]*Consumer
	metrics   *Metrics
	dlq       *DeadLetterQueue

	// Advanced features
	circuitBreaker *CircuitBreaker
	rateLimiter    interface{} // Can be *TokenBucket or *SlidingWindow
	encryption     *MessageEncryption
	scheduler      *MessageScheduler
	tracing        *MessageTracing
}

// Config holds comprehensive queue configuration
type Config struct {
	// Redis connection
	RedisAddress    string `json:"redis_address"`
	RedisPassword   string `json:"redis_password"`
	RedisDB         int    `json:"redis_db"`
	RedisUsername   string `json:"redis_username"`    // Redis 6.0+ ACL username
	RedisClientName string `json:"redis_client_name"` // Client identification

	// Redis Security
	EnableTLS     bool   `json:"enable_tls"`
	TLSInsecure   bool   `json:"tls_insecure"`     // Skip TLS verification
	TLSCertFile   string `json:"tls_cert_file"`    // Client certificate
	TLSKeyFile    string `json:"tls_key_file"`     // Client private key
	TLSCACertFile string `json:"tls_ca_cert_file"` // CA certificate

	// Redis Cluster/Sentinel
	RedisCluster      bool     `json:"redis_cluster"`      // Enable cluster mode
	ClusterAddresses  []string `json:"cluster_addresses"`  // Cluster node addresses
	SentinelEnabled   bool     `json:"sentinel_enabled"`   // Enable Sentinel
	SentinelAddresses []string `json:"sentinel_addresses"` // Sentinel addresses
	SentinelMaster    string   `json:"sentinel_master"`    // Master name for Sentinel
	SentinelPassword  string   `json:"sentinel_password"`  // Sentinel password

	// Connection pooling and timeouts
	PoolSize      int           `json:"pool_size"`
	MinIdleConns  int           `json:"min_idle_conns"`
	MaxConnAge    time.Duration `json:"max_conn_age"`
	PoolTimeout   time.Duration `json:"pool_timeout"`
	IdleTimeout   time.Duration `json:"idle_timeout"`
	IdleCheckFreq time.Duration `json:"idle_check_freq"`

	// Network timeouts
	DialTimeout  time.Duration `json:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`

	// Queue behavior
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
	VisibilityTimeout time.Duration `json:"visibility_timeout"`
	MessageTTL        time.Duration `json:"message_ttl"`

	// Consumer settings
	ConsumerGroup string `json:"consumer_group"`
	ConsumerName  string `json:"consumer_name"`
	BatchSize     int    `json:"batch_size"`
	PrefetchCount int    `json:"prefetch_count"`

	// Dead letter queue
	EnableDLQ bool   `json:"enable_dlq"`
	DLQName   string `json:"dlq_name"`

	// Performance
	EnableMetrics     bool `json:"enable_metrics"`
	EnableCompression bool `json:"enable_compression"`

	// Advanced features
	CircuitBreakerConfig CircuitBreakerConfig `json:"circuit_breaker"`
	RateLimiterConfig    RateLimiterConfig    `json:"rate_limiter"`
	EncryptionConfig     EncryptionConfig     `json:"encryption"`
	TracingConfig        TracingConfig        `json:"tracing"`
}

// DefaultConfig returns optimized default configuration
func DefaultConfig() *Config {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "default"
	}

	return &Config{
		RedisAddress:    "localhost:6379",
		RedisPassword:   "",
		RedisDB:         0,
		RedisUsername:   "", // Redis 6.0+ ACL support
		RedisClientName: fmt.Sprintf("queue-client-%s", hostname),

		// Security defaults
		EnableTLS:   false,
		TLSInsecure: false,

		// Clustering defaults
		RedisCluster:    false,
		SentinelEnabled: false,

		// Connection pool defaults
		PoolSize:      10,
		MinIdleConns:  2,
		MaxConnAge:    time.Hour,
		PoolTimeout:   time.Second * 4,
		IdleTimeout:   time.Minute * 5,
		IdleCheckFreq: time.Minute,

		// Network timeout defaults
		DialTimeout:  time.Second * 5,
		ReadTimeout:  time.Second * 3,
		WriteTimeout: time.Second * 3,

		// Queue behavior
		MaxRetries:        3,
		RetryDelay:        time.Second * 5,
		VisibilityTimeout: time.Minute * 5,
		MessageTTL:        time.Hour * 24,
		ConsumerGroup:     "default-group",
		ConsumerName:      fmt.Sprintf("%s-%s", hostname, uuid.New().String()[:8]),
		BatchSize:         10,
		PrefetchCount:     50,
		EnableDLQ:         true,
		DLQName:           "",
		EnableMetrics:     true,
		EnableCompression: false,
	}
}

// Message represents a queue message with full metadata
type Message struct {
	ID          string                 `json:"id"`
	Payload     map[string]interface{} `json:"payload"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Priority    int                    `json:"priority,omitempty"`
	Delay       time.Duration          `json:"delay,omitempty"`
	Deadline    *time.Time             `json:"deadline,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	CreatedAt   time.Time              `json:"created_at"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`

	// Internal fields
	StreamID string `json:"-"`
	Attempts int    `json:"-"`
}

// SendOptions provides advanced sending options
type SendOptions struct {
	Priority   int                    `json:"priority,omitempty"`
	Delay      time.Duration          `json:"delay,omitempty"`
	Deadline   *time.Time             `json:"deadline,omitempty"`
	Headers    map[string]string      `json:"headers,omitempty"`
	MaxRetries int                    `json:"max_retries,omitempty"`
	TTL        time.Duration          `json:"ttl,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ConsumeOptions provides advanced consumption options
type ConsumeOptions struct {
	BatchSize         int           `json:"batch_size,omitempty"`
	MaxConcurrency    int           `json:"max_concurrency,omitempty"`
	VisibilityTimeout time.Duration `json:"visibility_timeout,omitempty"`
	AutoAck           bool          `json:"auto_ack"`
	RetryPolicy       *RetryPolicy  `json:"retry_policy,omitempty"`
	FilterFunc        FilterFunc    `json:"-"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
	Jitter        bool          `json:"jitter"`
}

// HandlerFunc processes messages
type HandlerFunc func(ctx context.Context, msg *Message) error

// FilterFunc filters messages before processing
type FilterFunc func(msg *Message) bool

// BatchHandlerFunc processes multiple messages
type BatchHandlerFunc func(ctx context.Context, msgs []*Message) error

// QueueInfo provides comprehensive queue information
type QueueInfo struct {
	Name            string           `json:"name"`
	Length          int64            `json:"length"`
	ConsumerGroups  int              `json:"consumer_groups"`
	PendingMessages int64            `json:"pending_messages"`
	LastMessageID   string           `json:"last_message_id"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	Metrics         *MetricsSnapshot `json:"metrics,omitempty"`
}

// NewQueue creates a new feature-rich queue
func NewQueue(name string, config *Config) (*Queue, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Set default DLQ name if not provided
	if config.EnableDLQ && config.DLQName == "" {
		config.DLQName = name + ":dlq"
	}

	// Create Redis client with enhanced authentication and connection options
	var rdb redis.Cmdable
	var client *redis.Client

	if config.RedisCluster {
		// Redis Cluster configuration
		clusterClient, err := createRedisClusterClient(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create Redis cluster client: %v", err)
		}
		rdb = clusterClient
	} else if config.SentinelEnabled {
		// Redis Sentinel configuration
		sentinelClient, err := createRedisSentinelClient(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create Redis sentinel client: %v", err)
		}
		rdb = sentinelClient
		client = sentinelClient
	} else {
		// Single Redis instance
		singleClient, err := createRedisClient(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create Redis client: %v", err)
		}
		rdb = singleClient
		client = singleClient
	}

	q := &Queue{
		rdb:       rdb,
		client:    client,
		name:      name,
		config:    config,
		consumers: make(map[string]*Consumer),
	}

	// Initialize components
	if err := q.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize queue: %v", err)
	}

	return q, nil
}

// initialize sets up queue infrastructure
func (q *Queue) initialize() error {
	ctx := context.Background()

	// Create consumer group
	err := q.rdb.XGroupCreateMkStream(ctx, q.name, q.config.ConsumerGroup, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %v", err)
	}

	// Initialize dead letter queue
	if q.config.EnableDLQ {
		if q.client != nil {
			q.dlq = NewDeadLetterQueue(q.client, q.config.DLQName)
		} else {
			// For cluster clients, we need to create a separate client for DLQ operations
			singleClient, err := createRedisClient(q.config)
			if err != nil {
				return fmt.Errorf("failed to create DLQ client: %v", err)
			}
			q.dlq = NewDeadLetterQueue(singleClient, q.config.DLQName)
		}
		if err := q.dlq.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize DLQ: %v", err)
		}
	}

	// Initialize metrics
	if q.config.EnableMetrics {
		q.metrics = NewMetrics(q.name)
	}

	return nil
}

// Send sends a message with advanced options
func (q *Queue) Send(payload map[string]interface{}, options ...*SendOptions) (string, error) {
	return q.SendMessage(&Message{
		Payload: payload,
	}, options...)
}

// SendMessage sends a full message object
func (q *Queue) SendMessage(msg *Message, options ...*SendOptions) (string, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

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
		msg.MaxRetries = q.config.MaxRetries
	}
	msg.CreatedAt = time.Now()

	// Prepare Redis stream data
	streamData := map[string]interface{}{
		"id":          msg.ID,
		"payload":     q.serializePayload(msg.Payload),
		"priority":    msg.Priority,
		"max_retries": msg.MaxRetries,
		"created_at":  msg.CreatedAt.Unix(),
		"retry_count": msg.RetryCount,
	}

	if msg.Headers != nil {
		streamData["headers"] = q.serializeHeaders(msg.Headers)
	}

	if msg.Deadline != nil {
		streamData["deadline"] = msg.Deadline.Unix()
	}

	// Handle delay
	if msg.Delay > 0 {
		streamData["process_after"] = time.Now().Add(msg.Delay).Unix()
	}

	// Send to Redis stream
	ctx := context.Background()
	result := q.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: q.name,
		Values: streamData,
	})

	if result.Err() != nil {
		if q.metrics != nil {
			q.metrics.IncrementErrors("send")
		}
		return "", fmt.Errorf("failed to send message: %v", result.Err())
	}

	msg.StreamID = result.Val()

	if q.metrics != nil {
		q.metrics.IncrementSent()
	}

	return msg.ID, nil
}

// SendJSON sends JSON-serializable data
func (q *Queue) SendJSON(data interface{}, options ...*SendOptions) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}

	var payload map[string]interface{}
	err = json.Unmarshal(jsonData, &payload)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal to map: %v", err)
	}

	return q.Send(payload, options...)
}

// BatchSend sends multiple messages efficiently
func (q *Queue) BatchSend(messages []*Message, options ...*SendOptions) ([]string, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	ctx := context.Background()
	pipe := q.rdb.Pipeline()

	var ids []string
	for _, msg := range messages {
		// Apply options (same as SendMessage)
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

		if msg.ID == "" {
			msg.ID = uuid.New().String()
		}
		if msg.MaxRetries == 0 {
			msg.MaxRetries = q.config.MaxRetries
		}
		msg.CreatedAt = time.Now()

		streamData := map[string]interface{}{
			"id":          msg.ID,
			"payload":     q.serializePayload(msg.Payload),
			"priority":    msg.Priority,
			"max_retries": msg.MaxRetries,
			"created_at":  msg.CreatedAt.Unix(),
			"retry_count": msg.RetryCount,
		}

		if msg.Headers != nil {
			streamData["headers"] = q.serializeHeaders(msg.Headers)
		}

		if msg.Deadline != nil {
			streamData["deadline"] = msg.Deadline.Unix()
		}

		if msg.Delay > 0 {
			streamData["process_after"] = time.Now().Add(msg.Delay).Unix()
		}

		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: q.name,
			Values: streamData,
		})

		ids = append(ids, msg.ID)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		if q.metrics != nil {
			q.metrics.IncrementErrors("batch_send")
		}
		return nil, fmt.Errorf("failed to send batch: %v", err)
	}

	if q.metrics != nil {
		q.metrics.AddSent(len(messages))
	}

	return ids, nil
}

// Consume processes messages with advanced options
func (q *Queue) Consume(handler HandlerFunc, options ...*ConsumeOptions) error {
	opts := &ConsumeOptions{
		BatchSize:         q.config.BatchSize,
		MaxConcurrency:    1,
		VisibilityTimeout: q.config.VisibilityTimeout,
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

	return q.consumeMessages(handler, opts)
}

// consumeMessages is the core consumption logic
func (q *Queue) consumeMessages(handler HandlerFunc, opts *ConsumeOptions) error {
	ctx := context.Background()
	sem := make(chan struct{}, opts.MaxConcurrency)

	for {
		// Read messages from stream
		result := q.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    q.config.ConsumerGroup,
			Consumer: q.config.ConsumerName,
			Streams:  []string{q.name, ">"},
			Count:    int64(opts.BatchSize),
			Block:    100 * time.Millisecond,
		})

		if result.Err() != nil {
			if result.Err() == redis.Nil {
				continue
			}
			return fmt.Errorf("failed to read messages: %v", result.Err())
		}

		streams := result.Val()
		if len(streams) == 0 || len(streams[0].Messages) == 0 {
			continue
		}

		// Process messages
		var wg sync.WaitGroup
		for _, streamMsg := range streams[0].Messages {
			sem <- struct{}{}
			wg.Add(1)

			go func(sm redis.XMessage) {
				defer wg.Done()
				defer func() { <-sem }()

				msg, err := q.parseMessage(sm)
				if err != nil {
					fmt.Printf("Failed to parse message: %v\n", err)
					return
				}

				// Apply filter if provided
				if opts.FilterFunc != nil && !opts.FilterFunc(msg) {
					return
				}

				// Check if message should be processed (delay handling)
				if !q.shouldProcessMessage(msg) {
					return
				}

				// Process message
				err = q.processMessage(ctx, msg, handler, opts)
				if err != nil && q.metrics != nil {
					q.metrics.IncrementErrors("process")
				}

			}(streamMsg)
		}

		wg.Wait()
	}
}

// Helper functions
func (q *Queue) serializePayload(payload map[string]interface{}) string {
	data, _ := json.Marshal(payload)
	return string(data)
}

func (q *Queue) serializeHeaders(headers map[string]string) string {
	data, _ := json.Marshal(headers)
	return string(data)
}

func (q *Queue) parseMessage(streamMsg redis.XMessage) (*Message, error) {
	msg := &Message{
		StreamID: streamMsg.ID,
	}

	if id, ok := streamMsg.Values["id"].(string); ok {
		msg.ID = id
	}

	if payloadStr, ok := streamMsg.Values["payload"].(string); ok {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(payloadStr), &payload); err == nil {
			msg.Payload = payload
		}
	}

	if headersStr, ok := streamMsg.Values["headers"].(string); ok {
		var headers map[string]string
		if err := json.Unmarshal([]byte(headersStr), &headers); err == nil {
			msg.Headers = headers
		}
	}

	if priority, ok := streamMsg.Values["priority"].(string); ok {
		if p, err := json.Number(priority).Int64(); err == nil {
			msg.Priority = int(p)
		}
	}

	if maxRetries, ok := streamMsg.Values["max_retries"].(string); ok {
		if mr, err := json.Number(maxRetries).Int64(); err == nil {
			msg.MaxRetries = int(mr)
		}
	}

	if retryCount, ok := streamMsg.Values["retry_count"].(string); ok {
		if rc, err := json.Number(retryCount).Int64(); err == nil {
			msg.RetryCount = int(rc)
		}
	}

	if createdAt, ok := streamMsg.Values["created_at"].(string); ok {
		if ts, err := json.Number(createdAt).Int64(); err == nil {
			msg.CreatedAt = time.Unix(ts, 0)
		}
	}

	if deadline, ok := streamMsg.Values["deadline"].(string); ok {
		if ts, err := json.Number(deadline).Int64(); err == nil {
			deadlineTime := time.Unix(ts, 0)
			msg.Deadline = &deadlineTime
		}
	}

	return msg, nil
}

func (q *Queue) shouldProcessMessage(msg *Message) bool {
	now := time.Now()

	// Check if message has passed its deadline
	if msg.Deadline != nil && now.After(*msg.Deadline) {
		// Move to DLQ or handle expired message
		if q.config.EnableDLQ {
			q.dlq.Send(msg, "deadline_exceeded")
		}
		return false
	}

	// Check if message should be delayed
	if processAfter, ok := msg.Headers["process_after"]; ok {
		if ts, err := time.Parse(time.RFC3339, processAfter); err == nil {
			if now.Before(ts) {
				return false
			}
		}
	}

	return true
}

func (q *Queue) processMessage(ctx context.Context, msg *Message, handler HandlerFunc, opts *ConsumeOptions) error {
	start := time.Now()
	msg.Attempts++

	err := handler(ctx, msg)

	if q.metrics != nil {
		q.metrics.RecordProcessingTime(time.Since(start))
	}

	if opts.AutoAck {
		if err != nil {
			return q.handleError(msg, err, opts)
		} else {
			return q.Ack(msg.StreamID)
		}
	}

	return err
}

func (q *Queue) handleError(msg *Message, err error, opts *ConsumeOptions) error {
	msg.RetryCount++

	// Check if max retries exceeded
	if msg.RetryCount >= msg.MaxRetries {
		if q.config.EnableDLQ {
			return q.dlq.Send(msg, fmt.Sprintf("max_retries_exceeded: %v", err))
		}
		return q.Ack(msg.StreamID) // Acknowledge to prevent infinite retries
	}

	// Calculate retry delay
	delay := q.calculateRetryDelay(msg.RetryCount, opts.RetryPolicy)

	// Requeue with delay
	return q.requeue(msg, delay)
}

func (q *Queue) calculateRetryDelay(retryCount int, policy *RetryPolicy) time.Duration {
	if policy == nil {
		return q.config.RetryDelay
	}

	delay := policy.InitialDelay
	for i := 1; i < retryCount; i++ {
		delay = time.Duration(float64(delay) * policy.BackoffFactor)
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
			break
		}
	}

	// Add jitter if enabled
	if policy.Jitter {
		jitter := time.Duration(float64(delay) * 0.1 * (2.0*rand.Float64() - 1.0))
		delay += jitter
	}

	return delay
}

func (q *Queue) requeue(msg *Message, delay time.Duration) error {
	// Remove from current stream
	err := q.Ack(msg.StreamID)
	if err != nil {
		return err
	}

	// Send back with delay
	msg.Delay = delay
	msg.ProcessedAt = nil

	_, err = q.SendMessage(msg)
	return err
}

// More methods continue...

// Ack acknowledges a message by its stream ID
func (q *Queue) Ack(streamID string) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	ctx := context.Background()
	return q.rdb.XAck(ctx, q.name, q.config.ConsumerGroup, streamID).Err()
}

// GetMetrics returns current queue metrics
func (q *Queue) GetMetrics() *MetricsSnapshot {
	if q.metrics == nil {
		return nil
	}
	return q.metrics.GetSnapshot()
}

// Close gracefully closes the queue and its connections
func (q *Queue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Stop all consumers
	for _, consumer := range q.consumers {
		if consumer.IsActive() {
			consumer.Stop()
		}
	}

	// Close Redis connection
	if q.client != nil {
		return q.client.Close()
	}

	// For cluster clients, we need to handle differently
	if clusterClient, ok := q.rdb.(*redis.ClusterClient); ok {
		return clusterClient.Close()
	}

	return nil
}

// GetHealthStatus returns queue health information
func (q *Queue) GetHealthStatus() (*HealthStatus, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	ctx := context.Background()

	health := &HealthStatus{
		QueueName: q.name,
		Timestamp: time.Now(),
		Healthy:   true,
		Issues:    []string{},
	}

	// Test Redis connection
	if err := q.rdb.Ping(ctx).Err(); err != nil {
		health.Healthy = false
		health.RedisConnected = false
		health.Issues = append(health.Issues, fmt.Sprintf("Redis connection failed: %v", err))
	} else {
		health.RedisConnected = true
	}

	// Get consumer count
	health.ConsumerCount = len(q.consumers)

	// Get pending messages count
	if health.RedisConnected {
		if pendingInfo := q.rdb.XPending(ctx, q.name, q.config.ConsumerGroup).Val(); pendingInfo.Count >= 0 {
			health.PendingMessages = pendingInfo.Count
		}
	}

	// Get processing rate from metrics
	if q.metrics != nil {
		health.ProcessingRate = q.metrics.GetThroughput()
		health.ErrorRate = q.metrics.GetErrorRate()

		snapshot := q.metrics.GetSnapshot()
		health.LastActivity = snapshot.Timestamp

		// Check for high error rates
		if health.ErrorRate > 0.1 { // More than 10% error rate
			health.Issues = append(health.Issues, fmt.Sprintf("High error rate: %.1f%%", health.ErrorRate*100))
		}

		// Check for low throughput (could indicate problems)
		if health.ProcessingRate < 1.0 && snapshot.MessagesProcessed > 0 {
			health.Issues = append(health.Issues, "Low processing throughput detected")
		}
	}

	// Overall health determination
	if len(health.Issues) > 0 {
		health.Healthy = false
	}

	return health, nil
}

// GetInfo returns comprehensive queue information
func (q *Queue) GetInfo() (*QueueInfo, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	ctx := context.Background()

	// Get basic stream length
	lengthResult := q.rdb.XLen(ctx, q.name)
	length := int64(0)
	if lengthResult.Err() == nil {
		length = lengthResult.Val()
	}

	// Try to get stream info, but handle errors gracefully
	lastMessageID := "0-0"
	streamResult := q.rdb.XInfoStream(ctx, q.name)
	if streamResult.Err() == nil {
		streamInfo := streamResult.Val()
		lastMessageID = streamInfo.LastGeneratedID
	}

	// Get consumer group info
	groupResult := q.rdb.XInfoGroups(ctx, q.name)
	groupCount := 0
	if groupResult.Err() == nil {
		groupInfo := groupResult.Val()
		groupCount = len(groupInfo)
	}

	// Get pending count
	pendingCount := int64(0)
	if groupCount > 0 {
		pendingResult := q.rdb.XPending(ctx, q.name, q.config.ConsumerGroup)
		if pendingResult.Err() == nil {
			pendingInfo := pendingResult.Val()
			pendingCount = pendingInfo.Count
		}
	}

	info := &QueueInfo{
		Name:            q.name,
		Length:          length,
		ConsumerGroups:  groupCount,
		PendingMessages: pendingCount,
		LastMessageID:   lastMessageID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if q.metrics != nil {
		snapshot := q.metrics.GetSnapshot()
		info.Metrics = snapshot
	}

	return info, nil
}

// GetRedisClient returns the Redis client for advanced operations
func (q *Queue) GetRedisClient() *redis.Client {
	if q.client != nil {
		return q.client
	}

	// Try to cast the cmdable interface to a client
	if client, ok := q.rdb.(*redis.Client); ok {
		return client
	}

	return nil
}

// GetRedisCmdable returns the Redis cmdable interface (works with both single and cluster)
func (q *Queue) GetRedisCmdable() redis.Cmdable {
	return q.rdb
}

// GetCircuitBreaker returns the circuit breaker instance
func (q *Queue) GetCircuitBreaker() *CircuitBreaker {
	return q.circuitBreaker
}

// GetRateLimiter returns the rate limiter instance
func (q *Queue) GetRateLimiter() interface{} {
	return q.rateLimiter
}

// GetEncryption returns the encryption handler
func (q *Queue) GetEncryption() *MessageEncryption {
	return q.encryption
}

// GetScheduler returns the message scheduler
func (q *Queue) GetScheduler() *MessageScheduler {
	return q.scheduler
}

// GetTracing returns the message tracing handler
func (q *Queue) GetTracing() *MessageTracing {
	return q.tracing
}

// createRedisClient creates a single Redis client with enhanced authentication
func createRedisClient(config *Config) (*redis.Client, error) {
	opts := &redis.Options{
		Addr:         config.RedisAddress,
		Password:     config.RedisPassword,
		DB:           config.RedisDB,
		Username:     config.RedisUsername,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxConnAge:   config.MaxConnAge,
		PoolTimeout:  config.PoolTimeout,
		IdleTimeout:  config.IdleTimeout,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		MaxRetries:   3,
	}

	// Configure TLS if enabled
	if config.EnableTLS {
		tlsConfig, err := createTLSConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %v", err)
		}
		opts.TLSConfig = tlsConfig
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis server: %v", err)
	}

	return client, nil
}

// createRedisClusterClient creates a Redis cluster client
func createRedisClusterClient(config *Config) (*redis.ClusterClient, error) {
	addresses := config.ClusterAddresses
	if len(addresses) == 0 {
		addresses = []string{config.RedisAddress}
	}

	opts := &redis.ClusterOptions{
		Addrs:        addresses,
		Password:     config.RedisPassword,
		Username:     config.RedisUsername,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxConnAge:   config.MaxConnAge,
		PoolTimeout:  config.PoolTimeout,
		IdleTimeout:  config.IdleTimeout,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		MaxRetries:   3,
	}

	// Configure TLS if enabled
	if config.EnableTLS {
		tlsConfig, err := createTLSConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %v", err)
		}
		opts.TLSConfig = tlsConfig
	}

	client := redis.NewClusterClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis cluster: %v", err)
	}

	return client, nil
}

// createRedisSentinelClient creates a Redis client with Sentinel support
func createRedisSentinelClient(config *Config) (*redis.Client, error) {
	addresses := config.SentinelAddresses
	if len(addresses) == 0 {
		return nil, fmt.Errorf("sentinel addresses are required when sentinel is enabled")
	}

	if config.SentinelMaster == "" {
		return nil, fmt.Errorf("sentinel master name is required")
	}

	opts := &redis.FailoverOptions{
		MasterName:       config.SentinelMaster,
		SentinelAddrs:    addresses,
		SentinelPassword: config.SentinelPassword,
		Password:         config.RedisPassword,
		Username:         config.RedisUsername,
		DB:               config.RedisDB,
		PoolSize:         config.PoolSize,
		MinIdleConns:     config.MinIdleConns,
		MaxConnAge:       config.MaxConnAge,
		PoolTimeout:      config.PoolTimeout,
		IdleTimeout:      config.IdleTimeout,
		DialTimeout:      config.DialTimeout,
		ReadTimeout:      config.ReadTimeout,
		WriteTimeout:     config.WriteTimeout,
		MaxRetries:       3,
	}

	// Configure TLS if enabled
	if config.EnableTLS {
		tlsConfig, err := createTLSConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %v", err)
		}
		opts.TLSConfig = tlsConfig
	}

	client := redis.NewFailoverClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis via Sentinel: %v", err)
	}

	return client, nil
}

// createTLSConfig creates TLS configuration for secure Redis connections
func createTLSConfig(config *Config) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.TLSInsecure,
	}

	// Load client certificate if provided
	if config.TLSCertFile != "" && config.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.TLSCertFile, config.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate if provided
	if config.TLSCACertFile != "" {
		caCert, err := ioutil.ReadFile(config.TLSCACertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %v", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}
