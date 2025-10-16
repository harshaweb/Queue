// Simple example demonstrating the Redis Queue System Go SDK
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/queue-system/redis-queue/pkg/client"
)

func main() {
	// Example 1: Basic SDK Usage
	basicUsageExample()

	// Example 2: Advanced Features
	advancedFeaturesExample()

	// Example 3: Error Handling and Retries
	errorHandlingExample()

	// Example 4: Multiple Queues and Workers
	multipleQueuesExample()
}

// Example 1: Basic enqueue and consume
func basicUsageExample() {
	fmt.Println("=== Basic SDK Usage Example ===")

	// Create client with default Redis configuration
	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}

	queueClient, err := client.NewClient(config)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer queueClient.Close()

	ctx := context.Background()

	// Create a queue
	queueConfig := &client.QueueConfig{
		Name:              "example-queue",
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}

	err = queueClient.CreateQueue(ctx, queueConfig)
	if err != nil {
		log.Printf("Queue might already exist: %v", err)
	}

	// Send a simple message
	payload := map[string]interface{}{
		"message_type": "welcome_email",
		"user_id":      12345,
		"email":        "user@example.com",
		"timestamp":    time.Now().Unix(),
	}

	result, err := queueClient.Enqueue(ctx, "example-queue", payload)
	if err != nil {
		log.Fatal("Failed to enqueue message:", err)
	}

	fmt.Printf("✅ Message enqueued with ID: %s\n", result.MessageID)

	// Simple message consumer
	handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		data := payload.(map[string]interface{})
		fmt.Printf("📨 Processing message: %+v\n", data)

		// Simulate work
		time.Sleep(100 * time.Millisecond)

		fmt.Printf("✅ Message processed successfully: %s\n", msgCtx.MessageID)
		return client.Ack()
	}

	// Consume one message and return
	go func() {
		time.Sleep(1 * time.Second) // Give time for message to be available
		err = queueClient.Consume(ctx, "example-queue", handler,
			client.WithBatchSize(1),
			client.WithVisibilityTimeout(30*time.Second),
		)
		if err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	time.Sleep(2 * time.Second) // Wait for processing
	fmt.Println()
}

// Example 2: Advanced features like scheduling, priorities, and batching
func advancedFeaturesExample() {
	fmt.Println("=== Advanced Features Example ===")

	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}

	queueClient, err := client.NewClient(config)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer queueClient.Close()

	ctx := context.Background()

	// Scheduled message (delay by 5 seconds)
	scheduledPayload := map[string]interface{}{
		"type":    "scheduled_task",
		"task_id": "task-001",
		"delay":   "5s",
	}

	result, err := queueClient.Enqueue(ctx, "example-queue", scheduledPayload,
		client.WithDelay(5*time.Second),
	)
	if err != nil {
		log.Fatal("Failed to enqueue scheduled message:", err)
	}
	fmt.Printf("⏰ Scheduled message enqueued: %s (will be available in 5s)\n", result.MessageID)

	// High priority message
	priorityPayload := map[string]interface{}{
		"type":     "urgent_task",
		"priority": "high",
		"alert":    "System maintenance required",
	}

	result, err = queueClient.Enqueue(ctx, "example-queue", priorityPayload,
		client.WithPriority(10),
	)
	if err != nil {
		log.Fatal("Failed to enqueue priority message:", err)
	}
	fmt.Printf("🚨 High priority message enqueued: %s\n", result.MessageID)

	// Batch enqueue multiple messages
	batchPayloads := []interface{}{
		map[string]interface{}{"batch_id": "batch-001", "item": 1},
		map[string]interface{}{"batch_id": "batch-001", "item": 2},
		map[string]interface{}{"batch_id": "batch-001", "item": 3},
	}

	batchResult, err := queueClient.BatchEnqueue(ctx, "example-queue", batchPayloads)
	if err != nil {
		log.Fatal("Failed to batch enqueue:", err)
	}
	fmt.Printf("📦 Batch enqueued %d messages\n", len(batchResult.MessageIDs))

	// Message with TTL (time to live)
	ttlPayload := map[string]interface{}{
		"type":    "temporary_task",
		"expires": time.Now().Add(1 * time.Minute).Unix(),
	}

	result, err = queueClient.Enqueue(ctx, "example-queue", ttlPayload,
		client.WithTTL(1*time.Minute),
	)
	if err != nil {
		log.Fatal("Failed to enqueue TTL message:", err)
	}
	fmt.Printf("⏳ TTL message enqueued: %s (expires in 1 minute)\n", result.MessageID)

	fmt.Println()
}

