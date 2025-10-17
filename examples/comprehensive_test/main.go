package main

import (
	"context"
	"encoding/json"
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

	// Test delayed queue
	if err := testDelayedQueue(ctx); err != nil {
		log.Printf("❌ Delayed queue test failed: %v", err)
		return
	}
	fmt.Println("✅ Delayed queue test passed")

	// Test batch processing
	if err := testBatchProcessing(ctx); err != nil {
		log.Printf("❌ Batch processing test failed: %v", err)
		return
	}
	fmt.Println("✅ Batch processing test passed")

	// Test dead letter queue
	if err := testDeadLetterQueue(ctx); err != nil {
		log.Printf("❌ Dead letter queue test failed: %v", err)
		return
	}
	fmt.Println("✅ Dead letter queue test passed")

	// Test metrics collection
	if err := testMetrics(ctx); err != nil {
		log.Printf("❌ Metrics test failed: %v", err)
		return
	}
	fmt.Println("✅ Metrics collection test passed")

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

	// Test message scheduling
	if err := testScheduling(ctx); err != nil {
		log.Printf("❌ Scheduling test failed: %v", err)
		return
	}
	fmt.Println("✅ Message scheduling test passed")

	// Test message tracing
	if err := testTracing(ctx); err != nil {
		log.Printf("❌ Tracing test failed: %v", err)
		return
	}
	fmt.Println("✅ Message tracing test passed")

	// Test high-throughput scenario
	if err := testHighThroughput(ctx); err != nil {
		log.Printf("❌ High throughput test failed: %v", err)
		return
	}
	fmt.Println("✅ High throughput test passed")

	fmt.Println("\n🎉 All tests passed! Queue system is ready for production!")
	fmt.Println("=========================================================")
}

func testBasicQueue(ctx context.Context) error {
	fmt.Println("\n📋 Testing Basic Queue Functionality...")

	config := pkg.DefaultConfig()
	queueName := "test-basic-queue"

	queue, err := pkg.NewQueue(queueName, config)
	if err != nil {
		return fmt.Errorf("failed to create queue: %v", err)
	}
	defer queue.Close()

	// Test sending messages
	for i := 0; i < 5; i++ {
		msg := TestMessage{
			ID:        fmt.Sprintf("msg-%d", i),
			Content:   fmt.Sprintf("Test message %d", i),
			Timestamp: time.Now(),
		}

		_, err := queue.SendJSON(msg, nil)
		if err != nil {
			return fmt.Errorf("failed to send message %d: %v", i, err)
		}
	}

	// Test consuming messages
	var receivedCount int
	var wg sync.WaitGroup

	consumer := func(ctx context.Context, message *pkg.Message) error {
		var msg TestMessage
		if err := json.Unmarshal(message.Payload.([]byte), &msg); err != nil {
			return err
		}

		fmt.Printf("  📨 Received: %s - %s\n", msg.ID, msg.Content)
		receivedCount++

		if receivedCount >= 5 {
			wg.Done()
		}
		return nil
	}

	wg.Add(1)
	go queue.Consume(consumer, nil)

	// Wait for all messages to be consumed
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for messages")
	}

	// Test queue info
	info, err := queue.GetInfo()
	if err != nil {
		return fmt.Errorf("failed to get queue info: %v", err)
	}

	fmt.Printf("  📊 Queue Info: %d pending, %d consumer groups\n", info.PendingMessages, info.ConsumerGroups)
	return nil
}

func testPriorityQueue(ctx context.Context) error {
	fmt.Println("\n⭐ Testing Priority Queue...")

	config := pkg.DefaultConfig()
	config.QueueName = "test-priority-queue"

	queue, err := pkg.NewQueue(config)
	if err != nil {
		return err
	}
	defer queue.Close()

	priorityQueue := pkg.NewPriorityQueue(queue.GetRedisClient(), "test-priority")

	// Send messages with different priorities
	priorities := []int{1, 5, 3, 4, 2}
	for i, priority := range priorities {
		msg := TestMessage{
			ID:       fmt.Sprintf("priority-msg-%d", i),
			Content:  fmt.Sprintf("Priority %d message", priority),
			Priority: priority,
		}

		data, _ := json.Marshal(msg)
		if err := priorityQueue.Send(ctx, data, priority); err != nil {
			return fmt.Errorf("failed to send priority message: %v", err)
		}
	}

	// Consume and verify order (should be 5, 4, 3, 2, 1)
	expectedOrder := []int{5, 4, 3, 2, 1}
	for i, expectedPriority := range expectedOrder {
		message, err := priorityQueue.Receive(ctx)
		if err != nil {
			return fmt.Errorf("failed to receive priority message: %v", err)
		}

		var msg TestMessage
		if err := json.Unmarshal(message.Body, &msg); err != nil {
			return err
		}

		if msg.Priority != expectedPriority {
			return fmt.Errorf("wrong priority order: expected %d, got %d", expectedPriority, msg.Priority)
		}

		fmt.Printf("  🎯 Received priority %d message: %s\n", msg.Priority, msg.Content)
	}

	return nil
}

