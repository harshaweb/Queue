// Package pkg provides a comprehensive, feature-rich queue SDK for massive scale applications.
//
// This SDK offers advanced queueing capabilities including:
// - High availability and fault tolerance
// - Dead letter queues and retry mechanisms
// - Priority queues and delayed processing
// - Task scheduling and deadline management
// - Batch operations for high throughput
// - Real-time monitoring and metrics
// - Consumer groups and load balancing
//
// Basic Usage:
//
//	import "github.com/harshaweb/Queue/pkg"
//
//	// Create a queue
//	q, err := pkg.NewQueue("my-queue", &pkg.Config{
//		RedisAddress: "localhost:6379",
//	})
//	defer q.Close()
//
//	// Send a message
//	id, err := q.Send(map[string]interface{}{
//		"user_id": 12345,
//		"action":  "send_email",
//	})
//
//	// Process messages
//	err = q.Consume(func(ctx context.Context, msg *pkg.Message) error {
//		// Process message
//		return nil
//	})
//
// Advanced Features:
//
//	// Priority queue
//	pq, err := pkg.NewPriorityQueue("priority-queue", config, []int{10, 5, 1})
//
//	// Delayed processing
//	dq, err := pkg.NewDelayedQueue("delayed-queue", config)
//	dq.SendDelayed(payload, time.Hour)
//
//	// Batch processing
//	bq, err := pkg.NewBatchQueue("batch-queue", config)
//	bq.ConsumeBatch(func(ctx context.Context, msgs []*pkg.Message) error {
//		// Process batch
//		return nil
//	})
//
//	// Skip to deadline
//	err = q.SkipToDeadline()
//
//	// Get queue health
//	health, err := q.GetHealthStatus()
//
//	// Metrics and monitoring
//	metrics := q.GetMetrics()
package pkg