// Example 3: Error handling, retries, and dead letter queues
func errorHandlingExample() {
	fmt.Println("=== Error Handling and Retries Example ===")

	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}

	queueClient, err := client.NewClient(config)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer queueClient.Close()

	ctx := context.Background()

	// Create a queue with dead letter queue
	queueConfig := &client.QueueConfig{
		Name:              "retry-queue",
		VisibilityTimeout: 10 * time.Second,
		MaxRetries:        3,
		DeadLetterEnabled: true,
		DeadLetterQueue:   "dead-letter-queue",
	}

	err = queueClient.CreateQueue(ctx, queueConfig)
	if err != nil {
		log.Printf("Queue might already exist: %v", err)
	}

	// Create dead letter queue
	dlqConfig := &client.QueueConfig{
		Name:              "dead-letter-queue",
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        1,
	}

	err = queueClient.CreateQueue(ctx, dlqConfig)
	if err != nil {
		log.Printf("DLQ might already exist: %v", err)
	}

	// Send a message that will fail processing
	failingPayload := map[string]interface{}{
		"type":          "failing_task",
		"should_fail":   true,
		"max_retries":   2,
		"current_retry": 0,
	}

	result, err := queueClient.Enqueue(ctx, "retry-queue", failingPayload)
	if err != nil {
		log.Fatal("Failed to enqueue failing message:", err)
	}
	fmt.Printf("🎯 Enqueued message that will fail: %s\n", result.MessageID)

	// Consumer with retry logic
	handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		data := payload.(map[string]interface{})
		fmt.Printf("🔄 Processing attempt #%d for message: %s\n", msgCtx.RetryCount+1, msgCtx.MessageID)

		// Simulate a task that fails sometimes
		shouldFail, ok := data["should_fail"].(bool)
		if ok && shouldFail {
			if msgCtx.RetryCount < 2 {
				fmt.Printf("❌ Processing failed, will retry (attempt %d/3)\n", msgCtx.RetryCount+1)
				// Requeue with exponential backoff
				delay := time.Duration((msgCtx.RetryCount+1)*(msgCtx.RetryCount+1)) * time.Second
				return client.Requeue(&delay)
			} else {
				fmt.Printf("💀 Max retries exceeded, sending to dead letter queue\n")
				return client.DeadLetter()
			}
		}

		fmt.Printf("✅ Message processed successfully after %d attempts\n", msgCtx.RetryCount+1)
		return client.Ack()
	}

	// Start consumer for retry queue
	go func() {
		time.Sleep(500 * time.Millisecond)
		err = queueClient.Consume(ctx, "retry-queue", handler,
			client.WithBatchSize(1),
			client.WithVisibilityTimeout(10*time.Second),
			client.WithMaxRetries(3),
		)
		if err != nil {
			log.Printf("Retry queue consumer error: %v", err)
		}
	}()

	// Consumer for dead letter queue
	dlqHandler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		fmt.Printf("💀 Dead letter message received: %+v\n", payload)
		// Log, alert, or manually process dead letter messages
		return client.Ack()
	}

	go func() {
		time.Sleep(1 * time.Second)
		err = queueClient.Consume(ctx, "dead-letter-queue", dlqHandler,
			client.WithBatchSize(1),
			client.WithVisibilityTimeout(30*time.Second),
		)
		if err != nil {
			log.Printf("DLQ consumer error: %v", err)
		}
	}()

	time.Sleep(15 * time.Second) // Wait for retries to complete
	fmt.Println()
}

