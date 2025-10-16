package queue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockQueue is a mock implementation of the Queue interface for testing
type MockQueue struct {
	mock.Mock
}

func (m *MockQueue) Enqueue(ctx context.Context, message *Message, opts ...EnqueueOption) (*EnqueueResult, error) {
	args := m.Called(ctx, message, opts)
	return args.Get(0).(*EnqueueResult), args.Error(1)
}

func (m *MockQueue) Dequeue(ctx context.Context, opts ...DequeueOption) ([]*Message, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).([]*Message), args.Error(1)
}

func (m *MockQueue) Ack(ctx context.Context, messageID string) error {
	args := m.Called(ctx, messageID)
	return args.Error(0)
}

func (m *MockQueue) Nack(ctx context.Context, messageID string, requeue bool) error {
	args := m.Called(ctx, messageID, requeue)
	return args.Error(0)
}

func (m *MockQueue) GetStats(ctx context.Context) (*QueueStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(*QueueStats), args.Error(1)
}

func (m *MockQueue) MoveToFirst(ctx context.Context, messageID string) error {
	args := m.Called(ctx, messageID)
	return args.Error(0)
}

func (m *MockQueue) MoveToLast(ctx context.Context, messageID string) error {
	args := m.Called(ctx, messageID)
	return args.Error(0)
}

func (m *MockQueue) MoveTo(ctx context.Context, messageID string, position int64) error {
	args := m.Called(ctx, messageID, position)
	return args.Error(0)
}

func (m *MockQueue) Skip(ctx context.Context, messageID string) error {
	args := m.Called(ctx, messageID)
	return args.Error(0)
}

func (m *MockQueue) Requeue(ctx context.Context, messageID string, delay time.Duration) error {
	args := m.Called(ctx, messageID, delay)
	return args.Error(0)
}

func (m *MockQueue) MoveToDeadLetter(ctx context.Context, messageID string, reason string) error {
	args := m.Called(ctx, messageID, reason)
	return args.Error(0)
}

func (m *MockQueue) BatchEnqueue(ctx context.Context, messages []*Message, opts ...EnqueueOption) (*BatchEnqueueResult, error) {
	args := m.Called(ctx, messages, opts)
	return args.Get(0).(*BatchEnqueueResult), args.Error(1)
}

func (m *MockQueue) Peek(ctx context.Context, count int) ([]*Message, error) {
	args := m.Called(ctx, count)
	return args.Get(0).([]*Message), args.Error(1)
}

func (m *MockQueue) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockQueueManager is a mock implementation of the QueueManager interface
type MockQueueManager struct {
	mock.Mock
}

func (m *MockQueueManager) CreateQueue(ctx context.Context, config *QueueConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockQueueManager) GetQueue(ctx context.Context, name string) (Queue, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(Queue), args.Error(1)
}

func (m *MockQueueManager) DeleteQueue(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockQueueManager) ListQueues(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockQueueManager) PauseQueue(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockQueueManager) ResumeQueue(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockQueueManager) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Test Message structure and validation
func TestMessage(t *testing.T) {
	t.Run("Valid Message", func(t *testing.T) {
		message := &Message{
			ID:        "test-123",
			QueueName: "test-queue",
			Payload:   map[string]interface{}{"key": "value"},
			CreatedAt: time.Now(),
			Attempts:  0,
		}

		assert.Equal(t, "test-123", message.ID)
		assert.Equal(t, "test-queue", message.QueueName)
		assert.Equal(t, "value", message.Payload.(map[string]interface{})["key"])
		assert.Equal(t, 0, message.Attempts)
	})

	t.Run("Message with Metadata", func(t *testing.T) {
		metadata := map[string]string{
			"source":   "api",
			"trace_id": "trace-123",
			"user_id":  "user-456",
		}

		message := &Message{
			ID:        "test-456",
			QueueName: "test-queue",
			Payload:   "simple payload",
			Metadata:  metadata,
			CreatedAt: time.Now(),
		}

		assert.Equal(t, "api", message.Metadata["source"])
		assert.Equal(t, "trace-123", message.Metadata["trace_id"])
		assert.Equal(t, "user-456", message.Metadata["user_id"])
	})
}

// Test QueueConfig validation
func TestQueueConfig(t *testing.T) {
	t.Run("Valid Config", func(t *testing.T) {
		config := &QueueConfig{
			Name:              "test-queue",
			VisibilityTimeout: 30 * time.Second,
			MaxRetries:        3,
			DeadLetterQueue:   "test-dlq",
		}

		assert.Equal(t, "test-queue", config.Name)
		assert.Equal(t, 30*time.Second, config.VisibilityTimeout)
		assert.Equal(t, 3, config.MaxRetries)
		assert.Equal(t, "test-dlq", config.DeadLetterQueue)
	})

	t.Run("Config with Consumer Group", func(t *testing.T) {
		config := &QueueConfig{
			Name:              "test-queue",
			VisibilityTimeout: 30 * time.Second,
			MaxRetries:        3,
			ConsumerGroup:     "test-group",
		}

		assert.Equal(t, "test-group", config.ConsumerGroup)
	})
}

