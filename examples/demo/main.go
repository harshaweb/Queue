package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/queue-system/redis-queue/pkg/client"
)

func main() {
	fmt.Println("🚀 Redis Queue System Demo")
	fmt.Println("===========================")

	// Create client
	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}

	queueClient, err := client.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create queue client: %v", err)
	}
	defer queueClient.Close()

	ctx := context.Background()

	// Demo: Basic Queue Operations
	fmt.Println("\n📦 Demo: Basic Queue Operations")
	basicDemo(ctx, queueClient)

	fmt.Println("\n✅ Demo completed!")
}

func basicDemo(ctx context.Context, client *client.Client) {
	queueName := "demo-queue"

	// Enqueue a message
	payload := map[string]interface{}{
		"message":   "Hello, Queue!",
		"timestamp": time.Now(),
	}

	result, err := client.Enqueue(ctx, queueName, payload)
	if err != nil {
		log.Printf("Failed to enqueue: %v", err)
		return
	}
	fmt.Printf("✅ Message enqueued: %s\n", result.MessageID)

	// Dequeue and process
	messages, err := client.Dequeue(ctx, queueName, 1)
	if err != nil {
		log.Printf("Failed to dequeue: %v", err)
		return
	}

	if len(messages) > 0 {
		message := messages[0]
		fmt.Printf("📋 Processing message: %s\n", message.Payload["message"])

		// Acknowledge
		err = client.Ack(ctx, queueName, message.ID)
		if err != nil {
			log.Printf("Failed to ack: %v", err)
			return
		}
		fmt.Printf("✅ Message processed successfully\n")
	}
}
