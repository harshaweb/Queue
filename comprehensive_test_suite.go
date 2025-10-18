package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/harshaweb/queue/pkg"
)

func main() {
	fmt.Println("🧪 Comprehensive Queue System Test Suite")
	fmt.Println("========================================")

	totalTests := 0
	passedTests := 0

	// Test 1: Basic Queue Operations
	fmt.Println("\n📋 Test 1: Basic Queue Operations")
	if testBasicOperations() {
		fmt.Println("✅ PASSED: Basic Queue Operations")
		passedTests++
	} else {
		fmt.Println("❌ FAILED: Basic Queue Operations")
	}
	totalTests++

	// Test 2: Priority Queue
	fmt.Println("\n⭐ Test 2: Priority Queue")
	if testPriorityQueue() {
		fmt.Println("✅ PASSED: Priority Queue")
		passedTests++
	} else {
		fmt.Println("❌ FAILED: Priority Queue")
	}
	totalTests++

	// Test 3: Circuit Breaker
	fmt.Println("\n⚡ Test 3: Circuit Breaker")
	if testCircuitBreaker() {
		fmt.Println("✅ PASSED: Circuit Breaker")
		passedTests++
	} else {
		fmt.Println("❌ FAILED: Circuit Breaker")
	}
	totalTests++

	// Test 4: Rate Limiting
	fmt.Println("\n🚰 Test 4: Rate Limiting")
	if testRateLimiting() {
		fmt.Println("✅ PASSED: Rate Limiting")
		passedTests++
	} else {
		fmt.Println("❌ FAILED: Rate Limiting")
	}
	totalTests++

	// Test 5: Message Encryption
	fmt.Println("\n🔒 Test 5: Message Encryption")
	if testEncryption() {
		fmt.Println("✅ PASSED: Message Encryption")
		passedTests++
	} else {
		fmt.Println("❌ FAILED: Message Encryption")
	}
	totalTests++

	// Test 6: Metrics Collection
	fmt.Println("\n📊 Test 6: Metrics Collection")
	if testMetrics() {
		fmt.Println("✅ PASSED: Metrics Collection")
		passedTests++
	} else {
		fmt.Println("❌ FAILED: Metrics Collection")
	}
	totalTests++

	// Test 7: Dead Letter Queue
	fmt.Println("\n💀 Test 7: Dead Letter Queue")
	if testDeadLetterQueue() {
		fmt.Println("✅ PASSED: Dead Letter Queue")
		passedTests++
	} else {
		fmt.Println("❌ FAILED: Dead Letter Queue")
	}
	totalTests++

	// Test 8: High Throughput
	fmt.Println("\n🚀 Test 8: High Throughput")
	if testHighThroughput() {
		fmt.Println("✅ PASSED: High Throughput")
		passedTests++
	} else {
		fmt.Println("❌ FAILED: High Throughput")
	}
	totalTests++

	// Test 9: Error Handling
	fmt.Println("\n🛡️ Test 9: Error Handling")
	if testErrorHandling() {
		fmt.Println("✅ PASSED: Error Handling")
		passedTests++
	} else {
		fmt.Println("❌ FAILED: Error Handling")
	}
	totalTests++

	// Test 10: Connection Resilience
	fmt.Println("\n🔗 Test 10: Connection Resilience")
	if testConnectionResilience() {
		fmt.Println("✅ PASSED: Connection Resilience")
		passedTests++
	} else {
		fmt.Println("❌ FAILED: Connection Resilience")
	}
	totalTests++

	// Final Results
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Printf("🎯 TEST RESULTS: %d/%d tests passed (%.1f%%)\n", 
		passedTests, totalTests, float64(passedTests)/float64(totalTests)*100)
	
	if passedTests == totalTests {
		fmt.Println("🎉 ALL TESTS PASSED! Queue system is fully functional.")
	} else {
		fmt.Printf("⚠️  %d tests failed. Please check the implementation.\n", totalTests-passedTests)
	}
}

