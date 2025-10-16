package queue

import (
	"time"
)

// Additional queue-specific types that don't belong in message.go or interface.go

// DeadLetterPolicy defines dead letter queue behavior
type DeadLetterPolicy struct {
	Enabled       bool          `json:"enabled"`
	MaxRetries    int           `json:"max_retries"`
	QueueName     string        `json:"queue_name"`
	RetentionTime time.Duration `json:"retention_time"`
}

// QueueMetrics contains detailed queue metrics
type QueueMetrics struct {
	EnqueueRate     float64       `json:"enqueue_rate"`    // messages per second
	DequeueRate     float64       `json:"dequeue_rate"`    // messages per second
	ProcessingTime  time.Duration `json:"processing_time"` // average processing time
	ErrorRate       float64       `json:"error_rate"`      // errors per second
	ActiveConsumers int           `json:"active_consumers"`
	PendingMessages int64         `json:"pending_messages"`
	TotalBytes      int64         `json:"total_bytes"`
	LastUpdated     time.Time     `json:"last_updated"`
}

// QueueState represents the operational state of a queue
type QueueState int

const (
	QueueStateActive QueueState = iota
	QueueStatePaused
	QueueStateDraining
	QueueStateMaintenance
)

// String returns the string representation of QueueState
func (qs QueueState) String() string {
	switch qs {
	case QueueStateActive:
		return "active"
	case QueueStatePaused:
		return "paused"
	case QueueStateDraining:
		return "draining"
	case QueueStateMaintenance:
		return "maintenance"
	default:
		return "unknown"
	}
}
