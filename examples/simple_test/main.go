package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/harshaweb/queue/pkg"
)

func main() {
	fmt.Println("🚀 Testing Updated Queue System")
	fmt.Println("================================")

	ctx := context.Background()

	// Test 1: Basic Queue Operations
	fmt.Println("\n📋 Test 1: Basic Queue Operations")
	if err := testBasicOperations(ctx); err != nil {
		log.Printf("❌ Basic operations test failed: %v", err)
		return
	}
	fmt.Println("✅ Basic operations test passed")

	// Test 2: Priority Queue
	fmt.Println("\n⭐ Test 2: Priority Queue")
	if err := testPriorityQueue(ctx); err != nil {
		log.Printf("❌ Priority queue test failed: %v", err)
		return
	}
	fmt.Println("✅ Priority queue test passed")

	// Test 3: Circuit Breaker
	fmt.Println("\n⚡ Test 3: Circuit Breaker")
	if err := testCircuitBreaker(); err != nil {
		log.Printf("❌ Circuit breaker test failed: %v", err)
		return
	}
	fmt.Println("✅ Circuit breaker test passed")

	// Test 4: Rate Limiting
	fmt.Println("\n🎛️ Test 4: Rate Limiting")
	if err := testRateLimiting(); err != nil {
		log.Printf("❌ Rate limiting test failed: %v", err)
		return
	}
	fmt.Println("✅ Rate limiting test passed")

	// Test 5: Message Encryption
	fmt.Println("\n🔐 Test 5: Message Encryption")
	if err := testEncryption(); err != nil {
		log.Printf("❌ Encryption test failed: %v", err)
		return
	}
	fmt.Println("✅ Encryption test passed")

	fmt.Println("\n🎉 All basic tests passed! Queue system is functional!")
	fmt.Println("=====================================================")
}

func testBasicOperations(ctx context.Context) error {
	// Create queue
	config := pkg.DefaultConfig()
	queue, err := pkg.NewQueue("test-basic", config)
	if err != nil {
		return fmt.Errorf("failed to create queue: %v", err)
	}
	defer queue.Close()

	// Send a message
	testData := map[string]interface{}{
		"message":   "Hello, Queue!",
		"number":    42,
		"timestamp": time.Now().Unix(),
	}

	messageID, err := queue.Send(testData, nil)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	fmt.Printf("  📤 Sent message with ID: %s\n", messageID)

	// Send JSON message
	type TestMessage struct {
		Text  string `json:"text"`
		Value int    `json:"value"`
	}

	jsonMsg := TestMessage{
		Text:  "JSON Test Message",
		Value: 123,
	}

	jsonID, err := queue.SendJSON(jsonMsg, nil)
	if err != nil {
		return fmt.Errorf("failed to send JSON message: %v", err)
	}

	fmt.Printf("  📤 Sent JSON message with ID: %s\n", jsonID)

	// Get queue info
	info, err := queue.GetInfo()
	if err != nil {
		return fmt.Errorf("failed to get queue info: %v", err)
	}

	fmt.Printf("  📊 Queue length: %d messages\n", info.Length)

	return nil
}

func testPriorityQueue(ctx context.Context) error {
	// Create priority queue
	config := pkg.DefaultConfig()
	priorities := []int{1, 2, 3, 4, 5}

	pqueue, err := pkg.NewPriorityQueue("test-priority", config, priorities)
	if err != nil {
		return fmt.Errorf("failed to create priority queue: %v", err)
	}
	defer pqueue.Close()

	// Send messages with different priorities
	for _, priority := range []int{3, 1, 5, 2, 4} {
		message := map[string]interface{}{
			"priority": priority,
			"content":  fmt.Sprintf("Priority %d message", priority),
		}

		_, err := pqueue.Send(message, &pkg.SendOptions{Priority: priority})
		if err != nil {
			return fmt.Errorf("failed to send priority message: %v", err)
		}
	}

	fmt.Printf("  🎯 Sent 5 messages with different priorities\n")

	// Get queue info
	info, err := pqueue.GetInfo()
	if err != nil {
		return fmt.Errorf("failed to get priority queue info: %v", err)
	}

	fmt.Printf("  📊 Priority queue length: %d messages\n", info.Length)
	return nil
}

func testCircuitBreaker() error {
	config := pkg.CircuitBreakerConfig{
		Name:         "test-breaker",
		MaxFailures:  3,
		ResetTimeout: 5 * time.Second,
	}

	cb := pkg.NewCircuitBreaker(config)

	// Test normal operation
	successFunc := func() error {
		return nil // Success
	}

	err := cb.Execute(context.Background(), successFunc)
	if err != nil {
		return fmt.Errorf("expected success, got error: %v", err)
	}
	fmt.Printf("  ✅ Circuit breaker allows successful operations\n")

	// Test failure handling
	failureFunc := func() error {
		return fmt.Errorf("simulated failure")
	}

	// Trigger failures to open circuit
	for i := 0; i < 4; i++ {
		err := cb.Execute(context.Background(), failureFunc)
		if i < 3 {
			if err == nil {
				return fmt.Errorf("expected failure, got success")
			}
		} else {
			// Circuit should be open now
			if err == nil {
				return fmt.Errorf("expected circuit to be open")
			}
		}
	}

	stats := cb.Stats()
	fmt.Printf("  ⚡ Circuit breaker state: %s (after %d failures)\n", stats.State, stats.ConsecutiveErrors)

	return nil
}

func testRateLimiting() error {
	// Test token bucket
	bucket := pkg.NewTokenBucket(5, 2) // 5 tokens, 2 per second

	// Use all tokens
	for i := 0; i < 5; i++ {
		if !bucket.Allow() {
			return fmt.Errorf("expected token to be available")
		}
	}

	// Next request should be denied
	if bucket.Allow() {
		return fmt.Errorf("expected request to be denied")
	}

	fmt.Printf("  🪣 Token bucket: Used 5 tokens, 6th request blocked\n")

	// Test sliding window
	window := pkg.NewSlidingWindow(3, 2*time.Second)

	// Make 3 requests
	for i := 0; i < 3; i++ {
		if !window.Allow() {
			return fmt.Errorf("expected request to be allowed")
		}
	}

	// 4th request should be denied
	if window.Allow() {
		return fmt.Errorf("expected request to be denied")
	}

	fmt.Printf("  🪟 Sliding window: 3 requests allowed, 4th blocked\n")

	return nil
}

func testEncryption() error {
	// Generate encryption key
	key, err := pkg.GenerateEncryptionKey()
	if err != nil {
		return fmt.Errorf("failed to generate key: %v", err)
	}

	config := pkg.EncryptionConfig{
		Enabled:   true,
		Key:       key,
		Algorithm: "AES-256-GCM",
	}

	encryption, err := pkg.NewMessageEncryption(config)
	if err != nil {
		return fmt.Errorf("failed to create encryption: %v", err)
	}

	// Test encryption/decryption
	original := []byte("Secret message that needs encryption!")

	encrypted, err := encryption.Encrypt(original)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %v", err)
	}

	decrypted, err := encryption.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %v", err)
	}

	if string(decrypted) != string(original) {
		return fmt.Errorf("decrypted message doesn't match original")
	}

	fmt.Printf("  🔐 Encrypted %d bytes → %d bytes → %d bytes\n",
		len(original), len(encrypted), len(decrypted))

	return nil
}