func testDelayedQueue(ctx context.Context) error {
	fmt.Println("\n⏰ Testing Delayed Queue...")

	config := pkg.DefaultConfig()
	config.QueueName = "test-delayed-queue"

	queue, err := pkg.NewQueue(config)
	if err != nil {
		return err
	}
	defer queue.Close()

	delayedQueue := pkg.NewDelayedQueue(queue.GetRedisClient(), "test-delayed")

	// Send a message with 2-second delay
	msg := TestMessage{
		ID:      "delayed-msg-1",
		Content: "This message was delayed",
	}

	data, _ := json.Marshal(msg)
	delay := 2 * time.Second
	deliveryTime := time.Now().Add(delay)

	if err := delayedQueue.SendDelayed(ctx, data, deliveryTime); err != nil {
		return fmt.Errorf("failed to send delayed message: %v", err)
	}

	fmt.Printf("  ⏳ Sent delayed message, will be delivered at %v\n", deliveryTime.Format("15:04:05"))

	// Try to receive immediately (should be empty)
	_, err = delayedQueue.Receive(ctx)
	if err == nil {
		return fmt.Errorf("expected no message immediately, but got one")
	}

	// Wait for delay and then receive
	time.Sleep(delay + 500*time.Millisecond)

	message, err := delayedQueue.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to receive delayed message: %v", err)
	}

	var receivedMsg TestMessage
	if err := json.Unmarshal(message.Body, &receivedMsg); err != nil {
		return err
	}

	fmt.Printf("  📬 Received delayed message: %s\n", receivedMsg.Content)
	return nil
}

func testBatchProcessing(ctx context.Context) error {
	fmt.Println("\n📦 Testing Batch Processing...")

	config := pkg.DefaultConfig()
	config.QueueName = "test-batch-queue"

	queue, err := pkg.NewQueue(config)
	if err != nil {
		return err
	}
	defer queue.Close()

	batchQueue := pkg.NewBatchQueue(queue.GetRedisClient(), "test-batch", 3, 5*time.Second)

	// Send multiple messages
	for i := 0; i < 5; i++ {
		msg := TestMessage{
			ID:      fmt.Sprintf("batch-msg-%d", i),
			Content: fmt.Sprintf("Batch message %d", i),
		}

		data, _ := json.Marshal(msg)
		if err := batchQueue.Send(ctx, data); err != nil {
			return fmt.Errorf("failed to send batch message: %v", err)
		}
	}

	// Start batch processor
	batchProcessor := func(messages []*pkg.Message) error {
		fmt.Printf("  📦 Processing batch of %d messages:\n", len(messages))
		for _, msg := range messages {
			var testMsg TestMessage
			json.Unmarshal(msg.Body, &testMsg)
			fmt.Printf("    - %s: %s\n", testMsg.ID, testMsg.Content)
		}
		return nil
	}

	// Process batches
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := batchQueue.ProcessBatches(ctx, batchProcessor); err != nil {
		return fmt.Errorf("batch processing failed: %v", err)
	}

	return nil
}

func testDeadLetterQueue(ctx context.Context) error {
	fmt.Println("\n💀 Testing Dead Letter Queue...")

	config := pkg.DefaultConfig()
	config.QueueName = "test-dlq-queue"
	config.EnableDLQ = true
	config.MaxRetries = 2

	queue, err := pkg.NewQueue(config)
	if err != nil {
		return err
	}
	defer queue.Close()

	// Send a message that will fail
	msg := TestMessage{
		ID:      "failing-msg",
		Content: "This message will fail",
	}

	if err := queue.SendJSON(ctx, msg, nil); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	// Consumer that always fails
	failingConsumer := func(message *pkg.Message) error {
		fmt.Printf("  💥 Processing message (will fail): %s\n", string(message.Body))
		return fmt.Errorf("simulated processing error")
	}

	// Consume the message (it will fail and go to DLQ)
	go queue.Consume(ctx, failingConsumer)

	// Wait a bit for retries to complete
	time.Sleep(3 * time.Second)

	// Check DLQ
	dlqInfo, err := queue.GetDLQInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get DLQ info: %v", err)
	}

	if dlqInfo.MessageCount == 0 {
		return fmt.Errorf("expected message in DLQ, but DLQ is empty")
	}

	fmt.Printf("  ✅ Message successfully moved to DLQ after %d retries\n", config.MaxRetries)
	return nil
}