// Example 4: Multiple queues and worker patterns
func multipleQueuesExample() {
	fmt.Println("=== Multiple Queues and Workers Example ===")

	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}

	queueClient, err := client.NewClient(config)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer queueClient.Close()

	ctx := context.Background()

	// Create multiple queues for different types of work
	queues := []string{"email-queue", "image-processing-queue", "analytics-queue"}

	for _, queueName := range queues {
		queueConfig := &client.QueueConfig{
			Name:              queueName,
			VisibilityTimeout: 30 * time.Second,
			MaxRetries:        3,
		}

		err = queueClient.CreateQueue(ctx, queueConfig)
		if err != nil {
			log.Printf("Queue %s might already exist: %v", queueName, err)
		}
	}

	// Send different types of messages
	emailPayload := map[string]interface{}{
		"type":      "email",
		"recipient": "user@example.com",
		"subject":   "Welcome to our service!",
		"template":  "welcome",
	}

	imagePayload := map[string]interface{}{
		"type":       "image_processing",
		"image_url":  "https://example.com/image.jpg",
		"operations": []string{"resize", "optimize", "watermark"},
	}

	analyticsPayload := map[string]interface{}{
		"type":    "analytics",
		"event":   "user_signup",
		"user_id": 12345,
		"properties": map[string]interface{}{
			"source":  "organic",
			"country": "US",
		},
	}

	// Enqueue messages to different queues
	_, err = queueClient.Enqueue(ctx, "email-queue", emailPayload)
	if err != nil {
		log.Printf("Failed to enqueue email: %v", err)
	}

	_, err = queueClient.Enqueue(ctx, "image-processing-queue", imagePayload)
	if err != nil {
		log.Printf("Failed to enqueue image processing: %v", err)
	}

	_, err = queueClient.Enqueue(ctx, "analytics-queue", analyticsPayload)
	if err != nil {
		log.Printf("Failed to enqueue analytics: %v", err)
	}

	fmt.Printf("📬 Sent messages to %d different queues\n", len(queues))

	// Create specialized workers for each queue
	emailWorker := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		data := payload.(map[string]interface{})
		fmt.Printf("📧 Email worker processing: %s\n", data["subject"])
		time.Sleep(200 * time.Millisecond) // Simulate email sending
		return client.Ack()
	}

	imageWorker := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		data := payload.(map[string]interface{})
		fmt.Printf("🖼️  Image worker processing: %s\n", data["image_url"])
		time.Sleep(500 * time.Millisecond) // Simulate image processing
		return client.Ack()
	}

	analyticsWorker := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		data := payload.(map[string]interface{})
		fmt.Printf("📊 Analytics worker processing: %s\n", data["event"])
		time.Sleep(100 * time.Millisecond) // Simulate analytics processing
		return client.Ack()
	}

	// Start workers for each queue
	go func() {
		time.Sleep(500 * time.Millisecond)
		err = queueClient.Consume(ctx, "email-queue", emailWorker,
			client.WithBatchSize(2),
			client.WithVisibilityTimeout(30*time.Second),
		)
		if err != nil {
			log.Printf("Email worker error: %v", err)
		}
	}()

	go func() {
		time.Sleep(500 * time.Millisecond)
		err = queueClient.Consume(ctx, "image-processing-queue", imageWorker,
			client.WithBatchSize(1), // Image processing might be CPU intensive
			client.WithVisibilityTimeout(60*time.Second),
		)
		if err != nil {
			log.Printf("Image worker error: %v", err)
		}
	}()

	go func() {
		time.Sleep(500 * time.Millisecond)
		err = queueClient.Consume(ctx, "analytics-queue", analyticsWorker,
			client.WithBatchSize(10), // Analytics can be batched
			client.WithVisibilityTimeout(30*time.Second),
		)
		if err != nil {
			log.Printf("Analytics worker error: %v", err)
		}
	}()

	time.Sleep(3 * time.Second) // Wait for processing

	// Get stats for all queues
	fmt.Println("\n📈 Queue Statistics:")
	for _, queueName := range queues {
		stats, err := queueClient.GetQueueStats(ctx, queueName)
		if err != nil {
			log.Printf("Failed to get stats for %s: %v", queueName, err)
			continue
		}

		fmt.Printf("  %s: Length=%d, InFlight=%d, Processed=%d\n",
			queueName, stats.Length, stats.InFlight, stats.Processed)
	}

	fmt.Println("\n✅ Multiple queues example completed!")
}

// Helper function to demonstrate middleware usage
func middlewareExample() {
	fmt.Println("=== Middleware Example ===")

	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}

	queueClient, err := client.NewClient(config)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer queueClient.Close()

	// Add metrics middleware (built-in)
	metricsMiddleware := client.NewMetricsMiddleware()
	queueClient.Use(metricsMiddleware)

	// Add circuit breaker middleware (built-in)
	circuitBreakerMiddleware := client.NewCircuitBreakerMiddleware(&client.CircuitBreakerConfig{
		MaxRequests: 100,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
	})
	queueClient.Use(circuitBreakerMiddleware)

	fmt.Println("✅ Middleware configured!")
}
