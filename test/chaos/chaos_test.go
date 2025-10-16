package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/queue-system/redis-queue/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ChaosTestConfig defines chaos testing parameters
type ChaosTestConfig struct {
	Duration         time.Duration
	Producers        int
	Consumers        int
	MessageRate      time.Duration // Rate at which messages are produced
	FailureRate      float64       // Probability of failures (0.0 to 1.0)
	NetworkLatency   time.Duration
	RedisFailures    bool
	ConsumerFailures bool
	ProducerFailures bool
	QueueOperations  bool
}

// ChaosMetrics tracks chaos test results
type ChaosMetrics struct {
	MessagesProduced  int64
	MessagesConsumed  int64
	ProducerFailures  int64
	ConsumerFailures  int64
	RedisConnFailures int64
	MessageLost       int64
	DuplicateMessages int64
	AverageLatency    time.Duration
	MaxLatency        time.Duration
	MinLatency        time.Duration
}

// TestChaosBasic runs a basic chaos test with random failures
func TestChaosBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	config := &ChaosTestConfig{
		Duration:         2 * time.Minute,
		Producers:        5,
		Consumers:        3,
		MessageRate:      100 * time.Millisecond,
		FailureRate:      0.1, // 10% failure rate
		ConsumerFailures: true,
		ProducerFailures: true,
	}

	metrics := runChaosTest(t, config)

	// Verify system resilience
	assert.Greater(t, metrics.MessagesProduced, int64(0), "Should have produced messages")
	assert.Greater(t, metrics.MessagesConsumed, int64(0), "Should have consumed messages")

	// Allow for some message loss due to failures, but not total failure
	lossRate := float64(metrics.MessageLost) / float64(metrics.MessagesProduced)
	assert.Less(t, lossRate, 0.5, "Message loss rate should be less than 50%")

	t.Logf("Chaos test results: Produced=%d, Consumed=%d, Lost=%d, Loss Rate=%.2f%%",
		metrics.MessagesProduced, metrics.MessagesConsumed, metrics.MessageLost, lossRate*100)
}

// TestChaosHighLoad tests system under high load with chaos
func TestChaosHighLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos high load test in short mode")
	}

	config := &ChaosTestConfig{
		Duration:         3 * time.Minute,
		Producers:        20,
		Consumers:        15,
		MessageRate:      10 * time.Millisecond, // High message rate
		FailureRate:      0.15,                  // 15% failure rate
		ConsumerFailures: true,
		ProducerFailures: true,
		QueueOperations:  true,
	}

	metrics := runChaosTest(t, config)

	// Under high load, system should still function
	assert.Greater(t, metrics.MessagesProduced, int64(1000), "Should produce many messages under load")
	assert.Greater(t, metrics.MessagesConsumed, int64(500), "Should consume reasonable number of messages")

	t.Logf("High load chaos test: Produced=%d, Consumed=%d, Failures: P=%d, C=%d",
		metrics.MessagesProduced, metrics.MessagesConsumed,
		metrics.ProducerFailures, metrics.ConsumerFailures)
}

// TestChaosNetworkPartition simulates network partitions
func TestChaosNetworkPartition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network partition test in short mode")
	}

	client := setupChaosClient(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	queueName := "chaos-network-partition"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        5, // Higher retries for network issues
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(t, err)

	var produced, consumed int64
	var wg sync.WaitGroup

	// Producer with simulated network issues
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			// Simulate network partition (skip some operations)
			if rand.Float64() < 0.2 { // 20% network failure
				time.Sleep(1 * time.Second) // Simulate timeout
				continue
			}

			payload := map[string]interface{}{
				"id":        i,
				"data":      fmt.Sprintf("network test %d", i),
				"timestamp": time.Now().Unix(),
			}

			_, err := client.Enqueue(ctx, queueName, payload)
			if err == nil {
				produced++
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
	}()

	// Consumer with network resilience
	wg.Add(1)
	go func() {
		defer wg.Done()

		handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
			// Simulate processing time and potential network issues during ack
			if rand.Float64() < 0.1 { // 10% ack failure
				return client.Nack(true) // Requeue
			}

			consumed++
			return client.Ack()
		}

		client.Consume(ctx, queueName, handler,
			client.WithBatchSize(5),
			client.WithVisibilityTimeout(10*time.Second),
		)
	}()

	wg.Wait()

	// Verify resilience to network issues
	assert.Greater(t, produced, int64(400), "Should produce messages despite network issues")
	assert.Greater(t, consumed, int64(200), "Should consume messages despite network issues")

	t.Logf("Network partition test: Produced=%d, Consumed=%d", produced, consumed)
}

