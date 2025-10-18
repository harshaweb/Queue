package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/harshaweb/queue/pkg"
)

func main() {
	fmt.Println("Queue SDK - High Throughput Example")
	fmt.Println("===================================")

	// Configuration for high throughput
	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"
	config.MaxRetries = 3
	config.BatchSize = 100 // Process messages in batches
	config.EnableMetrics = true

	// Create queue
	q, err := pkg.NewQueue("high-throughput", config)
	if err != nil {
		log.Fatal("Failed to create queue:", err)
	}
	defer q.Close()

	fmt.Println("✅ High-throughput queue created")

	// Performance tracking
	var (
		messagesSent     int64
		messagesReceived int64
		startTime        = time.Now()
	)

	// Send messages at high throughput
	fmt.Println("📤 Sending 10,000 messages...")
	sendStart := time.Now()

	// Send messages concurrently
	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 1000

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < messagesPerGoroutine; j++ {
				message := map[string]interface{}{
					"id":           fmt.Sprintf("msg_%d_%d", goroutineID, j),
					"goroutine_id": goroutineID,
					"message_num":  j,
					"timestamp":    time.Now(),
					"data":         fmt.Sprintf("High throughput test message %d from goroutine %d", j, goroutineID),
				}

				_, err := q.Send(message, nil)
				if err != nil {
					log.Printf("Failed to send message: %v", err)
					continue
				}

				atomic.AddInt64(&messagesSent, 1)

				// Small delay to not overwhelm the system
				if j%100 == 0 {
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()
	sendDuration := time.Since(sendStart)

	fmt.Printf("✅ Sent %d messages in %v\n", messagesSent, sendDuration)
	fmt.Printf("📊 Send throughput: %.2f messages/second\n", float64(messagesSent)/sendDuration.Seconds())

	// Process messages with high concurrency
	fmt.Println("📥 Processing messages with high concurrency...")
	processStart := time.Now()

	// Start multiple consumers
	numConsumers := 5
	var consumerWg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < numConsumers; i++ {
		consumerWg.Add(1)
		go func(consumerID int) {
			defer consumerWg.Done()

			handler := func(ctx context.Context, msg *pkg.Message) error {
				// Simulate some processing
				time.Sleep(1 * time.Millisecond)

				// Track processed messages
				atomic.AddInt64(&messagesReceived, 1)

				received := atomic.LoadInt64(&messagesReceived)
				if received%1000 == 0 {
					fmt.Printf("🔄 Consumer %d: Processed %d messages so far\n", consumerID, received)
				}

				return nil
			}

			// Each consumer processes messages until context is cancelled
			if err := q.Consume(handler, nil); err != nil && err != context.Canceled {
				log.Printf("Consumer %d error: %v", consumerID, err)
			}
		}(i)
	}

	// Wait for processing to complete or timeout
	done := make(chan bool)
	go func() {
		consumerWg.Wait()
		done <- true
	}()

	select {
	case <-done:
		fmt.Println("✅ All consumers finished")
	case <-ctx.Done():
		fmt.Println("⏰ Processing timeout reached")
		cancel()
	}

	processDuration := time.Since(processStart)

	// Final metrics
	fmt.Println("\n📈 Final Performance Metrics")
	fmt.Println("============================")
	
	totalDuration := time.Since(startTime)
	processedCount := atomic.LoadInt64(&messagesReceived)
	
	fmt.Printf("📤 Messages Sent: %d\n", messagesSent)
	fmt.Printf("📥 Messages Processed: %d\n", processedCount)
	fmt.Printf("⏱️  Total Time: %v\n", totalDuration)
	fmt.Printf("⏱️  Send Time: %v\n", sendDuration)
	fmt.Printf("⏱️  Process Time: %v\n", processDuration)
	fmt.Printf("📊 Send Throughput: %.2f msg/sec\n", float64(messagesSent)/sendDuration.Seconds())
	fmt.Printf("📊 Process Throughput: %.2f msg/sec\n", float64(processedCount)/processDuration.Seconds())
	fmt.Printf("📊 Overall Throughput: %.2f msg/sec\n", float64(processedCount)/totalDuration.Seconds())

	// Get queue metrics if available
	if metrics := q.GetMetrics(); metrics != nil {
		fmt.Println("\n📊 Queue Metrics")
		fmt.Println("================")
		fmt.Printf("Messages Sent: %d\n", metrics.MessagesSent)
		fmt.Printf("Messages Processed: %d\n", metrics.MessagesProcessed)
		fmt.Printf("Processing Errors: %d\n", metrics.ProcessingErrors)
		fmt.Printf("Error Rate: %.2f%%\n", metrics.ErrorRate)
		if metrics.AverageProcessTime > 0 {
			fmt.Printf("Average Process Time: %v\n", metrics.AverageProcessTime)
		}
	}

	fmt.Println("\n🎉 High-throughput test completed!")
}