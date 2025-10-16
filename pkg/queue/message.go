package queue

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Message represents a queue message with metadata
type Message struct {
	ID         string                 `json:"id"`
	Payload    map[string]interface{} `json:"payload"`
	Headers    map[string]string      `json:"headers"`
	CreatedAt  time.Time              `json:"created_at"`
	ScheduleAt *time.Time             `json:"schedule_at,omitempty"`
	RetryCount int                    `json:"retry_count"`
	QueueName  string                 `json:"queue_name"`
}

// NewMessage creates a new message with a UUID
func NewMessage(payload map[string]interface{}, headers map[string]string) *Message {
	return &Message{
		ID:         uuid.New().String(),
		Payload:    payload,
		Headers:    headers,
		CreatedAt:  time.Now(),
		RetryCount: 0,
	}
}

// ToJSON serializes the message to JSON
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON deserializes a message from JSON
func FromJSON(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

// IsScheduled checks if the message is scheduled for future delivery
func (m *Message) IsScheduled() bool {
	return m.ScheduleAt != nil && m.ScheduleAt.After(time.Now())
}

// MessageOptions contains options for message operations
type MessageOptions struct {
	Priority   int
	ScheduleAt *time.Time
	Delay      *time.Duration
	TTL        *time.Duration
	Headers    map[string]string
}

// ConsumeOptions contains options for consuming messages
type ConsumeOptions struct {
	VisibilityTimeout time.Duration
	MaxRetries        int
	BackoffPolicy     BackoffPolicy
	BatchSize         int
	LongPoll          bool
	LongPollTimeout   time.Duration
}

// BackoffPolicy defines retry backoff strategies
type BackoffPolicy struct {
	Type        BackoffType   `json:"type"`
	InitialWait time.Duration `json:"initial_wait"`
	MaxWait     time.Duration `json:"max_wait"`
	Multiplier  float64       `json:"multiplier"`
}

type BackoffType string

const (
	BackoffExponential BackoffType = "exponential"
	BackoffLinear      BackoffType = "linear"
	BackoffFixed       BackoffType = "fixed"
)

// CalculateNextDelay calculates the next retry delay
func (bp *BackoffPolicy) CalculateNextDelay(retryCount int) time.Duration {
	switch bp.Type {
	case BackoffExponential:
		delay := time.Duration(float64(bp.InitialWait) * pow(bp.Multiplier, float64(retryCount)))
		if delay > bp.MaxWait {
			return bp.MaxWait
		}
		return delay
	case BackoffLinear:
		delay := bp.InitialWait + time.Duration(retryCount)*bp.InitialWait
		if delay > bp.MaxWait {
			return bp.MaxWait
		}
		return delay
	case BackoffFixed:
		return bp.InitialWait
	default:
		return bp.InitialWait
	}
}

func pow(base float64, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	result := base
	for i := 1; i < int(exp); i++ {
		result *= base
	}
	return result
}

// QueueStats represents queue statistics
type QueueStats struct {
	Name           string    `json:"name"`
	Length         int64     `json:"length"`
	InFlight       int64     `json:"in_flight"`
	Processed      int64     `json:"processed"`
	Failed         int64     `json:"failed"`
	ConsumerCount  int       `json:"consumer_count"`
	LastActivity   time.Time `json:"last_activity"`
	DeadLetterSize int64     `json:"dead_letter_size"`
}

// QueueConfig represents queue configuration
type QueueConfig struct {
	Name              string        `json:"name"`
	VisibilityTimeout time.Duration `json:"visibility_timeout"`
	MaxRetries        int           `json:"max_retries"`
	DeadLetterEnabled bool          `json:"dead_letter_enabled"`
	DeadLetterQueue   string        `json:"dead_letter_queue"`
	BackoffPolicy     BackoffPolicy `json:"backoff_policy"`
	MaxConsumers      int           `json:"max_consumers"`
	MessageRetention  time.Duration `json:"message_retention"`
}