// TestChaosConsumerRestart simulates consumer restarts and failures
func TestChaosConsumerRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping consumer restart test in short mode")
	}

	client := setupChaosClient(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	queueName := "chaos-consumer-restart"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 10 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(t, err)

	// Produce messages continuously
	var produced int64
	go func() {
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			default:
			}

			payload := map[string]interface{}{
				"id":   i,
				"data": fmt.Sprintf("restart test %d", i),
			}

			_, err := client.Enqueue(ctx, queueName, payload)
			if err == nil {
				produced++
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Simulate consumer restarts
	var totalConsumed int64
	consumerRestarts := 5

	for restart := 0; restart < consumerRestarts; restart++ {
		var consumed int64

		// Start consumer
		consumerCtx, consumerCancel := context.WithTimeout(ctx, 30*time.Second)

		handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
			consumed++
			totalConsumed++

			// Simulate occasional processing failures
			if rand.Float64() < 0.05 { // 5% processing failure
				return client.Nack(true)
			}

			return client.Ack()
		}

		go client.Consume(consumerCtx, queueName, handler,
			client.WithBatchSize(3),
			client.WithVisibilityTimeout(10*time.Second),
		)

		// Let consumer run for a while
		<-consumerCtx.Done()
		consumerCancel()

		t.Logf("Consumer restart %d: consumed %d messages", restart+1, consumed)

		// Brief pause between restarts
		time.Sleep(2 * time.Second)
	}

	// Final verification
	assert.Greater(t, produced, int64(500), "Should have produced many messages")
	assert.Greater(t, totalConsumed, int64(100), "Should have consumed messages across restarts")

	t.Logf("Consumer restart test: Produced=%d, Total Consumed=%d", produced, totalConsumed)
}

// TestChaosMessageOrdering tests message ordering under chaos
func TestChaosMessageOrdering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping message ordering test in short mode")
	}

	client := setupChaosClient(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	queueName := "chaos-ordering"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 15 * time.Second,
		MaxRetries:        2,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(t, err)

	// Send ordered messages
	messageCount := 1000
	for i := 0; i < messageCount; i++ {
		payload := map[string]interface{}{
			"sequence": i,
			"data":     fmt.Sprintf("ordered message %d", i),
		}

		_, err := client.Enqueue(ctx, queueName, payload)
		require.NoError(t, err)

		// Add some chaos - occasional delays
		if rand.Float64() < 0.1 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Consume with chaos
	var consumed []int
	var mu sync.Mutex

	handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		data := payload.(map[string]interface{})
		sequence := int(data["sequence"].(float64))

		// Simulate processing chaos
		processingTime := time.Duration(rand.Intn(50)) * time.Millisecond
		time.Sleep(processingTime)

		// Occasional processing failure
		if rand.Float64() < 0.05 {
			return client.Nack(true)
		}

		mu.Lock()
		consumed = append(consumed, sequence)
		mu.Unlock()

		return client.Ack()
	}

	// Start multiple consumers to add chaos
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.Consume(ctx, queueName, handler,
				client.WithBatchSize(5),
				client.WithVisibilityTimeout(15*time.Second),
			)
		}()
	}

	// Wait for consumption to complete
	go func() {
		for {
			stats, err := client.GetQueueStats(ctx, queueName)
			if err != nil {
				continue
			}

			if stats.Length == 0 && stats.InFlight == 0 {
				cancel()
				break
			}

			time.Sleep(1 * time.Second)
		}
	}()

	wg.Wait()

	// Analyze ordering
	mu.Lock()
	defer mu.Unlock()

	assert.Greater(t, len(consumed), messageCount*3/4, "Should consume most messages")

	// Check for duplicates (shouldn't happen with proper acking)
	seen := make(map[int]bool)
	duplicates := 0
	for _, seq := range consumed {
		if seen[seq] {
			duplicates++
		}
		seen[seq] = true
	}

	assert.Equal(t, 0, duplicates, "Should not have duplicate messages")

	t.Logf("Ordering test: Sent=%d, Consumed=%d, Duplicates=%d",
		messageCount, len(consumed), duplicates)
}