func testMetrics(ctx context.Context) error {
	fmt.Println("\n📊 Testing Metrics Collection...")

	config := pkg.DefaultConfig()
	config.QueueName = "test-metrics-queue"

	queue, err := pkg.NewQueue(config)
	if err != nil {
		return err
	}
	defer queue.Close()

	// Send some messages
	for i := 0; i < 10; i++ {
		msg := TestMessage{
			ID:      fmt.Sprintf("metrics-msg-%d", i),
			Content: fmt.Sprintf("Metrics test message %d", i),
		}

		if err := queue.SendJSON(ctx, msg, nil); err != nil {
			return fmt.Errorf("failed to send message: %v", err)
		}
	}

	// Process some messages
	var processedCount int
	consumer := func(message *pkg.Message) error {
		processedCount++
		return nil
	}

	go queue.Consume(ctx, consumer)
	time.Sleep(2 * time.Second)

	// Get metrics
	metrics := queue.GetMetrics()
	snapshot := metrics.GetSnapshot()

	fmt.Printf("  📈 Messages Sent: %d\n", snapshot.MessagesSent)
	fmt.Printf("  📈 Messages Processed: %d\n", snapshot.MessagesProcessed)
	fmt.Printf("  📈 Processing Rate: %.2f msg/sec\n", snapshot.ProcessingRate)
	fmt.Printf("  📈 Average Processing Time: %v\n", snapshot.AverageProcessingTime)

	if snapshot.MessagesSent != 10 {
		return fmt.Errorf("expected 10 sent messages, got %d", snapshot.MessagesSent)
	}

	return nil
}

func testCircuitBreaker(ctx context.Context) error {
	fmt.Println("\n⚡ Testing Circuit Breaker...")

	config := pkg.CircuitBreakerConfig{
		Name:         "test-circuit",
		MaxFailures:  3,
		ResetTimeout: 5 * time.Second,
	}

	cb := pkg.NewCircuitBreaker(config)

	// Function that always fails
	failingFunction := func() error {
		return fmt.Errorf("simulated failure")
	}

	// Test circuit opening after failures
	for i := 0; i < 5; i++ {
		err := cb.Execute(ctx, failingFunction)
		if i < 3 {
			if err == nil {
				return fmt.Errorf("expected function to fail, but it succeeded")
			}
			fmt.Printf("  💥 Failure %d: %v\n", i+1, err)
		} else {
			if err == nil {
				return fmt.Errorf("expected circuit to be open, but function executed")
			}
			fmt.Printf("  🚫 Circuit breaker OPEN: %v\n", err)
		}
	}

	stats := cb.Stats()
	if stats.State != pkg.CircuitOpen {
		return fmt.Errorf("expected circuit to be OPEN, but it's %s", stats.State)
	}

	fmt.Printf("  ✅ Circuit breaker opened after %d failures\n", stats.ConsecutiveErrors)
	return nil
}

func testRateLimiting(ctx context.Context) error {
	fmt.Println("\n🎛️ Testing Rate Limiting...")

	// Test token bucket
	tokenBucket := pkg.NewTokenBucket(5, 2) // 5 tokens, refill 2 per second

	fmt.Println("  🪣 Testing Token Bucket (5 tokens, 2/sec refill):")

	// Use all tokens
	for i := 0; i < 5; i++ {
		if !tokenBucket.Allow() {
			return fmt.Errorf("expected token to be available, but was denied")
		}
		fmt.Printf("    ✅ Request %d allowed\n", i+1)
	}

	// Next request should be denied
	if tokenBucket.Allow() {
		return fmt.Errorf("expected request to be denied, but was allowed")
	}
	fmt.Printf("    🚫 Request 6 denied (bucket empty)\n")

	// Test sliding window
	slidingWindow := pkg.NewSlidingWindow(3, 5*time.Second) // 3 requests per 5 seconds

	fmt.Println("  🪟 Testing Sliding Window (3 requests per 5 seconds):")

	for i := 0; i < 3; i++ {
		if !slidingWindow.Allow() {
			return fmt.Errorf("expected request to be allowed, but was denied")
		}
		fmt.Printf("    ✅ Request %d allowed\n", i+1)
	}

	// Next request should be denied
	if slidingWindow.Allow() {
		return fmt.Errorf("expected request to be denied, but was allowed")
	}
	fmt.Printf("    🚫 Request 4 denied (window full)\n")

	return nil
}

