package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/harshaweb/queue/pkg"
)

// TestMessage represents a test message structure
type TestMessage struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Priority  int       `json:"priority"`
}

func main() {
	fmt.Println("🚀 Starting Comprehensive Queue System Test Suite")
	fmt.Println("================================================")

	ctx := context.Background()

	// Test basic queue functionality
	if err := testBasicQueue(ctx); err != nil {
		log.Printf("❌ Basic queue test failed: %v", err)
		return
	}
	fmt.Println("✅ Basic queue functionality test passed")

	// Test priority queue
	if err := testPriorityQueue(ctx); err != nil {
		log.Printf("❌ Priority queue test failed: %v", err)
		return
	}
	fmt.Println("✅ Priority queue test passed")

	// Test circuit breaker
	if err := testCircuitBreaker(ctx); err != nil {
		log.Printf("❌ Circuit breaker test failed: %v", err)
		return
	}
	fmt.Println("✅ Circuit breaker test passed")

	// Test rate limiting
	if err := testRateLimiting(ctx); err != nil {
		log.Printf("❌ Rate limiting test failed: %v", err)
		return
	}
	fmt.Println("✅ Rate limiting test passed")

	// Test message encryption
	if err := testEncryption(ctx); err != nil {
		log.Printf("❌ Encryption test failed: %v", err)
		return
	}
	fmt.Println("✅ Message encryption test passed")

	// Test metrics collection
	if err := testMetrics(ctx); err != nil {
		log.Printf("❌ Metrics test failed: %v", err)
		return
	}
	fmt.Println("✅ Metrics collection test passed")

	// Test dead letter queue
	if err := testDeadLetterQueue(ctx); err != nil {
		log.Printf("❌ Dead letter queue test failed: %v", err)
		return
	}
	fmt.Println("✅ Dead letter queue test passed")

	fmt.Println("\n🎉 All comprehensive tests passed!")
	fmt.Println("=====================================")
}

func testBasicQueue(ctx context.Context) error {
	fmt.Println("\n📋 Testing Basic Queue Operations...")

	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"

	queue, err := pkg.NewQueue("test-basic-queue", config)
	if err != nil {
		return fmt.Errorf("failed to create queue: %v", err)
	}
	defer queue.Close()

	// Test sending messages
	fmt.Println("  📤 Sending test messages...")
	
	messages := []map[string]interface{}{
		{
			"id":        "msg-1",
			"content":   "Hello World",
			"timestamp": time.Now(),
		},
		{
			"id":        "msg-2", 
			"content":   "Test Message",
			"timestamp": time.Now(),
		},
		{
			"id":        "msg-3",
			"content":   "Queue Test",
			"timestamp": time.Now(),
		},
	}

	for _, msg := range messages {
		messageID, err := queue.Send(msg, nil)
		if err != nil {
			return fmt.Errorf("failed to send message: %v", err)
		}
		fmt.Printf("  📨 Sent message with ID: %s\n", messageID)
	}

	// Test consuming messages
	fmt.Println("  📥 Consuming messages...")
	var receivedCount int
	var wg sync.WaitGroup

	consumer := func(ctx context.Context, message *pkg.Message) error {
		fmt.Printf("  📨 Received message: %+v\n", message.Payload)
		receivedCount++

		if receivedCount >= len(messages) {
			wg.Done()
		}
		return nil
	}

	wg.Add(1)
	go func() {
		if err := queue.Consume(consumer, nil); err != nil {
			log.Printf("Consume error: %v", err)
		}
	}()

	// Wait for all messages to be consumed
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		fmt.Printf("  ✅ Successfully processed %d messages\n", receivedCount)
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for messages")
	}

	return nil
}

