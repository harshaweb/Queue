package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/harshaweb/queue/pkg"
)

func main() {
	fmt.Println("🔐 Redis Authentication & Connection Examples")
	fmt.Println("=============================================")

	// Example 1: Basic Redis with password authentication
	fmt.Println("\n1. Basic Redis with Password Authentication")
	basicAuthExample()

	// Example 2: Redis with ACL (username/password)
	fmt.Println("\n2. Redis 6.0+ ACL Authentication")
	aclAuthExample()

	// Example 3: Redis with TLS encryption
	fmt.Println("\n3. Redis with TLS Encryption")
	tlsExample()

	// Example 4: Redis Cluster configuration
	fmt.Println("\n4. Redis Cluster Configuration")
	clusterExample()

	// Example 5: Redis Sentinel configuration
	fmt.Println("\n5. Redis Sentinel Configuration")
	sentinelExample()

	// Example 6: Production-ready configuration
	fmt.Println("\n6. Production Configuration")
	productionExample()
}

func basicAuthExample() {
	config := &pkg.Config{
		RedisAddress:  "localhost:6379",
		RedisPassword: "your-redis-password", // Set your Redis password here
		RedisDB:       0,

		// Connection pool settings
		PoolSize:     20,
		MinIdleConns: 5,

		// Timeouts
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,

		// Queue settings
		MaxRetries:    3,
		EnableDLQ:     true,
		EnableMetrics: true,
	}

	fmt.Printf("  📊 Connecting to Redis at %s with password authentication\n", config.RedisAddress)

	// This would connect with password - commented out for demo
	// queue, err := pkg.NewQueue("auth-test", config)
	// if err != nil {
	//     log.Printf("  ❌ Failed to connect: %v", err)
	//     return
	// }
	// defer queue.Close()

	fmt.Printf("  ✅ Configuration ready for password authentication\n")
}

func aclAuthExample() {
	config := &pkg.Config{
		RedisAddress:  "localhost:6379",
		RedisUsername: "queue-user",          // ACL username
		RedisPassword: "queue-user-password", // ACL password
		RedisDB:       0,

		// Enhanced connection settings
		PoolSize:     25,
		MinIdleConns: 10,
		MaxConnAge:   time.Hour,
		PoolTimeout:  4 * time.Second,
		IdleTimeout:  5 * time.Minute,

		// Queue configuration
		MaxRetries:        5,
		RetryDelay:        2 * time.Second,
		VisibilityTimeout: 10 * time.Minute,
		EnableDLQ:         true,
		EnableMetrics:     true,
	}

	fmt.Printf("  👤 Redis ACL configuration: user=%s\n", config.RedisUsername)
	fmt.Printf("  ✅ Configuration ready for ACL authentication\n")
}

func tlsExample() {
	config := &pkg.Config{
		RedisAddress:  "redis-server.example.com:6380", // TLS port typically 6380
		RedisPassword: "secure-password",
		RedisDB:       0,

		// TLS Configuration
		EnableTLS:     true,
		TLSInsecure:   false, // Set to true only for testing
		TLSCertFile:   "/path/to/client.crt",
		TLSKeyFile:    "/path/to/client.key",
		TLSCACertFile: "/path/to/ca.crt",

		// Production connection settings
		PoolSize:     50,
		MinIdleConns: 15,
		MaxConnAge:   2 * time.Hour,
		PoolTimeout:  5 * time.Second,
		IdleTimeout:  10 * time.Minute,

		// Enhanced timeouts for secure connections
		DialTimeout:  10 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,

		// Queue settings
		MaxRetries:    3,
		EnableDLQ:     true,
		EnableMetrics: true,
	}

	fmt.Printf("  🔒 TLS Configuration: server=%s\n", config.RedisAddress)
	fmt.Printf("  📜 Client cert: %s\n", config.TLSCertFile)
	fmt.Printf("  🔑 Client key: %s\n", config.TLSKeyFile)
	fmt.Printf("  🏛️  CA cert: %s\n", config.TLSCACertFile)
	fmt.Printf("  ✅ Configuration ready for TLS connection\n")
}

func clusterExample() {
	config := &pkg.Config{
		RedisCluster: true,
		ClusterAddresses: []string{
			"redis-node1.example.com:6379",
			"redis-node2.example.com:6379",
			"redis-node3.example.com:6379",
			"redis-node4.example.com:6379",
			"redis-node5.example.com:6379",
			"redis-node6.example.com:6379",
		},
		RedisPassword: "cluster-password",
		RedisUsername: "cluster-user",

		// Cluster-optimized settings
		PoolSize:     100, // Higher for cluster
		MinIdleConns: 25,
		MaxConnAge:   time.Hour,
		PoolTimeout:  6 * time.Second,
		IdleTimeout:  15 * time.Minute,

		// Cluster timeouts
		DialTimeout:  15 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,

		// Queue configuration
		MaxRetries:    5,
		EnableDLQ:     true,
		EnableMetrics: true,
	}

	fmt.Printf("  🌐 Redis Cluster with %d nodes\n", len(config.ClusterAddresses))
	for i, addr := range config.ClusterAddresses {
		fmt.Printf("    Node %d: %s\n", i+1, addr)
	}
	fmt.Printf("  ✅ Configuration ready for cluster connection\n")
}