func testEncryption(ctx context.Context) error {
	fmt.Println("\n🔐 Testing Message Encryption...")

	// Generate encryption key
	key, err := pkg.GenerateEncryptionKey()
	if err != nil {
		return fmt.Errorf("failed to generate encryption key: %v", err)
	}

	config := pkg.EncryptionConfig{
		Enabled:   true,
		Key:       key,
		Algorithm: "AES-256-GCM",
	}

	encryption, err := pkg.NewMessageEncryption(config)
	if err != nil {
		return fmt.Errorf("failed to create encryption handler: %v", err)
	}

	// Test message encryption
	originalMessage := "This is a secret message that should be encrypted!"
	plaintext := []byte(originalMessage)

	fmt.Printf("  📝 Original message: %s\n", originalMessage)

	// Encrypt
	encrypted, err := encryption.Encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %v", err)
	}

	fmt.Printf("  🔒 Encrypted message length: %d bytes\n", len(encrypted))

	// Decrypt
	decrypted, err := encryption.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt message: %v", err)
	}

	decryptedMessage := string(decrypted)
	fmt.Printf("  🔓 Decrypted message: %s\n", decryptedMessage)

	if decryptedMessage != originalMessage {
		return fmt.Errorf("decrypted message doesn't match original")
	}

	stats := encryption.Stats()
	fmt.Printf("  📊 Encryption stats: %s, Key Hash: %s\n", stats.Algorithm, stats.KeyHash)

	return nil
}

func testScheduling(ctx context.Context) error {
	fmt.Println("\n📅 Testing Message Scheduling...")

	config := pkg.DefaultConfig()
	config.QueueName = "test-scheduling-queue"

	queue, err := pkg.NewQueue(config)
	if err != nil {
		return err
	}
	defer queue.Close()

	scheduler := pkg.NewMessageScheduler(queue.GetRedisClient(), config.QueueName)
	if err := scheduler.Start(); err != nil {
		return fmt.Errorf("failed to start scheduler: %v", err)
	}
	defer scheduler.Stop()

	// Schedule a message for 3 seconds from now
	msg := TestMessage{
		ID:      "scheduled-msg",
		Content: "This message was scheduled",
	}

	payload, _ := json.Marshal(msg)
	scheduledMsg := &pkg.RedisScheduledMessage{
		QueueName: config.QueueName,
		Payload:   payload,
		DeliverAt: time.Now().Add(3 * time.Second),
		Priority:  1,
	}

	if err := scheduler.ScheduleMessage(ctx, scheduledMsg); err != nil {
		return fmt.Errorf("failed to schedule message: %v", err)
	}

	fmt.Printf("  ⏰ Scheduled message for delivery at %v\n", scheduledMsg.DeliverAt.Format("15:04:05"))

	// Schedule a recurring message
	recurringMsg := &pkg.RecurringSchedule{
		QueueName: config.QueueName,
		Payload:   payload,
		Interval:  2 * time.Second,
		MaxRuns:   3,
	}

	if err := scheduler.ScheduleRecurring(ctx, recurringMsg); err != nil {
		return fmt.Errorf("failed to schedule recurring message: %v", err)
	}

	fmt.Printf("  🔄 Scheduled recurring message (every 2s, max 3 runs)\n")

	// Wait and check stats
	time.Sleep(6 * time.Second)

	stats := scheduler.Stats(ctx)
	fmt.Printf("  📊 Scheduler stats: %d scheduled, %d recurring\n",
		stats.ScheduledMessages, stats.RecurringSchedules)

	return nil
}