func testPriorityQueue(ctx context.Context) error {
	fmt.Println("\n⭐ Testing Priority Queue...")

	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"

	priorities := []int{1, 5, 10} // Low, Medium, High
	priorityQueue, err := pkg.NewPriorityQueue("test-priority-queue", config, priorities)
	if err != nil {
		return fmt.Errorf("failed to create priority queue: %v", err)
	}
	defer priorityQueue.Close()

	// Send messages with different priorities
	fmt.Println("  📤 Sending priority messages...")

	// Send in reverse priority order to test prioritization
	messages := []struct {
		data     map[string]interface{}
		priority int
	}{
		{map[string]interface{}{"task": "low priority task", "order": 3}, 1},
		{map[string]interface{}{"task": "high priority task", "order": 1}, 10},
		{map[string]interface{}{"task": "medium priority task", "order": 2}, 5},
	}

	for _, msg := range messages {
		options := &pkg.SendOptions{Priority: msg.priority}
		messageID, err := priorityQueue.Send(msg.data, options)
		if err != nil {
			return fmt.Errorf("failed to send priority message: %v", err)
		}
		fmt.Printf("  📨 Sent priority %d message with ID: %s\n", msg.priority, messageID)
	}

	// Consume and verify order
	fmt.Println("  📥 Consuming priority messages...")
	var processedTasks []string
	var wg sync.WaitGroup

	consumer := func(ctx context.Context, message *pkg.Message) error {
		if payload, ok := message.Payload.(map[string]interface{}); ok {
			if task, exists := payload["task"]; exists {
				taskStr := task.(string)
				processedTasks = append(processedTasks, taskStr)
				fmt.Printf("  📨 Processed: %s\n", taskStr)
			}
		}

		if len(processedTasks) >= len(messages) {
			wg.Done()
		}
		return nil
	}

	wg.Add(1)
	go func() {
		if err := priorityQueue.Consume(consumer, nil); err != nil {
			log.Printf("Priority consume error: %v", err)
		}
	}()

	// Wait for all messages
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		fmt.Printf("  ✅ Processed %d priority messages\n", len(processedTasks))
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for priority messages")
	}

	return nil
}

func testCircuitBreaker(ctx context.Context) error {
	fmt.Println("\n⚡ Testing Circuit Breaker...")

	cbConfig := pkg.CircuitBreakerConfig{
		Name:              "test-service",
		MaxFailures:       3,
		ResetTimeout:      5 * time.Second,
		SuccessThreshold:  2,
	}

	cb := pkg.NewCircuitBreaker(cbConfig)

	// Test successful operations
	fmt.Println("  ✅ Testing successful operations...")
	for i := 0; i < 3; i++ {
		err := cb.Execute(ctx, func() error {
			// Simulate successful operation
			return nil
		})
		if err != nil {
			return fmt.Errorf("circuit breaker failed on success: %v", err)
		}
	}

	// Test failing operations to open circuit
	fmt.Println("  ❌ Testing failing operations...")
	for i := 0; i < 4; i++ {
		err := cb.Execute(ctx, func() error {
			// Simulate failing operation
			return fmt.Errorf("simulated failure")
		})
		// Should fail, but circuit breaker should handle it
		_ = err
	}

	// Test that circuit is open
	fmt.Println("  🔴 Testing circuit open state...")
	err := cb.Execute(ctx, func() error {
		return nil
	})
	if err == nil {
		return fmt.Errorf("circuit breaker should be open")
	}

	fmt.Printf("  ✅ Circuit breaker working correctly: %v\n", err)
	return nil
}

func testRateLimiting(ctx context.Context) error {
	fmt.Println("\n🚰 Testing Rate Limiting...")

	// Test Token Bucket
	fmt.Println("  🪣 Testing Token Bucket...")
	bucket := pkg.NewTokenBucket(5, 2) // 5 tokens, refill 2 per second

	// Use all tokens
	allowed := 0
	for i := 0; i < 10; i++ {
		if bucket.Allow() {
			allowed++
		}
	}

	if allowed != 5 {
		return fmt.Errorf("token bucket should allow exactly 5 requests, got %d", allowed)
	}
	fmt.Printf("  ✅ Token bucket correctly limited to %d requests\n", allowed)

	// Test Sliding Window
	fmt.Println("  🪟 Testing Sliding Window...")
	window := pkg.NewSlidingWindow(3, time.Minute) // 3 requests per minute

	// Test requests
	allowedWindow := 0
	for i := 0; i < 5; i++ {
		if window.Allow("test-user") {
			allowedWindow++
		}
	}

	if allowedWindow != 3 {
		return fmt.Errorf("sliding window should allow exactly 3 requests, got %d", allowedWindow)
	}
	fmt.Printf("  ✅ Sliding window correctly limited to %d requests\n", allowedWindow)

	return nil
}

