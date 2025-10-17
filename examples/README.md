# Queue SDK Examples

This directory contains practical examples showing how to use the Queue SDK in different scenarios.

## Running the Examples

Make sure you have Redis running locally:

```bash
# Using Docker
docker run -d -p 6379:6379 redis:7-alpine

# Or install Redis locally
# See: https://redis.io/docs/install/
```

Then run any example:

```bash
cd examples/basic
go run main.go
```

## Available Examples

### 1. Basic Usage (`basic/`)
- Simple send and receive operations
- Basic queue creation and health checks
- Getting queue statistics
- Fundamental message processing

**Key concepts**: Queue creation, sending messages, receiving with handler functions

### 2. High Throughput (`high-throughput/`)
- Worker pools for concurrent processing
- Batch sending of messages  
- High-performance configuration
- Concurrent message processing at scale

**Key concepts**: Worker pools, batch operations, concurrency, performance tuning

### 3. JSON & Scheduling (`json-scheduling/`)
- Sending structured data as JSON
- Scheduled/delayed message delivery
- Priority-based message processing
- Batch JSON operations

**Key concepts**: JSON serialization, message scheduling, priorities, structured data

### 4. Error Handling (`error-handling/`)
- Intelligent retry logic
- Retryable vs non-retryable errors
- Dead letter queue patterns
- Payment processing simulation with failures

**Key concepts**: Error handling, retry strategies, failure recovery, production patterns

## Example Patterns

### Quick Send & Forget
```go
// Fire and forget - don't wait for processing
err := queue.SendAndForget("notifications", map[string]interface{}{
    "user_id": 123,
    "message": "Welcome!",
})
```

### Process Single Message
```go
// Process just one message and exit
err := queue.ProcessOnce("my-queue", func(ctx context.Context, msg *queue.Message) error {
    return processMessage(msg.Payload)
})
```

### Custom Configuration
```go
config := &queue.Config{
    RedisAddress:      "redis-cluster:6379",
    RedisPassword:     "secret",
    DefaultTimeout:    60 * time.Second,
    DefaultMaxRetries: 5,
    BatchSize:         100,
}
q, err := queue.New("my-queue", config)
```

### Monitoring
```go
// Health check
if err := q.Health(); err != nil {
    log.Printf("Queue unhealthy: %v", err)
}

// Statistics
stats, err := q.GetStats()
fmt.Printf("Pending: %d, Processing: %d\n", 
    stats.PendingMessages, stats.ProcessingMessages)
```

## Production Tips

1. **Always handle errors**: Return errors for retryable failures, nil for non-retryable
2. **Use worker pools**: For high throughput, use `NewWorkerPool()` 
3. **Batch when possible**: Use `BatchSend()` for sending multiple messages
4. **Monitor health**: Regularly check `Health()` and `GetStats()`
5. **Configure timeouts**: Set appropriate timeouts for your use case
6. **Handle context**: Respect context cancellation in your handlers

## Need Help?

- Check the main README.md for API documentation
- Look at the source code in `pkg/queue/simple.go`
- Open an issue on GitHub for questions