func testTracing(ctx context.Context) error {
	fmt.Println("\n🔍 Testing Message Tracing...")

	config := pkg.DefaultConfig()
	config.QueueName = "test-tracing-queue"

	queue, err := pkg.NewQueue(config)
	if err != nil {
		return err
	}
	defer queue.Close()

	tracingConfig := pkg.TracingConfig{
		Enabled:        true,
		SampleRate:     1.0,
		MaxTraceAge:    time.Hour,
		IncludePayload: true,
	}

	tracing := pkg.NewMessageTracing(queue.GetRedisClient(), config.QueueName, tracingConfig)

	// Start a trace
	messageID := "trace-test-msg"
	traceID, err := tracing.StartTrace(ctx, messageID, map[string]interface{}{
		"test": "tracing",
	})
	if err != nil {
		return fmt.Errorf("failed to start trace: %v", err)
	}

	fmt.Printf("  🏷️ Started trace: %s\n", traceID)

	// Add some events
	events := []pkg.TraceEvent{
		{Event: "message_processing_started"},
		{Event: "validation_completed", Duration: 10 * time.Millisecond},
		{Event: "business_logic_executed", Duration: 50 * time.Millisecond},
		{Event: "message_processed", Duration: 75 * time.Millisecond},
	}

	for _, event := range events {
		if err := tracing.AddEvent(ctx, messageID, traceID, event); err != nil {
			return fmt.Errorf("failed to add trace event: %v", err)
		}
		fmt.Printf("    📌 Added event: %s\n", event.Event)
		time.Sleep(20 * time.Millisecond) // Simulate processing time
	}

	// Get the complete trace
	trace, err := tracing.GetTrace(ctx, messageID)
	if err != nil {
		return fmt.Errorf("failed to get trace: %v", err)
	}

	fmt.Printf("  📋 Trace completed: %d events, status: %s, duration: %v\n",
		len(trace.Events), trace.Status, trace.TotalDuration)

	// Analyze performance
	analysis, err := tracing.AnalyzePerformance(ctx, time.Minute)
	if err != nil {
		return fmt.Errorf("failed to analyze performance: %v", err)
	}

	fmt.Printf("  📈 Performance analysis: %.1f%% success rate, %.2f msg/sec throughput\n",
		analysis.SuccessRate*100, analysis.Throughput)

	return nil
}

func testHighThroughput(ctx context.Context) error {
	fmt.Println("\n🚀 Testing High Throughput Performance...")

	config := pkg.DefaultConfig()
	config.QueueName = "test-high-throughput"
	config.BatchSize = 100

	queue, err := pkg.NewQueue(config)
	if err != nil {
		return err
	}
	defer queue.Close()

	const numMessages = 1000
	const numWorkers = 10

	fmt.Printf("  📊 Sending %d messages with %d workers...\n", numMessages, numWorkers)

	start := time.Now()

	// Send messages concurrently
	var sendWg sync.WaitGroup
	messageChan := make(chan int, numMessages)

	// Fill message channel
	for i := 0; i < numMessages; i++ {
		messageChan <- i
	}
	close(messageChan)

	// Start sender workers
	for w := 0; w < numWorkers; w++ {
		sendWg.Add(1)
		go func(workerID int) {
			defer sendWg.Done()
			for msgID := range messageChan {
				msg := TestMessage{
					ID:        fmt.Sprintf("high-throughput-msg-%d", msgID),
					Content:   fmt.Sprintf("Worker %d message %d", workerID, msgID),
					Timestamp: time.Now(),
				}

				if err := queue.SendJSON(ctx, msg, nil); err != nil {
					log.Printf("Failed to send message %d: %v", msgID, err)
				}
			}
		}(w)
	}

	sendWg.Wait()
	sendDuration := time.Since(start)

	fmt.Printf("  ✅ Sent %d messages in %v (%.2f msg/sec)\n",
		numMessages, sendDuration, float64(numMessages)/sendDuration.Seconds())

	// Consume messages concurrently
	var processedCount int64
	var processWg sync.WaitGroup
	var mu sync.Mutex

	consumer := func(message *pkg.Message) error {
		mu.Lock()
		processedCount++
		current := processedCount
		mu.Unlock()

		if current%100 == 0 {
			fmt.Printf("  📈 Processed %d messages...\n", current)
		}

		if current >= numMessages {
			processWg.Done()
		}
		return nil
	}

	processWg.Add(1)
	start = time.Now()

	// Start multiple consumers
	for w := 0; w < numWorkers; w++ {
		go queue.Consume(ctx, consumer)
	}

	// Wait for all messages to be processed
	done := make(chan bool)
	go func() {
		processWg.Wait()
		done <- true
	}()

	select {
	case <-done:
		processDuration := time.Since(start)
		fmt.Printf("  ✅ Processed %d messages in %v (%.2f msg/sec)\n",
			processedCount, processDuration, float64(processedCount)/processDuration.Seconds())
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout: only processed %d out of %d messages", processedCount, numMessages)
	}

	// Check final metrics
	metrics := queue.GetMetrics()
	snapshot := metrics.GetSnapshot()

	fmt.Printf("  📊 Final metrics - Sent: %d, Processed: %d, Rate: %.2f msg/sec\n",
		snapshot.MessagesSent, snapshot.MessagesProcessed, snapshot.ProcessingRate)

	return nil
}
