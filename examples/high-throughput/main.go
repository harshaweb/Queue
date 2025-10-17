package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	queue "github.com/harshaweb/Queue"
)

func main() {
	fmt.Println("Queue SDK - High Throughput Example")
	fmt.Println("===================================")

	// Configuration for high throughput
	config := &queue.Config{
		RedisAddress:      "localhost:6379",
		DefaultTimeout:    30 * time.Second,
		DefaultMaxRetries: 3,
		BatchSize:         100, // Process 100 messages at once
	}

	// Create worker pool with 10 concurrent workers
	pool, err := queue.NewWorkerPool("high-throughput", 10, config)
	if err != nil {
		log.Fatal("Failed to create worker pool:", err)
	}
	defer pool.Stop()

	fmt.Println("✅ Worker pool created with 10 workers")

	// Create a separate queue instance for sending messages
	sender, err := queue.New("high-throughput", config)
	if err != nil {
		log.Fatal("Failed to create sender queue:", err)
	}
	defer sender.Close()

	// Send a batch of messages
	fmt.Println("📤 Sending batch of messages...")

	var messages []map[string]any
	for i := 0; i < 1000; i++ {
		messages = append(messages, map[string]interface{}{
			"order_id":    i + 1,
			"customer_id": (i % 100) + 1,
			"amount":      float64(i+1) * 10.50,
			"timestamp":   time.Now().Format(time.RFC3339),
		})
	}

	// Send messages in batches
	batchSize := 50
	var wg sync.WaitGroup

	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}

		wg.Add(1)
		go func(batch []map[string]interface{}) {
			defer wg.Done()

			ids, err := sender.BatchSend(batch)
			if err != nil {
				log.Printf("Failed to send batch: %v", err)
				return
			}

			fmt.Printf("✅ Sent batch of %d messages (IDs: %s...%s)\n",
				len(batch), ids[0][:8], ids[len(ids)-1][:8])
		}(messages[i:end])
	}

	wg.Wait()
	fmt.Printf("✅ All %d messages sent successfully\n", len(messages))

	// Start processing with high concurrency
	fmt.Println("🔄 Starting high-throughput processing...")
	fmt.Println("   Processing orders with 10 concurrent workers")

	processedCount := 0
	var mu sync.Mutex

	err = pool.Start(func(ctx context.Context, msg *queue.Message) error {
		// Simulate order processing
		orderID := msg.Payload["order_id"]
		customerID := msg.Payload["customer_id"]
		amount := msg.Payload["amount"]

		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		processedCount++
		count := processedCount
		mu.Unlock()

		if count%100 == 0 {
			fmt.Printf("📈 Processed %d orders (Order ID: %v, Customer: %v, Amount: $%.2f)\n",
				count, orderID, customerID, amount)
		}

		return nil // Success
	})

	if err != nil {
		log.Fatal("Error in worker pool:", err)
	}
}
