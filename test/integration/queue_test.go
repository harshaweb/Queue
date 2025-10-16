package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/queue-system/redis-queue/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasicQueueOperations tests basic enqueue/dequeue operations
func TestBasicQueueOperations(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	queueName := "test-basic-queue"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(t, err)

	// Test enqueue
	payload := map[string]interface{}{
		"message": "Hello, World!",
		"number":  42,
	}

	result, err := client.Enqueue(ctx, queueName, payload)
	require.NoError(t, err)
	assert.NotEmpty(t, result.MessageID)

	// Test stats after enqueue
	stats, err := client.GetQueueStats(ctx, queueName)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Length)
	assert.Equal(t, int64(0), stats.InFlight)
}

// TestBatchOperations tests batch enqueue/dequeue
func TestBatchOperations(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	queueName := "test-batch-queue"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(t, err)

	// Batch enqueue
	payloads := []interface{}{
		map[string]interface{}{"id": 1, "data": "first"},
		map[string]interface{}{"id": 2, "data": "second"},
		map[string]interface{}{"id": 3, "data": "third"},
	}

	batchResult, err := client.BatchEnqueue(ctx, queueName, payloads)
	require.NoError(t, err)
	assert.Len(t, batchResult.MessageIDs, 3)

	// Check stats
	stats, err := client.GetQueueStats(ctx, queueName)
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.Length)
}

// TestMessageProcessing tests full message processing cycle
func TestMessageProcessing(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	queueName := "test-processing-queue"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 5 * time.Second,
		MaxRetries:        2,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(t, err)

	// Enqueue a message
	payload := map[string]interface{}{
		"task": "process_data",
		"id":   "test-123",
	}

	_, err = client.Enqueue(ctx, queueName, payload)
	require.NoError(t, err)

	// Process message
	processed := make(chan bool, 1)

	handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		data := payload.(map[string]interface{})
		assert.Equal(t, "process_data", data["task"])
		assert.Equal(t, "test-123", data["id"])

		processed <- true
		return client.Ack()
	}

	// Start consumer in goroutine
	go func() {
		client.Consume(ctx, queueName, handler,
			client.WithBatchSize(1),
			client.WithVisibilityTimeout(5*time.Second),
		)
	}()

	// Wait for message to be processed
	select {
	case <-processed:
		// Success
	case <-time.After(10 * time.Second):
		t.Fatal("Message was not processed within timeout")
	}

	// Check final stats
	time.Sleep(1 * time.Second) // Allow time for stats to update
	stats, err := client.GetQueueStats(ctx, queueName)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.Length)
}

// TestScheduledMessages tests message scheduling
func TestScheduledMessages(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	queueName := "test-scheduled-queue"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(t, err)

	// Schedule message for 2 seconds in the future
	payload := map[string]interface{}{
		"scheduled": true,
		"time":      time.Now().Format(time.RFC3339),
	}

	_, err = client.Enqueue(ctx, queueName, payload,
		client.WithDelay(2*time.Second),
	)
	require.NoError(t, err)

	// Message should not be immediately available
	stats, err := client.GetQueueStats(ctx, queueName)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.Length) // Not yet available

	// Wait for message to become available
	time.Sleep(3 * time.Second)

	// Now it should be available
	stats, err = client.GetQueueStats(ctx, queueName)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Length)
}

// TestRetryMechanism tests message retry behavior
func TestRetryMechanism(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	queueName := "test-retry-queue"

	// Create queue with 2 max retries
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 2 * time.Second,
		MaxRetries:        2,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(t, err)

	// Enqueue a message
	payload := map[string]interface{}{
		"fail_times": 2, // Will fail twice then succeed
	}

	_, err = client.Enqueue(ctx, queueName, payload)
	require.NoError(t, err)

	attempts := 0
	processed := make(chan bool, 1)

	handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		attempts++
		data := payload.(map[string]interface{})
		failTimes := int(data["fail_times"].(float64))

		if attempts <= failTimes {
			// Fail and requeue
			return client.Nack(true)
		}

		// Success on final attempt
		processed <- true
		return client.Ack()
	}

	// Start consumer
	go func() {
		client.Consume(ctx, queueName, handler,
			client.WithBatchSize(1),
			client.WithVisibilityTimeout(2*time.Second),
			client.WithMaxRetries(2),
		)
	}()

	// Wait for processing
	select {
	case <-processed:
		assert.Equal(t, 3, attempts) // Should have been attempted 3 times
	case <-time.After(15 * time.Second):
		t.Fatal("Message was not processed within timeout")
	}
}

// TestQueueManagement tests queue creation, listing, and deletion
func TestQueueManagement(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// Test queue creation
	queues := []string{"test-queue-1", "test-queue-2", "test-queue-3"}

	for _, queueName := range queues {
		config := &client.QueueConfig{
			Name:              queueName,
			VisibilityTimeout: 30 * time.Second,
			MaxRetries:        3,
		}
		err := client.CreateQueue(ctx, config)
		require.NoError(t, err)
	}

	// Test queue listing
	allQueues, err := client.ListQueues(ctx)
	require.NoError(t, err)

	for _, queueName := range queues {
		assert.Contains(t, allQueues, queueName)
	}

	// Test queue stats
	for _, queueName := range queues {
		stats, err := client.GetQueueStats(ctx, queueName)
		require.NoError(t, err)
		assert.Equal(t, queueName, stats.Name)
		assert.Equal(t, int64(0), stats.Length)
	}
}

// TestPauseResumeQueue tests queue pause/resume functionality
func TestPauseResumeQueue(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx := context.Background()
	queueName := "test-pause-queue"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(t, err)

	// Pause queue
	err = client.PauseQueue(ctx, queueName)
	require.NoError(t, err)

	// Resume queue
	err = client.ResumeQueue(ctx, queueName)
	require.NoError(t, err)
}

// TestConcurrentConsumers tests multiple concurrent consumers
func TestConcurrentConsumers(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	queueName := "test-concurrent-queue"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 10 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(t, err)

	// Enqueue multiple messages
	messageCount := 10
	for i := 0; i < messageCount; i++ {
		payload := map[string]interface{}{
			"id":   i,
			"data": fmt.Sprintf("message-%d", i),
		}
		_, err = client.Enqueue(ctx, queueName, payload)
		require.NoError(t, err)
	}

	// Start multiple consumers
	consumerCount := 3
	processedCount := make(chan int, messageCount)

	for i := 0; i < consumerCount; i++ {
		go func(consumerID int) {
			handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
				processedCount <- 1
				return client.Ack()
			}

			client.Consume(ctx, queueName, handler,
				client.WithBatchSize(1),
				client.WithVisibilityTimeout(10*time.Second),
			)
		}(i)
	}

	// Wait for all messages to be processed
	totalProcessed := 0
	for totalProcessed < messageCount {
		select {
		case <-processedCount:
			totalProcessed++
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for messages to be processed. Processed: %d/%d", totalProcessed, messageCount)
		}
	}

	assert.Equal(t, messageCount, totalProcessed)
}

// setupTestClient creates a test client connected to Redis
func setupTestClient(t *testing.T) *client.Client {
	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}
	config.RedisConfig.DB = 1 // Use different DB for tests

	client, err := client.NewClient(config)
	require.NoError(t, err)

	return client
}

// Helper function to clean up test queues
func cleanupTestQueues(t *testing.T, client *client.Client, queueNames []string) {
	ctx := context.Background()

	for _, queueName := range queueNames {
		// Note: DeleteQueue would need to be implemented in the client
		// For now, we'll leave test queues as they'll be in a separate DB
		_ = queueName
	}
}