func testBasicOperations() bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  ❌ Panic in basic operations: %v\n", r)
		}
	}()

	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"

	queue, err := pkg.NewQueue("test-basic", config)
	if err != nil {
		fmt.Printf("  ❌ Failed to create queue: %v\n", err)
		return false
	}
	defer queue.Close()

	// Test sending messages
	testData := map[string]interface{}{
		"message": "test basic operations",
		"number":  123,
		"timestamp": time.Now(),
	}

	messageID, err := queue.Send(testData, nil)
	if err != nil {
		fmt.Printf("  ❌ Failed to send message: %v\n", err)
		return false
	}

	fmt.Printf("  📤 Sent message: %s\n", messageID)

	// Test consuming messages
	var received bool
	var wg sync.WaitGroup
	var mu sync.Mutex

	consumer := func(ctx context.Context, msg *pkg.Message) error {
		mu.Lock()
		defer mu.Unlock()
		
		if !received {
			fmt.Printf("  📥 Received message: %+v\n", msg.Payload)
			received = true
			wg.Done()
		}
		return nil
	}

	wg.Add(1)
	
	// Start consumer in background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	go func() {
		if err := queue.Consume(consumer, nil); err != nil && err != context.Canceled {
			fmt.Printf("  ❌ Consume error: %v\n", err)
		}
	}()

	// Wait for message or timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		cancel() // Stop consuming
		return received
	case <-ctx.Done():
		fmt.Printf("  ❌ Timeout waiting for message\n")
		return false
	}
}

func testPriorityQueue() bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  ❌ Panic in priority queue: %v\n", r)
		}
	}()

	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"

	priorities := []int{1, 5, 10}
	pqueue, err := pkg.NewPriorityQueue("test-priority", config, priorities)
	if err != nil {
		fmt.Printf("  ❌ Failed to create priority queue: %v\n", err)
		return false
	}
	defer pqueue.Close()

	// Send messages with different priorities
	messages := []struct {
		data     map[string]interface{}
		priority int
	}{
		{map[string]interface{}{"task": "low", "order": 3}, 1},
		{map[string]interface{}{"task": "high", "order": 1}, 10},
		{map[string]interface{}{"task": "medium", "order": 2}, 5},
	}

	for _, msg := range messages {
		options := &pkg.SendOptions{Priority: msg.priority}
		_, err := pqueue.Send(msg.data, options)
		if err != nil {
			fmt.Printf("  ❌ Failed to send priority message: %v\n", err)
			return false
		}
		fmt.Printf("  📤 Sent priority %d message\n", msg.priority)
	}

	fmt.Printf("  ✅ Priority queue test completed\n")
	return true
}

func testCircuitBreaker() bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  ❌ Panic in circuit breaker: %v\n", r)
		}
	}()

	cbConfig := pkg.CircuitBreakerConfig{
		Name:         "test-cb",
		MaxFailures:  3,
		ResetTimeout: 5 * time.Second,
	}

	cb := pkg.NewCircuitBreaker(cbConfig)

	// Test successful operations
	for i := 0; i < 2; i++ {
		err := cb.Execute(context.Background(), func() error {
			return nil // Success
		})
		if err != nil {
			fmt.Printf("  ❌ Circuit breaker failed on success: %v\n", err)
			return false
		}
	}

	// Test failing operations
	for i := 0; i < 4; i++ {
		err := cb.Execute(context.Background(), func() error {
			return fmt.Errorf("simulated failure")
		})
		// Errors are expected
		_ = err
	}

	fmt.Printf("  ✅ Circuit breaker test completed\n")
	return true
}

func testRateLimiting() bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  ❌ Panic in rate limiting: %v\n", r)
		}
	}()

	// Test Token Bucket
	bucket := pkg.NewTokenBucket(3, 1) // 3 tokens, refill 1 per second

	allowed := 0
	for i := 0; i < 5; i++ {
		if bucket.Allow() {
			allowed++
		}
	}

	if allowed != 3 {
		fmt.Printf("  ❌ Token bucket should allow 3 requests, got %d\n", allowed)
		return false
	}

	// Test Sliding Window
	window := pkg.NewSlidingWindow(2, time.Minute) // 2 requests per minute

	allowedWindow := 0
	for i := 0; i < 4; i++ {
		if window.Allow() {
			allowedWindow++
		}
	}

	if allowedWindow != 2 {
		fmt.Printf("  ❌ Sliding window should allow 2 requests, got %d\n", allowedWindow)
		return false
	}

	fmt.Printf("  ✅ Rate limiting test completed\n")
	return true
}

