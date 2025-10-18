package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/harshaweb/queue/pkg"
)

// User represents a user in our system
type User struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Premium bool   `json:"premium"`
}

// NotificationTask represents a notification to send
type NotificationTask struct {
	Type         string    `json:"type"`
	UserID       int       `json:"user_id"`
	Message      string    `json:"message"`
	ScheduledFor time.Time `json:"scheduled_for,omitempty"`
	Priority     int       `json:"priority"`
}

func main() {
	fmt.Println("Queue SDK - JSON & Scheduling Example")
	fmt.Println("====================================")

	// Create queue for notifications
	config := pkg.DefaultConfig()
	config.RedisAddress = "localhost:6379"

	q, err := pkg.NewQueue("notifications", config)
	if err != nil {
		log.Fatal("Failed to create queue:", err)
	}
	defer q.Close()

	fmt.Println("✅ Notification queue created")

	// Example users
	users := []User{
		{ID: 1, Name: "Alice Johnson", Email: "alice@example.com", Premium: true},
		{ID: 2, Name: "Bob Smith", Email: "bob@example.com", Premium: false},
		{ID: 3, Name: "Carol Davis", Email: "carol@example.com", Premium: true},
	}

	// Send immediate notifications
	fmt.Println("📤 Sending immediate notifications...")

	for _, user := range users {
		notification := NotificationTask{
			Type:     "welcome",
			UserID:   user.ID,
			Message:  fmt.Sprintf("Welcome %s! Thanks for joining our platform.", user.Name),
			Priority: 1,
		}

		// Send JSON directly
		id, err := q.SendJSON(notification)
		if err != nil {
			log.Printf("Failed to send notification for user %d: %v", user.ID, err)
			continue
		}

		fmt.Printf("✅ Welcome notification sent to %s (ID: %s)\n", user.Name, id[:8])
	}

	// Send scheduled notifications (premium users get early access notification)
	fmt.Println("📅 Scheduling future notifications...")

	for _, user := range users {
		if user.Premium {
			scheduledNotification := NotificationTask{
				Type:         "early_access",
				UserID:       user.ID,
				Message:      fmt.Sprintf("Hi %s! Early access to new features is now available.", user.Name),
				ScheduledFor: time.Now().Add(5 * time.Second), // 5 seconds from now
				Priority:     5,                               // Higher priority
			}

			// Send with delay
			id, err := q.SendJSON(scheduledNotification, &queue.SendOptions{
				Delay:    5 * time.Second,
				Priority: 5,
			})
			if err != nil {
				log.Printf("Failed to schedule notification for user %d: %v", user.ID, err)
				continue
			}

			fmt.Printf("⏰ Scheduled early access notification for %s (ID: %s)\n", user.Name, id[:8])
		}
	}

	// Send batch notifications (daily digest)
	fmt.Println("📦 Sending batch notifications...")

	var dailyDigests []interface{}
	for _, user := range users {
		digest := NotificationTask{
			Type:     "daily_digest",
			UserID:   user.ID,
			Message:  fmt.Sprintf("Daily digest for %s - 3 new updates available!", user.Name),
			Priority: 2,
		}
		dailyDigests = append(dailyDigests, digest)
	}

	ids, err := q.BatchSendJSON(dailyDigests)
	if err != nil {
		log.Fatal("Failed to send batch notifications:", err)
	}

	fmt.Printf("✅ Sent %d daily digest notifications\n", len(ids))

	// Process notifications with different handling based on type
	fmt.Println("🔄 Starting notification processing...")
	fmt.Println("   Processing notifications by type and priority")

	err = q.Receive(func(ctx context.Context, msg *queue.Message) error {
		// Parse the notification task
		var task NotificationTask
		taskBytes, err := json.Marshal(msg.Payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %v", err)
		}

		err = json.Unmarshal(taskBytes, &task)
		if err != nil {
			return fmt.Errorf("failed to unmarshal notification task: %v", err)
		}

		// Process based on notification type
		switch task.Type {
		case "welcome":
			fmt.Printf("📧 Sending welcome email to user %d: %s\n", task.UserID, task.Message)
			// Simulate email sending
			time.Sleep(50 * time.Millisecond)

		case "early_access":
			fmt.Printf("🎯 Sending premium notification to user %d: %s\n", task.UserID, task.Message)
			// Simulate push notification
			time.Sleep(30 * time.Millisecond)

		case "daily_digest":
			fmt.Printf("📰 Sending daily digest to user %d: %s\n", task.UserID, task.Message)
			// Simulate digest compilation and sending
			time.Sleep(100 * time.Millisecond)

		default:
			fmt.Printf("❓ Unknown notification type: %s for user %d\n", task.Type, task.UserID)
		}

		fmt.Printf("✅ Notification processed (Priority: %d, User: %d)\n", task.Priority, task.UserID)
		return nil

	}, &queue.ReceiveOptions{
		BatchSize:      10,
		MaxConcurrency: 3, // 3 concurrent notification processors
		AutoAck:        true,
	})

	if err != nil {
		log.Fatal("Error processing notifications:", err)
	}
}
