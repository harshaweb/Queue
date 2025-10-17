package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	queue "github.com/harshaweb/Queue"
)

// PaymentRequest represents a payment to process
type PaymentRequest struct {
	ID       string  `json:"id"`
	UserID   int     `json:"user_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Method   string  `json:"method"` // "card", "bank", "wallet"
}

// Simulate different types of errors
var (
	ErrNetworkTimeout    = errors.New("network timeout - retryable")
	ErrInsufficientFunds = errors.New("insufficient funds - not retryable")
	ErrCardExpired       = errors.New("card expired - not retryable")
	ErrBankUnavailable   = errors.New("bank service unavailable - retryable")
)

func main() {
	fmt.Println("Queue SDK - Error Handling & Retry Example")
	fmt.Println("=========================================")

	// Create queue with custom retry configuration
	config := &queue.Config{
		RedisAddress:      "localhost:6379",
		DefaultTimeout:    30 * time.Second,
		DefaultMaxRetries: 5, // Retry up to 5 times
		BatchSize:         10,
	}

	q, err := queue.New("payments", config)
	if err != nil {
		log.Fatal("Failed to create queue:", err)
	}
	defer q.Close()

	fmt.Println("✅ Payment processing queue created")

	// Send some payment requests that will have different success/failure rates
	fmt.Println("📤 Submitting payment requests...")

	payments := []PaymentRequest{
		{ID: "pay_001", UserID: 123, Amount: 99.99, Currency: "USD", Method: "card"},
		{ID: "pay_002", UserID: 456, Amount: 49.99, Currency: "USD", Method: "bank"},
		{ID: "pay_003", UserID: 789, Amount: 199.99, Currency: "EUR", Method: "wallet"},
		{ID: "pay_004", UserID: 101, Amount: 29.99, Currency: "USD", Method: "card"},
		{ID: "pay_005", UserID: 202, Amount: 0.99, Currency: "USD", Method: "card"}, // This will fail (insufficient funds)
	}

	for _, payment := range payments {
		id, err := q.SendJSON(payment)
		if err != nil {
			log.Printf("Failed to submit payment %s: %v", payment.ID, err)
			continue
		}
		fmt.Printf("💳 Payment %s submitted (Queue ID: %s)\n", payment.ID, id[:8])
	}

	// Process payments with intelligent error handling
	fmt.Println("🔄 Starting payment processing with retry logic...")
	fmt.Println("   Simulating various failure scenarios")

	processedCount := 0
	failedCount := 0

	err = q.Receive(func(ctx context.Context, msg *queue.Message) error {
		// Parse payment request
		paymentID := msg.Payload["id"].(string)
		userID := int(msg.Payload["user_id"].(float64))
		amount := msg.Payload["amount"].(float64)
		method := msg.Payload["method"].(string)

		fmt.Printf("💸 Processing payment %s (User: %d, Amount: $%.2f, Method: %s)\n",
			paymentID, userID, amount, method)

		// Simulate payment processing with various error scenarios
		err := simulatePaymentProcessing(paymentID, amount, method)

		if err != nil {
			if isRetryableError(err) {
				// Log the retryable error and let the queue retry
				fmt.Printf("⚠️  Payment %s failed (retryable): %v - will retry\n", paymentID, err)
				return err // This will trigger a retry
			} else {
				// Log non-retryable error and acknowledge to prevent infinite retries
				fmt.Printf("❌ Payment %s failed (non-retryable): %v - moving to failed queue\n", paymentID, err)
				failedCount++

				// Here you might want to send to a dead letter queue or alert admins
				logFailedPayment(paymentID, userID, amount, err)

				return nil // Acknowledge the message to prevent retries
			}
		}

		// Payment successful
		processedCount++
		fmt.Printf("✅ Payment %s processed successfully (Total processed: %d)\n", paymentID, processedCount)

		return nil

	}, &queue.ReceiveOptions{
		MaxConcurrency: 2,     // Process 2 payments concurrently
		AutoAck:        false, // Manual ack so we can handle errors properly
	})

	if err != nil {
		log.Fatal("Error in payment processing:", err)
	}
}

func simulatePaymentProcessing(paymentID string, amount float64, method string) error {
	// Simulate processing time
	time.Sleep(time.Duration(rand.Intn(200)+50) * time.Millisecond)

	// Simulate different error scenarios based on payment characteristics
	switch {
	case amount < 1.0:
		return ErrInsufficientFunds // Non-retryable

	case paymentID == "pay_002" && rand.Float32() < 0.7: // 70% chance of bank being unavailable
		return ErrBankUnavailable // Retryable

	case method == "card" && rand.Float32() < 0.3: // 30% chance of network timeout
		return ErrNetworkTimeout // Retryable

	case paymentID == "pay_005":
		return ErrCardExpired // Non-retryable

	case rand.Float32() < 0.1: // 10% random failure rate
		if rand.Float32() < 0.5 {
			return ErrNetworkTimeout // Retryable
		}
		return ErrInsufficientFunds // Non-retryable
	}

	// Success case
	return nil
}

func isRetryableError(err error) bool {
	switch err {
	case ErrNetworkTimeout, ErrBankUnavailable:
		return true
	case ErrInsufficientFunds, ErrCardExpired:
		return false
	default:
		// For unknown errors, be conservative and don't retry
		return false
	}
}

func logFailedPayment(paymentID string, userID int, amount float64, err error) {
	// In a real system, you might:
	// 1. Send to a dead letter queue
	// 2. Log to a failure audit system
	// 3. Send alerts to administrators
	// 4. Refund the customer
	// 5. Update payment status in database

	fmt.Printf("📝 Logging failed payment - ID: %s, User: %d, Amount: $%.2f, Error: %v\n",
		paymentID, userID, amount, err)

	// Simulate logging to external system
	time.Sleep(10 * time.Millisecond)
}