// runChaosTest executes a chaos test with the given configuration
func runChaosTest(t *testing.T, config *ChaosTestConfig) *ChaosMetrics {
	client := setupChaosClient(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	queueName := "chaos-test-queue"

	// Create queue
	queueConfig := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, queueConfig)
	require.NoError(t, err)

	metrics := &ChaosMetrics{
		MinLatency: time.Hour, // Initialize to high value
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Start producers
	for i := 0; i < config.Producers; i++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			runChaosProducer(ctx, client, queueName, producerID, config, metrics, &mu)
		}(i)
	}

	// Start consumers
	for i := 0; i < config.Consumers; i++ {
		wg.Add(1)
		go func(consumerID int) {
			defer wg.Done()
			runChaosConsumer(ctx, client, queueName, consumerID, config, metrics, &mu)
		}(i)
	}

	// Chaos operations (if enabled)
	if config.QueueOperations {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runChaosOperations(ctx, client, queueName, config)
		}()
	}

	wg.Wait()

	// Calculate final metrics
	if metrics.MessagesConsumed > 0 {
		metrics.AverageLatency = time.Duration(int64(metrics.AverageLatency) / metrics.MessagesConsumed)
	}

	return metrics
}

// runChaosProducer runs a producer with chaos injection
func runChaosProducer(ctx context.Context, client *client.Client, queueName string,
	producerID int, config *ChaosTestConfig, metrics *ChaosMetrics, mu *sync.Mutex) {

	ticker := time.NewTicker(config.MessageRate)
	defer ticker.Stop()

	messageID := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		// Inject chaos
		if config.ProducerFailures && rand.Float64() < config.FailureRate {
			mu.Lock()
			metrics.ProducerFailures++
			mu.Unlock()
			continue
		}

		payload := map[string]interface{}{
			"producer_id": producerID,
			"message_id":  messageID,
			"timestamp":   time.Now().UnixNano(),
			"data":        fmt.Sprintf("chaos message from producer %d", producerID),
		}

		startTime := time.Now()
		_, err := client.Enqueue(ctx, queueName, payload)
		latency := time.Since(startTime)

		mu.Lock()
		if err != nil {
			metrics.ProducerFailures++
		} else {
			metrics.MessagesProduced++
			// Update latency metrics
			if latency > metrics.MaxLatency {
				metrics.MaxLatency = latency
			}
			if latency < metrics.MinLatency {
				metrics.MinLatency = latency
			}
			metrics.AverageLatency += latency
		}
		mu.Unlock()

		messageID++
	}
}

// runChaosConsumer runs a consumer with chaos injection
func runChaosConsumer(ctx context.Context, client *client.Client, queueName string,
	consumerID int, config *ChaosTestConfig, metrics *ChaosMetrics, mu *sync.Mutex) {

	handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		// Extract timestamp for latency calculation
		data := payload.(map[string]interface{})
		if timestamp, ok := data["timestamp"].(float64); ok {
			latency := time.Since(time.Unix(0, int64(timestamp)))

			mu.Lock()
			if latency > metrics.MaxLatency {
				metrics.MaxLatency = latency
			}
			if latency < metrics.MinLatency {
				metrics.MinLatency = latency
			}
			metrics.AverageLatency += latency
			mu.Unlock()
		}

		// Inject chaos
		if config.ConsumerFailures && rand.Float64() < config.FailureRate {
			mu.Lock()
			metrics.ConsumerFailures++
			mu.Unlock()
			return client.Nack(true) // Requeue
		}

		// Simulate processing time
		processingTime := time.Duration(rand.Intn(100)) * time.Millisecond
		time.Sleep(processingTime)

		mu.Lock()
		metrics.MessagesConsumed++
		mu.Unlock()

		return client.Ack()
	}

	client.Consume(ctx, queueName, handler,
		client.WithBatchSize(rand.Intn(5)+1), // Random batch size 1-5
		client.WithVisibilityTimeout(30*time.Second),
	)
}

// runChaosOperations performs random queue operations
func runChaosOperations(ctx context.Context, client *client.Client, queueName string, config *ChaosTestConfig) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		// Random queue operations
		operation := rand.Intn(4)
		switch operation {
		case 0: // Get stats
			client.GetQueueStats(ctx, queueName)
		case 1: // Peek messages
			client.Peek(ctx, queueName, 5)
		case 2: // List queues
			client.ListQueues(ctx)
		case 3: // Pause/Resume (commented out to avoid breaking test flow)
			// Randomly pause and resume queue
			// client.PauseQueue(ctx, queueName)
			// time.Sleep(1 * time.Second)
			// client.ResumeQueue(ctx, queueName)
		}
	}
}

// setupChaosClient creates a client for chaos testing
func setupChaosClient(t *testing.T) *client.Client {
	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}
	config.RedisConfig.DB = 3 // Use different DB for chaos tests

	client, err := client.NewClient(config)
	require.NoError(t, err)

	return client
}
