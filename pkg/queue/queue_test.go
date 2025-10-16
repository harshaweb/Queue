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

func (m *MockQueue) Enqueue(ctx context.Context, queueName string, message *Message, opts *MessageOptions) error {
	args := m.Called(ctx, queueName, message, opts)
	return args.Error(0)
}

func (m *MockQueue) Dequeue(ctx context.Context, queueName string, opts *ConsumeOptions) (*Message, error) {
	args := m.Called(ctx, queueName, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Message), args.Error(1)
}

func (m *MockQueue) Ack(ctx context.Context, queueName string, messageID string) error {
	args := m.Called(ctx, queueName, messageID)
	return args.Error(0)
}

func (m *MockQueue) Nack(ctx context.Context, queueName string, messageID string, requeue bool) error {
	args := m.Called(ctx, queueName, messageID, requeue)
	return args.Error(0)
}

func (m *MockQueue) MoveToLast(ctx context.Context, queueName string, messageID string) error {
	args := m.Called(ctx, queueName, messageID)
	return args.Error(0)
}

func (m *MockQueue) Skip(ctx context.Context, queueName string, messageID string, duration *time.Duration) error {
	args := m.Called(ctx, queueName, messageID, duration)
	return args.Error(0)
}

func (m *MockQueue) MoveToDeadLetter(ctx context.Context, queueName string, messageID string) error {
	args := m.Called(ctx, queueName, messageID)
	return args.Error(0)
}

func (m *MockQueue) Requeue(ctx context.Context, queueName string, messageID string, delay *time.Duration) error {
	args := m.Called(ctx, queueName, messageID, delay)
	return args.Error(0)
}

func (m *MockQueue) CreateQueue(ctx context.Context, config *QueueConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockQueue) DeleteQueue(ctx context.Context, queueName string) error {
	args := m.Called(ctx, queueName)
	return args.Error(0)
}

func (m *MockQueue) GetQueueStats(ctx context.Context, queueName string) (*QueueStats, error) {
	args := m.Called(ctx, queueName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*QueueStats), args.Error(1)
}

func (m *MockQueue) ListQueues(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockQueue) PauseQueue(ctx context.Context, queueName string) error {
	args := m.Called(ctx, queueName)
	return args.Error(0)
}

func (m *MockQueue) ResumeQueue(ctx context.Context, queueName string) error {
	args := m.Called(ctx, queueName)
	return args.Error(0)
}

func (m *MockQueue) GetConsumerGroups(ctx context.Context, queueName string) ([]ConsumerGroup, error) {
	args := m.Called(ctx, queueName)
	return args.Get(0).([]ConsumerGroup), args.Error(1)
}

func (m *MockQueue) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockQueue) CleanupExpiredMessages(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockQueue) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Test Message creation
func TestNewMessage(t *testing.T) {
	payload := map[string]interface{}{
		"key": "value",
		"num": 42,
	}
	headers := map[string]string{
		"source": "test",
	}

	message := NewMessage(payload, headers)

	assert.NotEmpty(t, message.ID)
	assert.Equal(t, payload, message.Payload)
	assert.Equal(t, headers, message.Headers)
	assert.NotZero(t, message.CreatedAt)
	assert.Equal(t, 0, message.RetryCount)
}

// Test Mock Queue Operations
func TestMockQueueEnqueue(t *testing.T) {
	mockQueue := new(MockQueue)
	ctx := context.Background()
	queueName := "test-queue"

	message := NewMessage(map[string]interface{}{"test": "data"}, nil)
	opts := &MessageOptions{Priority: 1}

	mockQueue.On("Enqueue", ctx, queueName, message, opts).Return(nil)

	err := mockQueue.Enqueue(ctx, queueName, message, opts)
	assert.NoError(t, err)
	mockQueue.AssertExpectations(t)
}

func TestMockQueueHealthCheck(t *testing.T) {
	mockQueue := new(MockQueue)
	ctx := context.Background()

	mockQueue.On("HealthCheck", ctx).Return(nil)

	err := mockQueue.HealthCheck(ctx)
	assert.NoError(t, err)
	mockQueue.AssertExpectations(t)
}
