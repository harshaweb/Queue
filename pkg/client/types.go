package client

import (
	"context"
	"encoding/json"
	"time"

	"github.com/harshaweb/Queue/pkg/queue"
)

// MessageHandler processes a received message
type MessageHandler func(ctx context.Context, payload interface{}, msgCtx *MessageContext) *MessageResult

// MessageContext provides context about the message being processed
type MessageContext struct {
	MessageID  string
	QueueName  string
	Headers    map[string]string
	CreatedAt  time.Time
	RetryCount int
	client     *Client
}

// MessageResult indicates how to handle a processed message
type MessageResult struct {
	Action       MessageAction
	Requeue      bool
	Delay        *time.Duration
	SkipDuration *time.Duration
}

// MessageAction defines possible actions after processing a message
type MessageAction int

const (
	ActionAck MessageAction = iota
	ActionNack
	ActionRequeue
	ActionMoveToLast
	ActionSkip
	ActionDeadLetter
)

// Helper methods for common message results
func Ack() *MessageResult {
	return &MessageResult{Action: ActionAck}
}

func Nack(requeue bool) *MessageResult {
	return &MessageResult{Action: ActionNack, Requeue: requeue}
}

func Requeue(delay *time.Duration) *MessageResult {
	return &MessageResult{Action: ActionRequeue, Delay: delay}
}

func MoveToLast() *MessageResult {
	return &MessageResult{Action: ActionMoveToLast}
}

func Skip(duration time.Duration) *MessageResult {
	return &MessageResult{Action: ActionSkip, SkipDuration: &duration}
}

func DeadLetter() *MessageResult {
	return &MessageResult{Action: ActionDeadLetter}
}

// EnqueueOptions contains options for enqueuing messages
type EnqueueOptions struct {
	Priority   int
	ScheduleAt *time.Time
	Delay      *time.Duration
	TTL        *time.Duration
	Headers    map[string]string
}

// EnqueueOption is a function that configures EnqueueOptions
type EnqueueOption func(*EnqueueOptions)

func WithPriority(priority int) EnqueueOption {
	return func(opts *EnqueueOptions) {
		opts.Priority = priority
	}
}

func WithScheduleAt(scheduleAt time.Time) EnqueueOption {
	return func(opts *EnqueueOptions) {
		opts.ScheduleAt = &scheduleAt
	}
}

func WithDelay(delay time.Duration) EnqueueOption {
	return func(opts *EnqueueOptions) {
		opts.Delay = &delay
	}
}

func WithTTL(ttl time.Duration) EnqueueOption {
	return func(opts *EnqueueOptions) {
		opts.TTL = &ttl
	}
}

func WithHeaders(headers map[string]string) EnqueueOption {
	return func(opts *EnqueueOptions) {
		opts.Headers = headers
	}
}

func WithHeader(key, value string) EnqueueOption {
	return func(opts *EnqueueOptions) {
		if opts.Headers == nil {
			opts.Headers = make(map[string]string)
		}
		opts.Headers[key] = value
	}
}

// ConsumeOptions contains options for consuming messages
type ConsumeOptions struct {
	BatchSize         int
	VisibilityTimeout time.Duration
	MaxRetries        int
	BackoffPolicy     queue.BackoffPolicy
	LongPoll          bool
	LongPollTimeout   time.Duration
	ErrorHandler      func(error) bool // Return true to continue, false to stop
}

// ConsumeOption is a function that configures ConsumeOptions
type ConsumeOption func(*ConsumeOptions)

func WithBatchSize(batchSize int) ConsumeOption {
	return func(opts *ConsumeOptions) {
		opts.BatchSize = batchSize
	}
}

func WithVisibilityTimeout(timeout time.Duration) ConsumeOption {
	return func(opts *ConsumeOptions) {
		opts.VisibilityTimeout = timeout
	}
}

func WithMaxRetries(maxRetries int) ConsumeOption {
	return func(opts *ConsumeOptions) {
		opts.MaxRetries = maxRetries
	}
}

func WithBackoffPolicy(policy queue.BackoffPolicy) ConsumeOption {
	return func(opts *ConsumeOptions) {
		opts.BackoffPolicy = policy
	}
}

func WithLongPoll(timeout time.Duration) ConsumeOption {
	return func(opts *ConsumeOptions) {
		opts.LongPoll = true
		opts.LongPollTimeout = timeout
	}
}

func WithErrorHandler(handler func(error) bool) ConsumeOption {
	return func(opts *ConsumeOptions) {
		opts.ErrorHandler = handler
	}
}

// Result types

type EnqueueResult struct {
	MessageID string `json:"message_id"`
	QueueName string `json:"queue_name"`
}

type BatchEnqueueResult struct {
	MessageIDs []string `json:"message_ids"`
	QueueName  string   `json:"queue_name"`
	Count      int      `json:"count"`
}

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

type QueueConfig struct {
	Name              string              `json:"name"`
	VisibilityTimeout time.Duration       `json:"visibility_timeout"`
	MaxRetries        int                 `json:"max_retries"`
	DeadLetterEnabled bool                `json:"dead_letter_enabled"`
	DeadLetterQueue   string              `json:"dead_letter_queue"`
	BackoffPolicy     queue.BackoffPolicy `json:"backoff_policy"`
	MaxConsumers      int                 `json:"max_consumers"`
	MessageRetention  time.Duration       `json:"message_retention"`
}

// MoveAction defines message move operations
type MoveAction int

const (
	MoveActionToLast MoveAction = iota
	MoveActionToDeadLetter
	MoveActionToDifferentQueue
)

// Serializer interface for message payload serialization
type Serializer interface {
	Serialize(data interface{}) (map[string]interface{}, error)
	Deserialize(data map[string]interface{}, target interface{}) error
}

// JSONSerializer implements JSON serialization
type JSONSerializer struct{}

func (s *JSONSerializer) Serialize(data interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"_type": "json",
		"data":  string(jsonData),
	}, nil
}

func (s *JSONSerializer) Deserialize(data map[string]interface{}, target interface{}) error {
	dataStr, ok := data["data"].(string)
	if !ok {
		return json.Unmarshal([]byte("{}"), target)
	}

	return json.Unmarshal([]byte(dataStr), target)
}

// Middleware interface for extending client functionality
type Middleware interface {
	Before(ctx context.Context, operation, queueName string) context.Context
	After(ctx context.Context, operation, queueName string, err error) context.Context
}

// TraceProvider interface for distributed tracing
type TraceProvider interface {
	StartSpan(ctx context.Context, operationName string) (context.Context, func())
	InjectHeaders(ctx context.Context) map[string]string
	ExtractHeaders(headers map[string]string) context.Context
}
