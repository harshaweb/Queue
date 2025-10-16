package redis

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/go-redis/redis/v8"
)

// Config represents Redis configuration
type Config struct {
	// Connection settings
	Addresses []string `yaml:"addresses"`
	Password  string   `yaml:"password"`
	DB        int      `yaml:"db"`

	// Pool settings
	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	MaxRetries   int           `yaml:"max_retries"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	PoolTimeout  time.Duration `yaml:"pool_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`

	// TLS settings
	TLSEnabled bool `yaml:"tls_enabled"`
	TLSConfig  *tls.Config

	// Cluster/Sentinel settings
	ClusterMode   bool     `yaml:"cluster_mode"`
	SentinelMode  bool     `yaml:"sentinel_mode"`
	MasterName    string   `yaml:"master_name"`
	SentinelAddrs []string `yaml:"sentinel_addrs"`
}

// DefaultConfig returns a default Redis configuration
func DefaultConfig() *Config {
	return &Config{
		Addresses:    []string{"localhost:6379"},
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 1,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
		IdleTimeout:  5 * time.Minute,
	}
}

// Client wraps Redis client with additional functionality
type Client struct {
	client redis.Cmdable
	config *Config
}

// NewClient creates a new Redis client based on configuration
func NewClient(config *Config) (*Client, error) {
	var client redis.Cmdable

	if config.ClusterMode {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        config.Addresses,
			Password:     config.Password,
			PoolSize:     config.PoolSize,
			MinIdleConns: config.MinIdleConns,
			MaxRetries:   config.MaxRetries,
			DialTimeout:  config.DialTimeout,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			PoolTimeout:  config.PoolTimeout,
			IdleTimeout:  config.IdleTimeout,
			TLSConfig:    config.TLSConfig,
		})
	} else if config.SentinelMode {
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    config.MasterName,
			SentinelAddrs: config.SentinelAddrs,
			Password:      config.Password,
			DB:            config.DB,
			PoolSize:      config.PoolSize,
			MinIdleConns:  config.MinIdleConns,
			MaxRetries:    config.MaxRetries,
			DialTimeout:   config.DialTimeout,
			ReadTimeout:   config.ReadTimeout,
			WriteTimeout:  config.WriteTimeout,
			PoolTimeout:   config.PoolTimeout,
			IdleTimeout:   config.IdleTimeout,
			TLSConfig:     config.TLSConfig,
		})
	} else {
		// Single node or simple setup
		addr := "localhost:6379"
		if len(config.Addresses) > 0 {
			addr = config.Addresses[0]
		}

		client = redis.NewClient(&redis.Options{
			Addr:         addr,
			Password:     config.Password,
			DB:           config.DB,
			PoolSize:     config.PoolSize,
			MinIdleConns: config.MinIdleConns,
			MaxRetries:   config.MaxRetries,
			DialTimeout:  config.DialTimeout,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			PoolTimeout:  config.PoolTimeout,
			IdleTimeout:  config.IdleTimeout,
			TLSConfig:    config.TLSConfig,
		})
	}

	return &Client{
		client: client,
		config: config,
	}, nil
}

// Ping tests the connection to Redis
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Get returns the Redis client
func (c *Client) Get() redis.Cmdable {
	return c.client
}

// Close closes the Redis connection
func (c *Client) Close() error {
	if closer, ok := c.client.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// IsClusterMode returns true if running in cluster mode
func (c *Client) IsClusterMode() bool {
	return c.config.ClusterMode
}

// GetShardKey returns the shard key for a given queue name
func (c *Client) GetShardKey(queueName string) string {
	if !c.config.ClusterMode {
		return queueName
	}
	// Use consistent hashing for cluster mode
	return "{" + queueName + "}"
}

// StreamKey returns the Redis stream key for a queue
func (c *Client) StreamKey(queueName string) string {
	return c.GetShardKey("queue:" + queueName)
}

// ConsumerGroupKey returns the consumer group key
func (c *Client) ConsumerGroupKey(queueName string) string {
	return c.GetShardKey("cg:" + queueName)
}

// DeadLetterKey returns the dead letter queue key
func (c *Client) DeadLetterKey(queueName string) string {
	return c.GetShardKey("dlq:" + queueName)
}

// ScheduledKey returns the scheduled messages key
func (c *Client) ScheduledKey(queueName string) string {
	return c.GetShardKey("scheduled:" + queueName)
}

// QueueConfigKey returns the queue configuration key
func (c *Client) QueueConfigKey(queueName string) string {
	return c.GetShardKey("config:" + queueName)
}

// QueueStatsKey returns the queue statistics key
func (c *Client) QueueStatsKey(queueName string) string {
	return c.GetShardKey("stats:" + queueName)
}

// QueuesSetKey returns the key for the set of all queues
func (c *Client) QueuesSetKey() string {
	return "queues"
}
