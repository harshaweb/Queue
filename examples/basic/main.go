package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/harshaweb/queue/pkg"
)

func main() {
	fmt.Println("=== Queue SDK Basic Example ===")

	// Create a queue with default configuration
	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"
	config.EnableMetrics = true

	queue, err := pkg.NewQueue("test-queue", config)
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		log.Println("Make sure Redis is running on localhost:6379")
		return
	}
	defer queue.Close()

	fmt.Println("✅ Connected to Redis successfully")

	// Send some test messages
	fmt.Println("\n📤 Sending messages...")

	messages := []map[string]interface{}{
		{"task": "send_email", "user_id": 12345, "priority": "high"},
		{"task": "process_payment", "user_id": 67890, "amount": 100.50},
		{"task": "generate_report", "report_type": "daily"},
	}

	var sentIDs []string
	for i, msg := range messages {
		id, err := queue.Send(msg, &pkg.SendOptions{
			Priority: i + 1,
			Headers: map[string]string{
				"source":    "example",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		})
		if err != nil {
			log.Printf("Failed to send message: %v", err)
			continue
		}
		sentIDs = append(sentIDs, id)
		fmt.Printf("  ✅ Sent message %d with ID: %s\n", i+1, id)
	}

	// Send a delayed message
	delayedID, err := queue.Send(map[string]interface{}{
		"task":      "cleanup",
		"scheduled": true,
	}, &pkg.SendOptions{
		Delay: 5 * time.Second,
		Headers: map[string]string{
			"type": "delayed",
		},
	})
	if err == nil {
		fmt.Printf("  ⏰ Sent delayed message with ID: %s (will process in 5 seconds)\n", delayedID)
	}

	// Send a message with deadline
	deadline := time.Now().Add(30 * time.Second)
	deadlineID, err := queue.Send(map[string]interface{}{
		"task":    "urgent_notification",
		"expires": true,
	}, &pkg.SendOptions{
		Priority: 10,
		Deadline: &deadline,
		Headers: map[string]string{
			"urgency": "high",
		},
	})
	if err == nil {
		fmt.Printf("  ⏰ Sent message with deadline: %s (expires at %s)\n", deadlineID, deadline.Format("15:04:05"))
	}

	// Check queue status
	info, err := queue.GetInfo()
	if err == nil {
		fmt.Printf("\n📊 Queue Info:\n")
		fmt.Printf("  - Queue Length: %d messages\n", info.Length)
		fmt.Printf("  - Pending Messages: %d\n", info.PendingMessages)
		fmt.Printf("  - Consumer Groups: %d\n", info.ConsumerGroups)
	}

	// Get health status
	health, err := queue.GetHealthStatus()
	if err == nil {
		fmt.Printf("\n🏥 Health Status:\n")
		fmt.Printf("  - Healthy: %v\n", health.Healthy)
		fmt.Printf("  - Redis Connected: %v\n", health.RedisConnected)
		fmt.Printf("  - Active Consumers: %d\n", health.ConsumerCount)
		if len(health.Issues) > 0 {
			fmt.Printf("  - Issues: %v\n", health.Issues)
		}
	}

	// Create a consumer to process messages
	fmt.Println("\n🔄 Starting message consumer...")

	processedCount := 0
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start consuming messages
	go func() {
		err := queue.Consume(func(ctx context.Context, msg *pkg.Message) error {
			processedCount++

			fmt.Printf("  � Processing message %d:\n", processedCount)
			fmt.Printf("     ID: %s\n", msg.ID)
			fmt.Printf("     Priority: %d\n", msg.Priority)
			fmt.Printf("     Payload: %v\n", msg.Payload)

			if msg.Headers != nil {
				fmt.Printf("     Headers: %v\n", msg.Headers)
			}

			if msg.Deadline != nil {
				fmt.Printf("     Deadline: %s\n", msg.Deadline.Format("15:04:05"))
			}

			// Simulate processing time
			time.Sleep(500 * time.Millisecond)

			fmt.Printf("     ✅ Completed processing message %d\n", processedCount)
			return nil
		}, &pkg.ConsumeOptions{
			AutoAck:        true,
			MaxConcurrency: 2,
			BatchSize:      5,
		})

		if err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	// Wait for processing
	fmt.Println("⏳ Processing messages for 30 seconds...")
	<-ctx.Done()

	// Show final metrics
	if metrics := queue.GetMetrics(); metrics != nil {
		fmt.Printf("\n📈 Final Metrics:\n")
		fmt.Printf("  - Messages Sent: %d\n", metrics.MessagesSent)
		fmt.Printf("  - Messages Processed: %d\n", metrics.MessagesProcessed)
		fmt.Printf("  - Processing Errors: %d\n", metrics.ProcessingErrors)
		fmt.Printf("  - Average Processing Time: %v\n", metrics.AverageProcessTime)
		fmt.Printf("  - Throughput: %.2f messages/second\n", metrics.Throughput)
		fmt.Printf("  - Error Rate: %.2f%%\n", metrics.ErrorRate*100)
	}

	fmt.Println("\n🎉 Example completed successfully!")
}