func sentinelExample() {
	config := &pkg.Config{
		SentinelEnabled: true,
		SentinelAddresses: []string{
			"sentinel1.example.com:26379",
			"sentinel2.example.com:26379",
			"sentinel3.example.com:26379",
		},
		SentinelMaster:   "mymaster",
		SentinelPassword: "sentinel-password",
		RedisPassword:    "redis-password",
		RedisUsername:    "redis-user",
		RedisDB:          0,

		// Sentinel-optimized settings
		PoolSize:     30,
		MinIdleConns: 10,
		MaxConnAge:   90 * time.Minute,
		PoolTimeout:  5 * time.Second,
		IdleTimeout:  10 * time.Minute,

		// Failover timeouts
		DialTimeout:  10 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,

		// Queue configuration
		MaxRetries:    3,
		EnableDLQ:     true,
		EnableMetrics: true,
	}

	fmt.Printf("  🎯 Redis Sentinel: master=%s\n", config.SentinelMaster)
	fmt.Printf("  🔍 Sentinel nodes:\n")
	for i, addr := range config.SentinelAddresses {
		fmt.Printf("    Sentinel %d: %s\n", i+1, addr)
	}
	fmt.Printf("  ✅ Configuration ready for Sentinel connection\n")
}

func productionExample() {
	config := &pkg.Config{
		// Production Redis settings
		RedisAddress:  "prod-redis.internal:6379",
		RedisPassword: "prod-secure-password",
		RedisUsername: "queue-service",
		RedisDB:       3, // Dedicated database

		// Security
		EnableTLS:     true,
		TLSCertFile:   "/etc/ssl/certs/queue-client.crt",
		TLSKeyFile:    "/etc/ssl/private/queue-client.key",
		TLSCACertFile: "/etc/ssl/certs/ca.crt",

		// Production connection pool
		PoolSize:      100,
		MinIdleConns:  25,
		MaxConnAge:    4 * time.Hour,
		PoolTimeout:   10 * time.Second,
		IdleTimeout:   30 * time.Minute,
		IdleCheckFreq: time.Minute,

		// Production timeouts
		DialTimeout:  15 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,

		// Queue production settings
		MaxRetries:        5,
		RetryDelay:        5 * time.Second,
		VisibilityTimeout: 15 * time.Minute,
		MessageTTL:        24 * time.Hour,
		ConsumerGroup:     "prod-consumers",
		BatchSize:         50,
		PrefetchCount:     200,

		// Production features
		EnableDLQ:         true,
		EnableMetrics:     true,
		EnableCompression: true,
	}

	fmt.Printf("  🏭 Production Configuration Summary:\n")
	fmt.Printf("    🔗 Redis: %s (DB: %d)\n", config.RedisAddress, config.RedisDB)
	fmt.Printf("    👤 User: %s\n", config.RedisUsername)
	fmt.Printf("    🔒 TLS: %v\n", config.EnableTLS)
	fmt.Printf("    🏊 Pool: %d connections (min %d idle)\n", config.PoolSize, config.MinIdleConns)
	fmt.Printf("    ⏱️  Timeouts: dial=%v, read=%v, write=%v\n",
		config.DialTimeout, config.ReadTimeout, config.WriteTimeout)
	fmt.Printf("    📦 Batch size: %d\n", config.BatchSize)
	fmt.Printf("    🔄 Max retries: %d\n", config.MaxRetries)
	fmt.Printf("    💀 DLQ: %v\n", config.EnableDLQ)
	fmt.Printf("    📊 Metrics: %v\n", config.EnableMetrics)
	fmt.Printf("    🗜️  Compression: %v\n", config.EnableCompression)
	fmt.Printf("  ✅ Production configuration ready\n")

	// Example of creating a production queue
	fmt.Printf("\n  🚀 Creating production queue...\n")

	// Use default for demo (would use production config in real scenario)
	demoConfig := pkg.DefaultConfig()
	queue, err := pkg.NewQueue("production-queue", demoConfig)
	if err != nil {
		log.Printf("  ❌ Failed to create queue: %v", err)
		return
	}
	defer queue.Close()

	fmt.Printf("  ✅ Production queue created successfully\n")

	// Test basic functionality
	_ = context.Background() // Would use ctx for operations

	// Send a test message
	testData := map[string]interface{}{
		"service":   "queue-demo",
		"timestamp": time.Now().Unix(),
		"message":   "Production test message",
	}

	messageID, err := queue.Send(testData, nil)
	if err != nil {
		log.Printf("  ❌ Failed to send message: %v", err)
		return
	}

	fmt.Printf("  📤 Sent test message: %s\n", messageID)

	// Get queue info
	info, err := queue.GetInfo()
	if err != nil {
		log.Printf("  ❌ Failed to get queue info: %v", err)
		return
	}

	fmt.Printf("  📊 Queue length: %d messages\n", info.Length)
	fmt.Printf("  ✅ Production queue is operational!\n")
}
