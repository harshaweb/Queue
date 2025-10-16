package benchmark

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/queue-system/redis-queue/pkg/client"
	"github.com/stretchr/testify/require"
)

// BenchmarkEnqueue measures enqueue performance
func BenchmarkEnqueue(b *testing.B) {
	client := setupBenchmarkClient(b)
	defer client.Close()

	ctx := context.Background()
	queueName := "benchmark-enqueue"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(b, err)

	payload := map[string]interface{}{
		"id":        1,
		"data":      "benchmark data",
		"timestamp": time.Now().Unix(),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Enqueue(ctx, queueName, payload)
			if err != nil {
				b.Fatalf("Enqueue failed: %v", err)
			}
		}
	})
}

// BenchmarkBatchEnqueue measures batch enqueue performance
func BenchmarkBatchEnqueue(b *testing.B) {
	client := setupBenchmarkClient(b)
	defer client.Close()

	ctx := context.Background()
	queueName := "benchmark-batch-enqueue"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(b, err)

	// Prepare batch of messages
	batchSizes := []int{10, 50, 100, 500}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize%d", batchSize), func(b *testing.B) {
			payloads := make([]interface{}, batchSize)
			for i := 0; i < batchSize; i++ {
				payloads[i] = map[string]interface{}{
					"id":        i,
					"data":      fmt.Sprintf("batch data %d", i),
					"timestamp": time.Now().Unix(),
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := client.BatchEnqueue(ctx, queueName, payloads)
				if err != nil {
					b.Fatalf("BatchEnqueue failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkDequeue measures dequeue performance
func BenchmarkDequeue(b *testing.B) {
	client := setupBenchmarkClient(b)
	defer client.Close()

	ctx := context.Background()
	queueName := "benchmark-dequeue"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(b, err)

	// Pre-populate queue with messages
	payload := map[string]interface{}{
		"id":   1,
		"data": "benchmark data",
	}

	// Add many messages for the benchmark
	for i := 0; i < b.N*2; i++ {
		_, err := client.Enqueue(ctx, queueName, payload)
		require.NoError(b, err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			messages, err := client.Dequeue(ctx, queueName, 1)
			if err != nil {
				b.Fatalf("Dequeue failed: %v", err)
			}
			if len(messages) > 0 {
				// Ack the message
				err = client.Ack(ctx, queueName, messages[0].ID)
				if err != nil {
					b.Fatalf("Ack failed: %v", err)
				}
			}
		}
	})
}

// BenchmarkConcurrentProducerConsumer tests concurrent production and consumption
func BenchmarkConcurrentProducerConsumer(b *testing.B) {
	client := setupBenchmarkClient(b)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	queueName := "benchmark-concurrent"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(b, err)

	configurations := []struct {
		name      string
		producers int
		consumers int
		messages  int
	}{
		{"1P1C", 1, 1, 1000},
		{"5P5C", 5, 5, 5000},
		{"10P10C", 10, 10, 10000},
		{"20P5C", 20, 5, 10000},
		{"5P20C", 5, 20, 10000},
	}

	for _, config := range configurations {
		b.Run(config.name, func(b *testing.B) {
			benchmarkConcurrentWorkload(b, client, queueName, config.producers, config.consumers, config.messages)
		})
	}
}

func benchmarkConcurrentWorkload(b *testing.B, client *client.Client, queueName string, producers, consumers, totalMessages int) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var wg sync.WaitGroup

	// Metrics
	produced := make(chan int, totalMessages)
	consumed := make(chan int, totalMessages)

	b.ResetTimer()

	// Start producers
	messagesPerProducer := totalMessages / producers
	for i := 0; i < producers; i++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()
			for j := 0; j < messagesPerProducer; j++ {
				payload := map[string]interface{}{
					"producer_id": producerID,
					"message_id":  j,
					"data":        fmt.Sprintf("message from producer %d", producerID),
					"timestamp":   time.Now().Unix(),
				}

				_, err := client.Enqueue(ctx, queueName, payload)
				if err != nil {
					b.Errorf("Producer %d failed to enqueue: %v", producerID, err)
					return
				}
				produced <- 1
			}
		}(i)
	}

	// Start consumers
	for i := 0; i < consumers; i++ {
		wg.Add(1)
		go func(consumerID int) {
			defer wg.Done()

			handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
				consumed <- 1
				return client.Ack()
			}

			client.Consume(ctx, queueName, handler,
				client.WithBatchSize(10),
				client.WithVisibilityTimeout(30*time.Second),
			)
		}(i)
	}

	// Wait for completion or timeout
	go func() {
		wg.Wait()
		cancel()
	}()

	// Monitor progress
	producedCount := 0
	consumedCount := 0

	for {
		select {
		case <-produced:
			producedCount++
		case <-consumed:
			consumedCount++
			if consumedCount >= totalMessages {
				cancel()
				return
			}
		case <-ctx.Done():
			b.Logf("Benchmark completed - Produced: %d, Consumed: %d", producedCount, consumedCount)
			return
		}
	}
}

