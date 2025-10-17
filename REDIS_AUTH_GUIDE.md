# 🔐 Redis Authentication & Security Features

## Overview

The queue system now provides comprehensive Redis authentication and security features, supporting various deployment scenarios from simple password authentication to enterprise-grade TLS encryption, Redis Cluster, and Sentinel configurations.

## 🚀 Features Added

### ✅ **Basic Authentication**
- **Password Authentication**: Standard Redis AUTH
- **Redis 6.0+ ACL**: Username/password authentication 
- **Database Selection**: Multi-database support

### ✅ **Advanced Security**
- **TLS/SSL Encryption**: Full TLS 1.2+ support
- **Client Certificates**: Mutual TLS authentication
- **CA Certificate Validation**: Custom certificate authorities
- **Insecure Mode**: For testing environments

### ✅ **High Availability**
- **Redis Cluster**: Multi-node cluster support
- **Redis Sentinel**: Automatic failover
- **Connection Pooling**: Optimized connection management
- **Health Monitoring**: Connection health checks

### ✅ **Production Features**
- **Connection Timeouts**: Configurable dial, read, write timeouts
- **Pool Management**: Min/max connections, idle timeouts
- **Automatic Retry**: Configurable retry logic
- **Graceful Degradation**: Fallback mechanisms

## 📚 Configuration Reference

### Basic Password Authentication

```go
config := &pkg.Config{
    RedisAddress:  "localhost:6379",
    RedisPassword: "your-redis-password",
    RedisDB:       0,
    
    // Connection pool
    PoolSize:     20,
    MinIdleConns: 5,
    
    // Timeouts
    DialTimeout:  5 * time.Second,
    ReadTimeout:  3 * time.Second,
    WriteTimeout: 3 * time.Second,
}

queue, err := pkg.NewQueue("secure-queue", config)
```

### Redis 6.0+ ACL Authentication

```go
config := &pkg.Config{
    RedisAddress:  "localhost:6379",
    RedisUsername: "queue-service",     // ACL username
    RedisPassword: "secure-password",   // ACL password
    RedisDB:       0,
    
    // Enhanced connection settings
    PoolSize:      25,
    MinIdleConns:  10,
    MaxConnAge:    time.Hour,
    PoolTimeout:   4 * time.Second,
    IdleTimeout:   5 * time.Minute,
}
```

### TLS/SSL Encryption

```go
config := &pkg.Config{
    RedisAddress:  "secure-redis.example.com:6380",
    RedisPassword: "secure-password",
    RedisDB:       0,
    
    // TLS Configuration
    EnableTLS:     true,
    TLSInsecure:   false,                    // Set true only for testing
    TLSCertFile:   "/path/to/client.crt",    // Client certificate
    TLSKeyFile:    "/path/to/client.key",    // Client private key
    TLSCACertFile: "/path/to/ca.crt",        // CA certificate
    
    // Production settings
    PoolSize:      50,
    MinIdleConns:  15,
    DialTimeout:   10 * time.Second,
    ReadTimeout:   5 * time.Second,
    WriteTimeout:  5 * time.Second,
}
```

### Redis Cluster Configuration

```go
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
    PoolSize:      100,
    MinIdleConns:  25,
    DialTimeout:   15 * time.Second,
    ReadTimeout:   10 * time.Second,
    WriteTimeout:  10 * time.Second,
}
```

### Redis Sentinel Configuration

```go
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
    
    // Failover settings
    PoolSize:     30,
    MinIdleConns: 10,
    DialTimeout:  10 * time.Second,
}
```

## 🏭 Production Configuration Example

```go
config := &pkg.Config{
    // Redis connection
    RedisAddress:  "prod-redis.internal:6379",
    RedisPassword: "prod-secure-password",
    RedisUsername: "queue-service",
    RedisDB:       3,
    
    // Security
    EnableTLS:     true,
    TLSCertFile:   "/etc/ssl/certs/queue-client.crt",
    TLSKeyFile:    "/etc/ssl/private/queue-client.key",
    TLSCACertFile: "/etc/ssl/certs/ca.crt",
    
    // Production connection pool
    PoolSize:       100,
    MinIdleConns:   25,
    MaxConnAge:     4 * time.Hour,
    PoolTimeout:    10 * time.Second,
    IdleTimeout:    30 * time.Minute,
    IdleCheckFreq:  time.Minute,
    
    // Production timeouts
    DialTimeout:  15 * time.Second,
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 10 * time.Second,
    
    // Queue settings
    MaxRetries:        5,
    RetryDelay:        5 * time.Second,
    VisibilityTimeout: 15 * time.Minute,
    MessageTTL:        24 * time.Hour,
    BatchSize:         50,
    
    // Production features
    EnableDLQ:         true,
    EnableMetrics:     true,
    EnableCompression: true,
}

queue, err := pkg.NewQueue("production-queue", config)
```

## 🔧 Environment Variables Support

You can also configure Redis authentication via environment variables:

```bash
# Basic authentication
REDIS_ADDRESS=localhost:6379
REDIS_PASSWORD=your-password
REDIS_DB=0

# ACL authentication (Redis 6.0+)
REDIS_USERNAME=queue-user
REDIS_PASSWORD=user-password

# TLS configuration
REDIS_TLS_ENABLED=true
REDIS_TLS_CERT_FILE=/path/to/client.crt
REDIS_TLS_KEY_FILE=/path/to/client.key
REDIS_TLS_CA_FILE=/path/to/ca.crt

# Cluster configuration
REDIS_CLUSTER=true
REDIS_CLUSTER_ADDRESSES=node1:6379,node2:6379,node3:6379

# Sentinel configuration
REDIS_SENTINEL=true
REDIS_SENTINEL_ADDRESSES=sentinel1:26379,sentinel2:26379,sentinel3:26379
REDIS_SENTINEL_MASTER=mymaster
REDIS_SENTINEL_PASSWORD=sentinel-pass

# Connection pool settings
REDIS_POOL_SIZE=50
REDIS_MIN_IDLE_CONNS=10
REDIS_DIAL_TIMEOUT=10s
REDIS_READ_TIMEOUT=5s
REDIS_WRITE_TIMEOUT=5s
```

## 🛡️ Security Best Practices

### 1. **Use ACL Authentication**
```go
// Create dedicated user for queue operations
// Redis CLI: ACL SETUSER queue-user on >secure-password ~queue:* +@all -@dangerous
config.RedisUsername = "queue-user"
config.RedisPassword = "secure-password"
```

### 2. **Enable TLS in Production**
```go
config.EnableTLS = true
config.TLSInsecure = false  // Never set to true in production
config.TLSCertFile = "/etc/ssl/certs/queue-client.crt"
config.TLSKeyFile = "/etc/ssl/private/queue-client.key"
config.TLSCACertFile = "/etc/ssl/certs/ca.crt"
```

### 3. **Use Dedicated Database**
```go
config.RedisDB = 3  // Use dedicated database for queue operations
```

### 4. **Configure Connection Limits**
```go
config.PoolSize = 50        // Limit max connections
config.MinIdleConns = 10    // Maintain minimum idle connections
config.MaxConnAge = 4 * time.Hour  // Rotate connections
```

### 5. **Set Appropriate Timeouts**
```go
config.DialTimeout = 15 * time.Second   // Allow time for TLS handshake
config.ReadTimeout = 10 * time.Second   // Prevent hanging reads
config.WriteTimeout = 10 * time.Second  // Prevent hanging writes
```

## 🚀 Deployment Scenarios

### **Scenario 1: Development Environment**
- Basic password authentication
- Single Redis instance
- Relaxed timeouts

### **Scenario 2: Staging Environment**
- ACL authentication
- TLS encryption
- Connection pooling

### **Scenario 3: Production Environment**
- Full TLS with client certificates
- Redis Cluster or Sentinel
- Optimized connection pools
- Comprehensive monitoring

### **Scenario 4: High-Security Environment**
- Mutual TLS authentication
- Custom CA certificates
- Network isolation
- Audit logging

## 📊 Performance Considerations

### **Connection Pooling**
- **PoolSize**: 2-4x number of expected concurrent operations
- **MinIdleConns**: 25-50% of PoolSize
- **MaxConnAge**: 1-4 hours to handle network changes

### **Timeouts**
- **DialTimeout**: 5-15 seconds (longer for TLS)
- **ReadTimeout**: 3-10 seconds (based on message size)
- **WriteTimeout**: 3-10 seconds (based on message size)

### **TLS Impact**
- ~10-20% performance overhead
- Longer connection establishment time
- Higher CPU usage for encryption

### **Cluster Performance**
- Better horizontal scaling
- Higher latency for cross-shard operations
- Automatic load distribution

## 🔍 Monitoring & Debugging

### **Connection Health**
```go
// Get queue health status
health, err := queue.GetHealthStatus()
if err != nil {
    log.Printf("Health check failed: %v", err)
}

fmt.Printf("Redis Connected: %v\n", health.RedisConnected)
fmt.Printf("Active Consumers: %d\n", health.ActiveConsumers)
```

### **Connection Metrics**
```go
// Monitor connection pool usage
client := queue.GetRedisClient()
if client != nil {
    stats := client.PoolStats()
    fmt.Printf("Pool hits: %d\n", stats.Hits)
    fmt.Printf("Pool misses: %d\n", stats.Misses)
    fmt.Printf("Active connections: %d\n", stats.TotalConns)
    fmt.Printf("Idle connections: %d\n", stats.IdleConns)
}
```

## ✅ Testing Your Configuration

```go
func testRedisConnection(config *pkg.Config) error {
    queue, err := pkg.NewQueue("test-queue", config)
    if err != nil {
        return fmt.Errorf("failed to create queue: %v", err)
    }
    defer queue.Close()
    
    // Test basic operations
    msgID, err := queue.Send(map[string]interface{}{
        "test": true,
        "timestamp": time.Now().Unix(),
    }, nil)
    if err != nil {
        return fmt.Errorf("failed to send test message: %v", err)
    }
    
    info, err := queue.GetInfo()
    if err != nil {
        return fmt.Errorf("failed to get queue info: %v", err)
    }
    
    fmt.Printf("✅ Redis connection successful\n")
    fmt.Printf("📤 Sent test message: %s\n", msgID)
    fmt.Printf("📊 Queue length: %d\n", info.Length)
    
    return nil
}
```

## 🎯 Summary

The enhanced Redis authentication support provides:

✅ **Multiple Authentication Methods**: Password, ACL, TLS certificates  
✅ **High Availability**: Cluster and Sentinel support  
✅ **Production Security**: TLS encryption with client certificates  
✅ **Performance Optimization**: Advanced connection pooling  
✅ **Monitoring Integration**: Health checks and metrics  
✅ **Backward Compatibility**: Existing code continues to work  

This makes the queue system suitable for **any Redis deployment scenario**, from simple development setups to enterprise-grade production environments with stringent security requirements.

**The queue system is now enterprise-ready with military-grade security!** 🔐🚀