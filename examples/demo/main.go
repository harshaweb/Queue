package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/queue-system/redis-queue/pkg/client"
)

// Order represents an e-commerce order
type Order struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	Amount     float64   `json:"amount"`
	Items      []Item    `json:"items"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// Item represents an order item
type Item struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// EmailNotification represents an email to be sent
type EmailNotification struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Type    string `json:"type"`
}

// InventoryUpdate represents an inventory change
type InventoryUpdate struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Operation string `json:"operation"` // "reserve", "release", "consume"
}

// PaymentRequest represents a payment to be processed
type PaymentRequest struct {
	OrderID       string  `json:"order_id"`
	Amount        float64 `json:"amount"`
	PaymentMethod string  `json:"payment_method"`
	CustomerID    string  `json:"customer_id"`
}

func main() {
	log.Println("Starting Redis Queue Demo Application")

	// Create client
	config := client.DefaultConfig()
	config.RedisConfig.Addresses = []string{"localhost:6379"}

	queueClient, err := client.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create queue client: %v", err)
	}
	defer queueClient.Close()

	// Create demo queues
	if err := createQueues(queueClient); err != nil {
		log.Fatalf("Failed to create queues: %v", err)
	}

	// Start workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Start order processing worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		runOrderProcessor(ctx, queueClient)
	}()

	// Start email worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		runEmailWorker(ctx, queueClient)
	}()

	// Start inventory worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		runInventoryWorker(ctx, queueClient)
	}()

	// Start payment worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		runPaymentWorker(ctx, queueClient)
	}()

	// Generate sample orders
	go generateSampleOrders(ctx, queueClient)

	// Run demo for 2 minutes
	time.Sleep(2 * time.Minute)

	log.Println("Stopping demo...")
	cancel()
	wg.Wait()

	// Print final statistics
	printQueueStats(queueClient)

	log.Println("Demo completed")
}

func createQueues(client *client.Client) error {
	ctx := context.Background()

	queues := []struct {
		name              string
		maxRetries        int
		visTimeout        time.Duration
		deadLetterEnabled bool
	}{
		{"orders", 3, 30 * time.Second, true},
		{"emails", 5, 60 * time.Second, true},
		{"inventory", 2, 15 * time.Second, true},
		{"payments", 3, 45 * time.Second, true},
	}

	for _, q := range queues {
		config := &client.QueueConfig{
			Name:              q.name,
			MaxRetries:        q.maxRetries,
			VisibilityTimeout: q.visTimeout,
			DeadLetterEnabled: q.deadLetterEnabled,
			MessageRetention:  24 * time.Hour,
		}

		if err := client.CreateQueue(ctx, config); err != nil {
			log.Printf("Queue %s might already exist: %v", q.name, err)
		} else {
			log.Printf("Created queue: %s", q.name)
		}
	}

	return nil
}

func generateSampleOrders(ctx context.Context, client *client.Client) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	orderCounter := 1

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			order := Order{
				ID:         fmt.Sprintf("order-%d", orderCounter),
				CustomerID: fmt.Sprintf("customer-%d", (orderCounter%10)+1),
				Amount:     float64(orderCounter*10 + 50),
				Items: []Item{
					{
						ProductID: fmt.Sprintf("product-%d", (orderCounter%5)+1),
						Quantity:  orderCounter%3 + 1,
						Price:     float64(orderCounter*5 + 25),
					},
				},
				Status:    "pending",
				CreatedAt: time.Now(),
			}

			_, err := client.Enqueue(ctx, "orders", order)
			if err != nil {
				log.Printf("Failed to enqueue order: %v", err)
			} else {
				log.Printf("Generated order: %s", order.ID)
			}

			orderCounter++
		}
	}
}

func runOrderProcessor(ctx context.Context, client *client.Client) {
	log.Println("Starting order processor...")

	handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		var order Order
		data, _ := json.Marshal(payload)
		if err := json.Unmarshal(data, &order); err != nil {
			log.Printf("Failed to parse order: %v", err)
			return client.Nack(false)
		}

		log.Printf("Processing order %s (attempt %d)", order.ID, msgCtx.RetryCount+1)

		// Simulate processing time
		time.Sleep(time.Duration(500+msgCtx.RetryCount*200) * time.Millisecond)

		// Simulate occasional failures (10% failure rate)
		if order.ID[len(order.ID)-1] == '3' && msgCtx.RetryCount == 0 {
			log.Printf("Order %s failed processing (simulated)", order.ID)
			return client.Nack(true) // Requeue for retry
		}

		// Process order - trigger downstream processes

		// 1. Reserve inventory
		for _, item := range order.Items {
			inventoryUpdate := InventoryUpdate{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				Operation: "reserve",
			}
			client.Enqueue(ctx, "inventory", inventoryUpdate)
		}

		// 2. Process payment
		paymentReq := PaymentRequest{
			OrderID:       order.ID,
			Amount:        order.Amount,
			PaymentMethod: "credit_card",
			CustomerID:    order.CustomerID,
		}
		client.Enqueue(ctx, "payments", paymentReq)

		// 3. Send confirmation email
		email := EmailNotification{
			To:      fmt.Sprintf("customer-%s@example.com", order.CustomerID),
			Subject: fmt.Sprintf("Order Confirmation - %s", order.ID),
			Body:    fmt.Sprintf("Your order %s has been confirmed.", order.ID),
			Type:    "order_confirmation",
		}
		client.Enqueue(ctx, "emails", email)

		log.Printf("Order %s processed successfully", order.ID)
		return client.Ack()
	}

	err := client.Consume(ctx, "orders", handler,
		client.WithBatchSize(1),
		client.WithVisibilityTimeout(30*time.Second),
		client.WithMaxRetries(3),
	)

	if err != nil && err != context.Canceled {
		log.Printf("Order processor error: %v", err)
	}
}

func runEmailWorker(ctx context.Context, client *client.Client) {
	log.Println("Starting email worker...")

	handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		var email EmailNotification
		data, _ := json.Marshal(payload)
		if err := json.Unmarshal(data, &email); err != nil {
			log.Printf("Failed to parse email: %v", err)
			return client.Nack(false)
		}

		log.Printf("Sending email to %s: %s", email.To, email.Subject)

		// Simulate email sending
		time.Sleep(200 * time.Millisecond)

		// Simulate occasional email failures (5% failure rate)
		if email.To[len(email.To)-5] == '5' && msgCtx.RetryCount == 0 {
			log.Printf("Email sending failed (simulated): %s", email.To)
			return client.Nack(true)
		}

		log.Printf("Email sent successfully to %s", email.To)
		return client.Ack()
	}

	err := client.Consume(ctx, "emails", handler,
		client.WithBatchSize(2), // Process emails in batches
		client.WithVisibilityTimeout(60*time.Second),
		client.WithMaxRetries(5),
	)

	if err != nil && err != context.Canceled {
		log.Printf("Email worker error: %v", err)
	}
}

func runInventoryWorker(ctx context.Context, client *client.Client) {
	log.Println("Starting inventory worker...")

	handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		var update InventoryUpdate
		data, _ := json.Marshal(payload)
		if err := json.Unmarshal(data, &update); err != nil {
			log.Printf("Failed to parse inventory update: %v", err)
			return client.Nack(false)
		}

		log.Printf("Processing inventory %s for product %s (qty: %d)",
			update.Operation, update.ProductID, update.Quantity)

		// Simulate inventory processing
		time.Sleep(100 * time.Millisecond)

		// Simulate inventory shortage (rare case)
		if update.ProductID == "product-3" && update.Operation == "reserve" && msgCtx.RetryCount == 0 {
			log.Printf("Inventory shortage for %s (simulated)", update.ProductID)
			// Move to dead letter queue for manual handling
			return client.DeadLetter()
		}

		log.Printf("Inventory %s completed for %s", update.Operation, update.ProductID)
		return client.Ack()
	}

	err := client.Consume(ctx, "inventory", handler,
		client.WithBatchSize(1),
		client.WithVisibilityTimeout(15*time.Second),
		client.WithMaxRetries(2),
	)

	if err != nil && err != context.Canceled {
		log.Printf("Inventory worker error: %v", err)
	}
}

func runPaymentWorker(ctx context.Context, client *client.Client) {
	log.Println("Starting payment worker...")

	handler := func(ctx context.Context, payload interface{}, msgCtx *client.MessageContext) *client.MessageResult {
		var payment PaymentRequest
		data, _ := json.Marshal(payload)
		if err := json.Unmarshal(data, &payment); err != nil {
			log.Printf("Failed to parse payment: %v", err)
			return client.Nack(false)
		}

		log.Printf("Processing payment for order %s (amount: $%.2f)",
			payment.OrderID, payment.Amount)

		// Simulate payment processing time
		time.Sleep(300 * time.Millisecond)

		// Simulate payment failures (8% failure rate)
		if payment.OrderID[len(payment.OrderID)-1] == '7' && msgCtx.RetryCount < 2 {
			log.Printf("Payment failed for order %s (simulated)", payment.OrderID)
			return client.Nack(true)
		}

		// If payment finally fails after retries, move to last for manual review
		if payment.OrderID[len(payment.OrderID)-1] == '7' && msgCtx.RetryCount >= 2 {
			log.Printf("Payment permanently failed for order %s, moving to end of queue", payment.OrderID)
			return client.MoveToLast()
		}

		log.Printf("Payment processed successfully for order %s", payment.OrderID)
		return client.Ack()
	}

	err := client.Consume(ctx, "payments", handler,
		client.WithBatchSize(1),
		client.WithVisibilityTimeout(45*time.Second),
		client.WithMaxRetries(3),
	)

	if err != nil && err != context.Canceled {
		log.Printf("Payment worker error: %v", err)
	}
}

func printQueueStats(client *client.Client) {
	ctx := context.Background()

	queues := []string{"orders", "emails", "inventory", "payments"}

	fmt.Println("\n=== Final Queue Statistics ===")
	for _, queueName := range queues {
		stats, err := client.GetQueueStats(ctx, queueName)
		if err != nil {
			log.Printf("Failed to get stats for %s: %v", queueName, err)
			continue
		}

		fmt.Printf("\nQueue: %s\n", stats.Name)
		fmt.Printf("  Length: %d\n", stats.Length)
		fmt.Printf("  In Flight: %d\n", stats.InFlight)
		fmt.Printf("  Processed: %d\n", stats.Processed)
		fmt.Printf("  Failed: %d\n", stats.Failed)
		fmt.Printf("  Consumers: %d\n", stats.ConsumerCount)
		fmt.Printf("  Dead Letter Size: %d\n", stats.DeadLetterSize)
		fmt.Printf("  Last Activity: %s\n", stats.LastActivity.Format("15:04:05"))
	}
}