// BenchmarkQueueOperations tests various queue operations performance
func BenchmarkQueueOperations(b *testing.B) {
	client := setupBenchmarkClient(b)
	defer client.Close()

	ctx := context.Background()
	queueName := "benchmark-operations"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(b, err)

	// Pre-populate with messages
	for i := 0; i < 1000; i++ {
		payload := map[string]interface{}{
			"id":   i,
			"data": fmt.Sprintf("message %d", i),
		}
		_, err := client.Enqueue(ctx, queueName, payload)
		require.NoError(b, err)
	}

	b.Run("GetStats", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.GetQueueStats(ctx, queueName)
			if err != nil {
				b.Fatalf("GetStats failed: %v", err)
			}
		}
	})

	b.Run("ListQueues", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.ListQueues(ctx)
			if err != nil {
				b.Fatalf("ListQueues failed: %v", err)
			}
		}
	})

	b.Run("Peek", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Peek(ctx, queueName, 10)
			if err != nil {
				b.Fatalf("Peek failed: %v", err)
			}
		}
	})
}

// BenchmarkScheduledMessages tests scheduled message performance
func BenchmarkScheduledMessages(b *testing.B) {
	client := setupBenchmarkClient(b)
	defer client.Close()

	ctx := context.Background()
	queueName := "benchmark-scheduled"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(b, err)

	delays := []time.Duration{
		1 * time.Second,
		5 * time.Second,
		30 * time.Second,
		1 * time.Minute,
	}

	for _, delay := range delays {
		b.Run(fmt.Sprintf("Delay%v", delay), func(b *testing.B) {
			payload := map[string]interface{}{
				"scheduled": true,
				"delay":     delay.String(),
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := client.Enqueue(ctx, queueName, payload,
					client.WithDelay(delay),
				)
				if err != nil {
					b.Fatalf("Scheduled enqueue failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkLargePayloads tests performance with different payload sizes
func BenchmarkLargePayloads(b *testing.B) {
	client := setupBenchmarkClient(b)
	defer client.Close()

	ctx := context.Background()
	queueName := "benchmark-large-payloads"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(b, err)

	payloadSizes := []int{
		1024,    // 1KB
		10240,   // 10KB
		102400,  // 100KB
		1048576, // 1MB
	}

	for _, size := range payloadSizes {
		b.Run(fmt.Sprintf("Size%dB", size), func(b *testing.B) {
			// Create payload of specified size
			data := make([]byte, size)
			for i := range data {
				data[i] = byte(i % 256)
			}

			payload := map[string]interface{}{
				"size": size,
				"data": string(data),
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := client.Enqueue(ctx, queueName, payload)
				if err != nil {
					b.Fatalf("Large payload enqueue failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkMemoryUsage tracks memory usage during operations
func BenchmarkMemoryUsage(b *testing.B) {
	client := setupBenchmarkClient(b)
	defer client.Close()

	ctx := context.Background()
	queueName := "benchmark-memory"

	// Create queue
	config := &client.QueueConfig{
		Name:              queueName,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
	err := client.CreateQueue(ctx, config)
	require.NoError(b, err)

	b.Run("MemoryUsage", func(b *testing.B) {
		payload := map[string]interface{}{
			"id":   1,
			"data": "memory test data",
		}

		var messages []string

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := client.Enqueue(ctx, queueName, payload)
			if err != nil {
				b.Fatalf("Enqueue failed: %v", err)
			}
			messages = append(messages, result.MessageID)

			// Periodically clean up to avoid memory buildup
			if i%1000 == 0 && len(messages) > 0 {
				// Dequeue and ack some messages
				for j := 0; j < 100 && j < len(messages); j++ {
					msgs, err := client.Dequeue(ctx, queueName, 1)
					if err == nil && len(msgs) > 0 {
						client.Ack(ctx, queueName, msgs[0].ID)
					}
				}
			}
		}
	})
}

// setupBenchmarkClient creates a client for benchmarking
func setupBenchmarkClient(b *testing.B) *client.Client {
	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}
	config.RedisConfig.DB = 2 // Use different DB for benchmarks

	client, err := client.NewClient(config)
	require.NoError(b, err)

	return client
}

// Helper function to run a simple throughput test
func runThroughputTest(b *testing.B, client *client.Client, queueName string, duration time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var produced, consumed int64
	var wg sync.WaitGroup

	// Producer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				payload := map[string]interface{}{
					"id":        produced,
					"timestamp": time.Now().Unix(),
				}
				_, err := client.Enqueue(ctx, queueName, payload)
				if err == nil {
					produced++
				}
			}
		}
	}()

	// Consumer
	wg.Add(1)
	go func() {
		defer wg.Done()

		handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
			consumed++
			return client.Ack()
		}

		client.Consume(ctx, queueName, handler,
			client.WithBatchSize(10),
			client.WithVisibilityTimeout(30*time.Second),
		)
	}()

	wg.Wait()

	b.Logf("Throughput test completed - Produced: %d, Consumed: %d, Duration: %v",
		produced, consumed, duration)
	b.Logf("Production rate: %.2f msg/sec, Consumption rate: %.2f msg/sec",
		float64(produced)/duration.Seconds(),
		float64(consumed)/duration.Seconds())
}