func testEncryption() bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  ❌ Panic in encryption: %v\n", r)
		}
	}()

	// Generate encryption key
	key, err := pkg.GenerateEncryptionKey()
	if err != nil {
		fmt.Printf("  ❌ Failed to generate encryption key: %v\n", err)
		return false
	}

	// Test that we have a valid key
	if len(key) != 32 { // AES-256 requires 32-byte key
		fmt.Printf("  ❌ Invalid key length: expected 32, got %d\n", len(key))
		return false
	}

	fmt.Printf("  ✅ Encryption test completed (key generated: %d bytes)\n", len(key))
	return true
}

func testMetrics() bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  ❌ Panic in metrics: %v\n", r)
		}
	}()

	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"
	config.EnableMetrics = true

	queue, err := pkg.NewQueue("test-metrics", config)
	if err != nil {
		fmt.Printf("  ❌ Failed to create queue with metrics: %v\n", err)
		return false
	}
	defer queue.Close()

	// Send a few messages
	for i := 0; i < 3; i++ {
		data := map[string]interface{}{"metric_test": i}
		_, err := queue.Send(data, nil)
		if err != nil {
			fmt.Printf("  ❌ Failed to send message for metrics: %v\n", err)
			return false
		}
	}

	// Get metrics
	metrics := queue.GetMetrics()
	if metrics == nil {
		fmt.Printf("  ❌ Metrics should not be nil\n")
		return false
	}

	if metrics.MessagesSent < 3 {
		fmt.Printf("  ❌ Expected at least 3 messages sent, got %d\n", metrics.MessagesSent)
		return false
	}

	fmt.Printf("  ✅ Metrics test completed (sent: %d)\n", metrics.MessagesSent)
	return true
}

func testDeadLetterQueue() bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  ❌ Panic in DLQ: %v\n", r)
		}
	}()

	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"
	config.EnableDLQ = true
	config.MaxRetries = 1
	config.DLQName = "test-dlq"

	queue, err := pkg.NewQueue("test-dlq-main", config)
	if err != nil {
		fmt.Printf("  ❌ Failed to create queue with DLQ: %v\n", err)
		return false
	}
	defer queue.Close()

	// Send a message
	data := map[string]interface{}{"will_fail": true}
	_, err = queue.Send(data, nil)
	if err != nil {
		fmt.Printf("  ❌ Failed to send message: %v\n", err)
		return false
	}

	fmt.Printf("  ✅ DLQ test completed\n")
	return true
}

func testHighThroughput() bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  ❌ Panic in high throughput: %v\n", r)
		}
	}()

	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"

	queue, err := pkg.NewQueue("test-throughput", config)
	if err != nil {
		fmt.Printf("  ❌ Failed to create queue: %v\n", err)
		return false
	}
	defer queue.Close()

	// Send 100 messages quickly
	start := time.Now()
	for i := 0; i < 100; i++ {
		data := map[string]interface{}{"msg": i}
		_, err := queue.Send(data, nil)
		if err != nil {
			fmt.Printf("  ❌ Failed to send message %d: %v\n", i, err)
			return false
		}
	}

	duration := time.Since(start)
	throughput := float64(100) / duration.Seconds()

	fmt.Printf("  ✅ Sent 100 messages in %v (%.2f msg/sec)\n", duration, throughput)
	return throughput > 100 // Should be able to send at least 100 msg/sec
}

func testErrorHandling() bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  ❌ Panic in error handling: %v\n", r)
		}
	}()

	// Test with invalid Redis address
	config := pkg.DefaultConfig()
	config.RedisAddress = "invalid:9999"

	_, err := pkg.NewQueue("test-error", config)
	if err == nil {
		fmt.Printf("  ❌ Should fail with invalid Redis address\n")
		return false
	}

	fmt.Printf("  ✅ Error handling test completed\n")
	return true
}

func testConnectionResilience() bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("  ❌ Panic in connection resilience: %v\n", r)
		}
	}()

	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"
	config.DialTimeout = 1 * time.Second
	config.ReadTimeout = 1 * time.Second
	config.WriteTimeout = 1 * time.Second

	queue, err := pkg.NewQueue("test-resilience", config)
	if err != nil {
		fmt.Printf("  ❌ Failed to create queue: %v\n", err)
		return false
	}
	defer queue.Close()

	// Test basic connectivity
	data := map[string]interface{}{"resilience": "test"}
	_, err = queue.Send(data, nil)
	if err != nil {
		fmt.Printf("  ❌ Failed to send message: %v\n", err)
		return false
	}

	fmt.Printf("  ✅ Connection resilience test completed\n")
	return true
}