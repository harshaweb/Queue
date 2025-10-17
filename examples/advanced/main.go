package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/harshaweb/queue/pkg"
)

func main() {
	fmt.Println("=== Advanced Features Example ===")

	// Create configuration with advanced settings
	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"
	config.EnableMetrics = true
	config.EnableDLQ = true
	config.MaxRetries = 2

	// 1. Priority Queue Example
	fmt.Println("\n🔥 Priority Queue Example")
	fmt.Println("=========================")

	priorityQueue, err := pkg.NewPriorityQueue("priority-test", config, []int{10, 5, 1})
	if err != nil {
		log.Printf("Failed to create priority queue: %v", err)
		return
	}
	defer priorityQueue.Close()

	// Send messages with different priorities
	highPriorityMsg := map[string]interface{}{
		"task": "urgent_email",
		"type": "critical",
	}

	mediumPriorityMsg := map[string]interface{}{
		"task": "regular_notification",
		"type": "normal",
	}

	lowPriorityMsg := map[string]interface{}{
		"task": "cleanup_job",
		"type": "background",
	}

	// Send in reverse priority order to demonstrate priority handling
	id1, _ := priorityQueue.Send(lowPriorityMsg, &pkg.SendOptions{Priority: 1})
	id2, _ := priorityQueue.Send(mediumPriorityMsg, &pkg.SendOptions{Priority: 5})
	id3, _ := priorityQueue.Send(highPriorityMsg, &pkg.SendOptions{Priority: 10})

	fmt.Printf("✅ Sent priority messages: Low(%s), Medium(%s), High(%s)\n", id1, id2, id3)

	// Process by priority (high priority messages will be processed first)
	go func() {
		priorityQueue.ConsumeByPriority(func(ctx context.Context, msg *pkg.Message) error {
			fmt.Printf("📥 Processing priority %d message: %v\n", msg.Priority, msg.Payload["task"])
			time.Sleep(200 * time.Millisecond)
			return nil
		})
	}()

	time.Sleep(3 * time.Second)

	// 2. Delayed Queue Example
	fmt.Println("\n⏰ Delayed Queue Example")
	fmt.Println("========================")

	delayedQueue, err := pkg.NewDelayedQueue("delayed-test", config)
	if err != nil {
		log.Printf("Failed to create delayed queue: %v", err)
		return
	}
	defer delayedQueue.Close()

	// Start the delayed queue scheduler
	delayedQueue.Start()
	defer delayedQueue.Stop()

	// Send delayed messages
	futureMsg1 := map[string]interface{}{
		"task": "reminder",
		"text": "5 second reminder",
	}

	futureMsg2 := map[string]interface{}{
		"task": "reminder",
		"text": "10 second reminder",
	}

	id4, _ := delayedQueue.SendDelayed(futureMsg1, 5*time.Second)
	id5, _ := delayedQueue.SendDelayed(futureMsg2, 10*time.Second)

	fmt.Printf("✅ Scheduled delayed messages: 5s(%s), 10s(%s)\n", id4, id5)

	// Process delayed messages
	go func() {
		delayedQueue.Consume(func(ctx context.Context, msg *pkg.Message) error {
			fmt.Printf("⏰ Processing delayed message: %v (sent at %s)\n",
				msg.Payload["text"], time.Now().Format("15:04:05"))
			return nil
		})
	}()

	// 3. Batch Processing Example
	fmt.Println("\n📦 Batch Processing Example")
	fmt.Println("===========================")

	batchQueue, err := pkg.NewBatchQueue("batch-test", config)
	if err != nil {
		log.Printf("Failed to create batch queue: %v", err)
		return
	}
	defer batchQueue.Close()

	// Send multiple messages for batch processing
	for i := 1; i <= 10; i++ {
		msg := map[string]interface{}{
			"batch_item": i,
			"data":       fmt.Sprintf("item_%d", i),
		}
		batchQueue.Send(msg)
	}

	fmt.Println("✅ Sent 10 messages for batch processing")

	// Process messages in batches
	go func() {
		batchQueue.ConsumeBatch(func(ctx context.Context, msgs []*pkg.Message) error {
			fmt.Printf("📦 Processing batch of %d messages: ", len(msgs))
			for _, msg := range msgs {
				fmt.Printf("%v ", msg.Payload["batch_item"])
			}
			fmt.Println()
			time.Sleep(500 * time.Millisecond)
			return nil
		}, &pkg.ConsumeOptions{
			BatchSize: 5, // Process 5 messages at a time
		})
	}()

	// 4. Message Scheduling Example
	fmt.Println("\n📅 Message Scheduling Example")
	fmt.Println("=============================")

	regularQueue, err := pkg.NewQueue("scheduler-test", config)
	if err != nil {
		log.Printf("Failed to create regular queue: %v", err)
		return
	}
	defer regularQueue.Close()

	scheduler := regularQueue.NewScheduler()
	scheduler.Start()
	defer scheduler.Stop()

	// Schedule a one-time message
	oneTimeMsg := &pkg.Message{
		Payload: map[string]interface{}{
			"task": "one_time_job",
			"time": "future",
		},
	}

	scheduleID1, _ := scheduler.ScheduleMessage(oneTimeMsg, time.Now().Add(3*time.Second))
	fmt.Printf("✅ Scheduled one-time message: %s\n", scheduleID1)

	// Schedule a recurring message
	recurringMsg := &pkg.Message{
		Payload: map[string]interface{}{
			"task": "recurring_job",
			"type": "periodic",
		},
	}

	scheduleID2, _ := scheduler.ScheduleRecurring(recurringMsg, 2*time.Second, 3) // Run 3 times
	fmt.Printf("✅ Scheduled recurring message (3 times every 2s): %s\n", scheduleID2)

	// Process scheduled messages
	go func() {
		regularQueue.Consume(func(ctx context.Context, msg *pkg.Message) error {
			fmt.Printf("📅 Processing scheduled message: %v\n", msg.Payload["task"])
			return nil
		})
	}()

	// 5. Advanced Message Features
	fmt.Println("\n🚀 Advanced Message Features")
	fmt.Println("============================")

	advancedQueue, err := pkg.NewQueue("advanced-test", config)
	if err != nil {
		log.Printf("Failed to create advanced queue: %v", err)
		return
	}
	defer advancedQueue.Close()

	// Send message with deadline
	deadline := time.Now().Add(8 * time.Second)
	deadlineMsg := map[string]interface{}{
		"task":   "time_sensitive",
		"urgent": true,
	}

	id6, _ := advancedQueue.Send(deadlineMsg, &pkg.SendOptions{
		Priority: 10,
		Deadline: &deadline,
		Headers: map[string]string{
			"deadline": deadline.Format(time.RFC3339),
			"source":   "advanced_example",
		},
		MaxRetries: 1,
	})

	fmt.Printf("✅ Sent message with deadline: %s (expires at %s)\n",
		id6, deadline.Format("15:04:05"))

	// Process advanced messages
	go func() {
		advancedQueue.Consume(func(ctx context.Context, msg *pkg.Message) error {
			fmt.Printf("🚀 Processing advanced message: %v\n", msg.Payload["task"])
			if msg.Headers != nil {
				fmt.Printf("   Headers: %v\n", msg.Headers)
			}
			if msg.Deadline != nil {
				fmt.Printf("   Deadline: %s\n", msg.Deadline.Format("15:04:05"))
			}
			return nil
		})
	}()

	// Wait for all processing to complete
	fmt.Println("\n⏳ Processing messages for 15 seconds...")
	time.Sleep(15 * time.Second)

	// Skip messages past deadline
	fmt.Println("\n⏭️ Skipping messages past deadline...")
	advancedQueue.SkipToDeadline()

	// Show final metrics for all queues
	fmt.Println("\n📈 Final Metrics Summary")
	fmt.Println("========================")

	queues := map[string]*pkg.Queue{
		"Priority": priorityQueue.Queue,
		"Delayed":  delayedQueue.Queue,
		"Batch":    batchQueue.Queue,
		"Advanced": advancedQueue,
	}

	for name, q := range queues {
		if metrics := q.GetMetrics(); metrics != nil {
			fmt.Printf("%s Queue - Sent: %d, Processed: %d, Errors: %d\n",
				name, metrics.MessagesSent, metrics.MessagesProcessed, metrics.ProcessingErrors)
		}
	}

	fmt.Println("\n🎉 Advanced features example completed!")
}
