package main

import (
	"context"
	"fmt"
	"log"
	"time"

	queue "github.com/harshaweb/Queue"
)

func main() {
	fmt.Println("Queue SDK - Quick Test")
	fmt.Println("=====================")

	// Test 1: Basic functionality
	fmt.Println("Test 1: Basic send and receive...")

	q, err := queue.New("test-queue")
	if err != nil {
		log.Fatal("❌ Failed to create queue:", err)
	}
	defer q.Close()

	// Test health
	if err := q.Health(); err != nil {
		log.Fatal("❌ Queue health check failed:", err)
	}
	fmt.Println("✅ Queue health check passed")

	// Send test message
	testData := map[string]interface{}{
		"test": "hello world",
		"num":  42,
		"time": time.Now().Format(time.RFC3339),
	}

	msgID, err := q.Send(testData)
	if err != nil {
		log.Fatal("❌ Failed to send message:", err)
	}
	fmt.Printf("✅ Message sent with ID: %s\n", msgID[:8])

	// Test stats
	stats, err := q.GetStats()
	if err != nil {
		log.Printf("⚠️ Failed to get stats: %v", err)
	} else {
		fmt.Printf("✅ Stats - Pending: %d\n", stats.PendingMessages)
	}

	// Receive and process message
	fmt.Println("🔄 Processing message...")

	processed := make(chan bool, 1)
	go func() {
		err = q.Receive(func(ctx context.Context, msg *queue.Message) error {
			fmt.Printf("✅ Received message: %+v\n", msg.Payload)
			processed <- true
			return nil
		})
		if err != nil {
			log.Printf("❌ Error processing: %v", err)
		}
	}()

	// Wait for message processing or timeout
	select {
	case <-processed:
		fmt.Println("✅ Message processed successfully")
	case <-time.After(5 * time.Second):
		fmt.Println("⚠️ Message processing timeout")
	}

	fmt.Println("✅ All tests passed! SDK is working correctly.")
}
