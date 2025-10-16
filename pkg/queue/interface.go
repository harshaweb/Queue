package queue

import (
	"context"
	"errors"
	"time"
)

// Queue defines the interface for queue operations
type Queue interface {
	// Core operations
	Enqueue(ctx context.Context, queueName string, message *Message, opts *MessageOptions) error
	Dequeue(ctx context.Context, queueName string, opts *ConsumeOptions) (*Message, error)
	Ack(ctx context.Context, queueName string, messageID string) error
	Nack(ctx context.Context, queueName string, messageID string, requeue bool) error

	// Advanced operations
	MoveToLast(ctx context.Context, queueName string, messageID string) error
	Skip(ctx context.Context, queueName string, messageID string, duration *time.Duration) error
	MoveToDeadLetter(ctx context.Context, queueName string, messageID string) error
	Requeue(ctx context.Context, queueName string, messageID string, delay *time.Duration) error

	// Queue management
	CreateQueue(ctx context.Context, config *QueueConfig) error
	DeleteQueue(ctx context.Context, queueName string) error
	GetQueueStats(ctx context.Context, queueName string) (*QueueStats, error)
	ListQueues(ctx context.Context) ([]string, error)

	// Consumer group management
	PauseQueue(ctx context.Context, queueName string) error
	ResumeQueue(ctx context.Context, queueName string) error
	GetConsumerGroups(ctx context.Context, queueName string) ([]ConsumerGroup, error)

	// Health and cleanup
	HealthCheck(ctx context.Context) error
	CleanupExpiredMessages(ctx context.Context) error
	Close() error
}

// ConsumerGroup represents a consumer group
type ConsumerGroup struct {
	Name      string `json:"name"`
	Consumers int    `json:"consumers"`
	Pending   int64  `json:"pending"`
	LastID    string `json:"last_id"`
}

// QueueManager provides high-level queue management
type QueueManager interface {
	Queue

	// Batch operations
	BatchEnqueue(ctx context.Context, queueName string, messages []*Message, opts *MessageOptions) error
	BatchDequeue(ctx context.Context, queueName string, count int, opts *ConsumeOptions) ([]*Message, error)

	// Scheduled message handling
	ProcessScheduledMessages(ctx context.Context) error

	// Migration and backup
	ExportQueue(ctx context.Context, queueName string) ([]byte, error)
	ImportQueue(ctx context.Context, queueName string, data []byte) error
}

// Common errors
var (
	ErrQueueNotFound     = errors.New("queue not found")
	ErrMessageNotFound   = errors.New("message not found")
	ErrQueueExists       = errors.New("queue already exists")
	ErrQueuePaused       = errors.New("queue is paused")
	ErrInvalidMessage    = errors.New("invalid message")
	ErrMaxRetriesReached = errors.New("max retries reached")
	ErrTimeout           = errors.New("operation timeout")
)