// Test EnqueueOptions
func TestEnqueueOptions(t *testing.T) {
	t.Run("Default Options", func(t *testing.T) {
		opts := &EnqueueOptions{}

		// Apply default values
		applyEnqueueDefaults(opts)

		assert.Zero(t, opts.Delay)
		assert.Empty(t, opts.MessageID)
		assert.Nil(t, opts.Metadata)
	})

	t.Run("With Delay", func(t *testing.T) {
		opts := &EnqueueOptions{
			Delay: 5 * time.Minute,
		}

		assert.Equal(t, 5*time.Minute, opts.Delay)
	})

	t.Run("With Custom Message ID", func(t *testing.T) {
		opts := &EnqueueOptions{
			MessageID: "custom-id-123",
		}

		assert.Equal(t, "custom-id-123", opts.MessageID)
	})

	t.Run("With Metadata", func(t *testing.T) {
		metadata := map[string]string{
			"source":   "test",
			"priority": "high",
		}

		opts := &EnqueueOptions{
			Metadata: metadata,
		}

		assert.Equal(t, "test", opts.Metadata["source"])
		assert.Equal(t, "high", opts.Metadata["priority"])
	})
}

// Test DequeueOptions
func TestDequeueOptions(t *testing.T) {
	t.Run("Default Options", func(t *testing.T) {
		opts := &DequeueOptions{}

		// Apply default values
		applyDequeueDefaults(opts)

		assert.Equal(t, 1, opts.BatchSize)
		assert.Equal(t, 30*time.Second, opts.VisibilityTimeout)
		assert.Empty(t, opts.ConsumerID)
	})

	t.Run("Custom Batch Size", func(t *testing.T) {
		opts := &DequeueOptions{
			BatchSize: 10,
		}

		assert.Equal(t, 10, opts.BatchSize)
	})

	t.Run("Custom Visibility Timeout", func(t *testing.T) {
		opts := &DequeueOptions{
			VisibilityTimeout: 60 * time.Second,
		}

		assert.Equal(t, 60*time.Second, opts.VisibilityTimeout)
	})

	t.Run("With Consumer ID", func(t *testing.T) {
		opts := &DequeueOptions{
			ConsumerID: "consumer-123",
		}

		assert.Equal(t, "consumer-123", opts.ConsumerID)
	})
}

// Test QueueStats
func TestQueueStats(t *testing.T) {
	t.Run("Basic Stats", func(t *testing.T) {
		stats := &QueueStats{
			Name:            "test-queue",
			Length:          100,
			InFlight:        5,
			DeadLetterCount: 3,
			TotalProcessed:  1000,
			CreatedAt:       time.Now().Add(-24 * time.Hour),
		}

		assert.Equal(t, "test-queue", stats.Name)
		assert.Equal(t, int64(100), stats.Length)
		assert.Equal(t, int64(5), stats.InFlight)
		assert.Equal(t, int64(3), stats.DeadLetterCount)
		assert.Equal(t, int64(1000), stats.TotalProcessed)
	})
}

// Test error types
func TestErrors(t *testing.T) {
	t.Run("ErrQueueNotFound", func(t *testing.T) {
		err := ErrQueueNotFound
		assert.EqualError(t, err, "queue not found")
	})

	t.Run("ErrMessageNotFound", func(t *testing.T) {
		err := ErrMessageNotFound
		assert.EqualError(t, err, "message not found")
	})

	t.Run("ErrQueueExists", func(t *testing.T) {
		err := ErrQueueExists
		assert.EqualError(t, err, "queue already exists")
	})

	t.Run("ErrInvalidMessage", func(t *testing.T) {
		err := ErrInvalidMessage
		assert.EqualError(t, err, "invalid message")
	})
}

// Test EnqueueResult
func TestEnqueueResult(t *testing.T) {
	t.Run("Single Message Result", func(t *testing.T) {
		result := &EnqueueResult{
			MessageID: "msg-123",
			QueuedAt:  time.Now(),
		}

		assert.Equal(t, "msg-123", result.MessageID)
		assert.False(t, result.QueuedAt.IsZero())
	})
}

// Test BatchEnqueueResult
func TestBatchEnqueueResult(t *testing.T) {
	t.Run("Batch Result", func(t *testing.T) {
		messageIDs := []string{"msg-1", "msg-2", "msg-3"}
		result := &BatchEnqueueResult{
			MessageIDs: messageIDs,
			QueuedAt:   time.Now(),
		}

		assert.Len(t, result.MessageIDs, 3)
		assert.Contains(t, result.MessageIDs, "msg-1")
		assert.Contains(t, result.MessageIDs, "msg-2")
		assert.Contains(t, result.MessageIDs, "msg-3")
		assert.False(t, result.QueuedAt.IsZero())
	})
}

// Helper function to apply default enqueue options
func applyEnqueueDefaults(opts *EnqueueOptions) {
	if opts.Delay == 0 {
		opts.Delay = 0
	}
	if opts.MessageID == "" {
		opts.MessageID = ""
	}
	if opts.Metadata == nil {
		opts.Metadata = nil
	}
}

// Helper function to apply default dequeue options
func applyDequeueDefaults(opts *DequeueOptions) {
	if opts.BatchSize <= 0 {
		opts.BatchSize = 1
	}
	if opts.VisibilityTimeout <= 0 {
		opts.VisibilityTimeout = 30 * time.Second
	}
	if opts.ConsumerID == "" {
		opts.ConsumerID = ""
	}
}