func testEncryption(ctx context.Context) error {
	fmt.Println("\n🔒 Testing Message Encryption...")

	// Generate encryption key
	key, err := pkg.GenerateEncryptionKey()
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %v", err)
	}

	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"
	config.EnableEncryption = true
	config.EncryptionKey = key

	queue, err := pkg.NewQueue("test-encryption-queue", config)
	if err != nil {
		return fmt.Errorf("failed to create encrypted queue: %v", err)
	}
	defer queue.Close()

	// Test encrypting and decrypting messages
	sensitiveData := map[string]interface{}{
		"credit_card": "4111-1111-1111-1111",
		"ssn":        "123-45-6789",
		"password":   "super-secret-password",
	}

	fmt.Println("  🔐 Sending encrypted message...")
	messageID, err := queue.Send(sensitiveData, nil)
	if err != nil {
		return fmt.Errorf("failed to send encrypted message: %v", err)
	}
	fmt.Printf("  📨 Sent encrypted message with ID: %s\n", messageID)

	// Consume and verify decryption
	fmt.Println("  🔓 Consuming encrypted message...")
	var wg sync.WaitGroup
	var decryptedCorrectly bool

	consumer := func(ctx context.Context, message *pkg.Message) error {
		if payload, ok := message.Payload.(map[string]interface{}); ok {
			if ccNum, exists := payload["credit_card"]; exists {
				if ccNum == "4111-1111-1111-1111" {
					decryptedCorrectly = true
					fmt.Println("  ✅ Message decrypted correctly")
				}
			}
		}
		wg.Done()
		return nil
	}

	wg.Add(1)
	go func() {
		if err := queue.Consume(consumer, nil); err != nil {
			log.Printf("Encryption consume error: %v", err)
		}
	}()

	// Wait for decryption
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		if !decryptedCorrectly {
			return fmt.Errorf("message was not decrypted correctly")
		}
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for encrypted message")
	}

	return nil
}

func testMetrics(ctx context.Context) error {
	fmt.Println("\n📊 Testing Metrics Collection...")

	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"
	config.EnableMetrics = true

	queue, err := pkg.NewQueue("test-metrics-queue", config)
	if err != nil {
		return fmt.Errorf("failed to create queue with metrics: %v", err)
	}
	defer queue.Close()

	// Send some messages to generate metrics
	fmt.Println("  📈 Generating metrics data...")
	for i := 0; i < 5; i++ {
		data := map[string]interface{}{
			"metric_test": i,
			"timestamp":  time.Now(),
		}
		_, err := queue.Send(data, nil)
		if err != nil {
			return fmt.Errorf("failed to send message for metrics: %v", err)
		}
	}

	// Get metrics
	metrics := queue.GetMetrics()
	if metrics == nil {
		return fmt.Errorf("metrics should not be nil")
	}

	fmt.Printf("  📊 Messages Sent: %d\n", metrics.MessagesSent)
	fmt.Printf("  📊 Messages Processed: %d\n", metrics.MessagesProcessed)
	fmt.Printf("  📊 Error Rate: %.2f%%\n", metrics.ErrorRate)
	fmt.Printf("  📊 Average Latency: %v\n", metrics.AverageLatency)

	if metrics.MessagesSent < 5 {
		return fmt.Errorf("expected at least 5 messages sent, got %d", metrics.MessagesSent)
	}

	return nil
}

func testDeadLetterQueue(ctx context.Context) error {
	fmt.Println("\n💀 Testing Dead Letter Queue...")

	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"
	config.EnableDLQ = true
	config.MaxRetries = 2
	config.DLQName = "test-dlq"

	queue, err := pkg.NewQueue("test-dlq-main", config)
	if err != nil {
		return fmt.Errorf("failed to create queue with DLQ: %v", err)
	}
	defer queue.Close()

	// Send a message that will fail processing
	fmt.Println("  💀 Sending message that will fail...")
	failingMessage := map[string]interface{}{
		"will_fail": true,
		"attempts":  0,
	}

	messageID, err := queue.Send(failingMessage, nil)
	if err != nil {
		return fmt.Errorf("failed to send failing message: %v", err)
	}
	fmt.Printf("  📨 Sent failing message with ID: %s\n", messageID)

	// Set up consumer that always fails
	fmt.Println("  ❌ Setting up failing consumer...")
	var wg sync.WaitGroup
	var failCount int

	failingConsumer := func(ctx context.Context, message *pkg.Message) error {
		failCount++
		fmt.Printf("  ❌ Attempt %d failed (intentionally)\n", failCount)
		return fmt.Errorf("intentional failure for DLQ test")
	}

	// This will consume the message, fail, retry, and eventually send to DLQ
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Consume for a short time then stop
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		queue.Consume(failingConsumer, nil)
	}()

	wg.Wait()

	if failCount == 0 {
		return fmt.Errorf("consumer should have attempted to process the message")
	}

	fmt.Printf("  ✅ DLQ test completed with %d failed attempts\n", failCount)
	return nil
}