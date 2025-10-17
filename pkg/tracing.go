package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// MessageTracing provides comprehensive message tracing and observability
type MessageTracing struct {
	rdb       *redis.Client
	queueName string
	enabled   bool
	mu        sync.RWMutex
}

// TraceEvent represents a single trace event in message lifecycle
type TraceEvent struct {
	ID           string                 `json:"id"`
	MessageID    string                 `json:"message_id"`
	QueueName    string                 `json:"queue_name"`
	Event        string                 `json:"event"`
	Timestamp    time.Time              `json:"timestamp"`
	Consumer     string                 `json:"consumer,omitempty"`
	Duration     time.Duration          `json:"duration,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	SpanID       string                 `json:"span_id,omitempty"`
	TraceID      string                 `json:"trace_id,omitempty"`
	ParentSpanID string                 `json:"parent_span_id,omitempty"`
}

// MessageTrace represents the complete trace of a message
type MessageTrace struct {
	MessageID     string        `json:"message_id"`
	QueueName     string        `json:"queue_name"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	TotalDuration time.Duration `json:"total_duration"`
	Events        []TraceEvent  `json:"events"`
	Status        string        `json:"status"` // "processing", "completed", "failed", "timeout"
	RetryCount    int           `json:"retry_count"`
}

// TracingConfig configures message tracing behavior
type TracingConfig struct {
	Enabled           bool          `json:"enabled"`
	SampleRate        float64       `json:"sample_rate"`          // 0.0 to 1.0
	MaxTraceAge       time.Duration `json:"max_trace_age"`        // How long to keep traces
	MaxTracesPerQueue int           `json:"max_traces_per_queue"` // Max traces to keep per queue
	IncludePayload    bool          `json:"include_payload"`      // Whether to include message payload in traces
}

// NewMessageTracing creates a new message tracing instance
func NewMessageTracing(rdb *redis.Client, queueName string, config TracingConfig) *MessageTracing {
	if config.SampleRate == 0 {
		config.SampleRate = 1.0 // Default to trace all messages
	}
	if config.MaxTraceAge == 0 {
		config.MaxTraceAge = 24 * time.Hour // Default to 24 hours
	}
	if config.MaxTracesPerQueue == 0 {
		config.MaxTracesPerQueue = 10000 // Default to 10k traces
	}

	return &MessageTracing{
		rdb:       rdb,
		queueName: queueName,
		enabled:   config.Enabled,
	}
}

// StartTrace begins tracing for a message
func (mt *MessageTracing) StartTrace(ctx context.Context, messageID string, metadata map[string]interface{}) (string, error) {
	if !mt.enabled {
		return "", nil
	}

	traceID := generateTraceID()
	spanID := generateSpanID()

	event := TraceEvent{
		ID:        generateEventID(),
		MessageID: messageID,
		QueueName: mt.queueName,
		Event:     "message_received",
		Timestamp: time.Now(),
		TraceID:   traceID,
		SpanID:    spanID,
		Metadata:  metadata,
	}

	trace := MessageTrace{
		MessageID: messageID,
		QueueName: mt.queueName,
		StartTime: time.Now(),
		Events:    []TraceEvent{event},
		Status:    "processing",
	}

	return traceID, mt.saveTrace(ctx, &trace)
}

// AddEvent adds an event to a message trace
func (mt *MessageTracing) AddEvent(ctx context.Context, messageID, traceID string, event TraceEvent) error {
	if !mt.enabled || traceID == "" {
		return nil
	}

	event.ID = generateEventID()
	event.MessageID = messageID
	event.QueueName = mt.queueName
	event.TraceID = traceID
	event.Timestamp = time.Now()

	// Get existing trace
	trace, err := mt.getTrace(ctx, messageID)
	if err != nil {
		return err
	}

	// Add event
	trace.Events = append(trace.Events, event)

	// Update status based on event
	switch event.Event {
	case "message_processed":
		trace.Status = "completed"
		trace.EndTime = time.Now()
		trace.TotalDuration = trace.EndTime.Sub(trace.StartTime)
	case "message_failed":
		trace.Status = "failed"
		trace.EndTime = time.Now()
		trace.TotalDuration = trace.EndTime.Sub(trace.StartTime)
	case "message_retry":
		trace.RetryCount++
	case "message_timeout":
		trace.Status = "timeout"
		trace.EndTime = time.Now()
		trace.TotalDuration = trace.EndTime.Sub(trace.StartTime)
	}

	return mt.saveTrace(ctx, trace)
}

// EndTrace completes a message trace
func (mt *MessageTracing) EndTrace(ctx context.Context, messageID, traceID string, status string, duration time.Duration) error {
	if !mt.enabled || traceID == "" {
		return nil
	}

	event := TraceEvent{
		Event:    fmt.Sprintf("trace_completed_%s", status),
		Duration: duration,
	}

	return mt.AddEvent(ctx, messageID, traceID, event)
}

// GetTrace retrieves a complete message trace
func (mt *MessageTracing) GetTrace(ctx context.Context, messageID string) (*MessageTrace, error) {
	if !mt.enabled {
		return nil, fmt.Errorf("tracing is disabled")
	}

	return mt.getTrace(ctx, messageID)
}

// GetTraces retrieves traces for the queue with optional filtering
func (mt *MessageTracing) GetTraces(ctx context.Context, filter TraceFilter) ([]*MessageTrace, error) {
	if !mt.enabled {
		return nil, fmt.Errorf("tracing is disabled")
	}

	key := fmt.Sprintf("%s:traces", mt.queueName)

	// Get all trace keys
	result := mt.rdb.SMembers(ctx, key)
	messageIDs, err := result.Result()
	if err != nil {
		return nil, err
	}

	var traces []*MessageTrace
	for _, messageID := range messageIDs {
		trace, err := mt.getTrace(ctx, messageID)
		if err != nil {
			continue
		}

		// Apply filters
		if filter.Status != "" && trace.Status != filter.Status {
			continue
		}
		if !filter.StartTime.IsZero() && trace.StartTime.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && trace.StartTime.After(filter.EndTime) {
			continue
		}
		if filter.MinDuration > 0 && trace.TotalDuration < filter.MinDuration {
			continue
		}
		if filter.MaxDuration > 0 && trace.TotalDuration > filter.MaxDuration {
			continue
		}

		traces = append(traces, trace)
	}

	// Limit results
	if filter.Limit > 0 && len(traces) > filter.Limit {
		traces = traces[:filter.Limit]
	}

	return traces, nil
}

// TraceFilter defines filtering options for traces
type TraceFilter struct {
	Status      string        `json:"status"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	MinDuration time.Duration `json:"min_duration"`
	MaxDuration time.Duration `json:"max_duration"`
	Limit       int           `json:"limit"`
}

// AnalyzePerformance analyzes queue performance based on traces
func (mt *MessageTracing) AnalyzePerformance(ctx context.Context, period time.Duration) (*PerformanceAnalysis, error) {
	if !mt.enabled {
		return nil, fmt.Errorf("tracing is disabled")
	}

	since := time.Now().Add(-period)
	filter := TraceFilter{
		StartTime: since,
	}

	traces, err := mt.GetTraces(ctx, filter)
	if err != nil {
		return nil, err
	}

	analysis := &PerformanceAnalysis{
		Period:        period,
		TotalMessages: len(traces),
		StartTime:     since,
		EndTime:       time.Now(),
	}

	if len(traces) == 0 {
		return analysis, nil
	}

	// Calculate statistics
	var totalDuration time.Duration
	var completedCount, failedCount, timeoutCount int
	var durations []time.Duration

	for _, trace := range traces {
		switch trace.Status {
		case "completed":
			completedCount++
		case "failed":
			failedCount++
		case "timeout":
			timeoutCount++
		}

		if trace.TotalDuration > 0 {
			totalDuration += trace.TotalDuration
			durations = append(durations, trace.TotalDuration)
		}
	}

	analysis.CompletedMessages = completedCount
	analysis.FailedMessages = failedCount
	analysis.TimeoutMessages = timeoutCount
	analysis.SuccessRate = float64(completedCount) / float64(len(traces))

	if len(durations) > 0 {
		analysis.AverageProcessingTime = totalDuration / time.Duration(len(durations))
		analysis.MinProcessingTime = minDuration(durations)
		analysis.MaxProcessingTime = maxDuration(durations)
		analysis.MedianProcessingTime = medianDuration(durations)
	}

	analysis.Throughput = float64(completedCount) / period.Seconds()

	return analysis, nil
}

// PerformanceAnalysis provides queue performance metrics
type PerformanceAnalysis struct {
	Period                time.Duration `json:"period"`
	StartTime             time.Time     `json:"start_time"`
	EndTime               time.Time     `json:"end_time"`
	TotalMessages         int           `json:"total_messages"`
	CompletedMessages     int           `json:"completed_messages"`
	FailedMessages        int           `json:"failed_messages"`
	TimeoutMessages       int           `json:"timeout_messages"`
	SuccessRate           float64       `json:"success_rate"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	MinProcessingTime     time.Duration `json:"min_processing_time"`
	MaxProcessingTime     time.Duration `json:"max_processing_time"`
	MedianProcessingTime  time.Duration `json:"median_processing_time"`
	Throughput            float64       `json:"throughput"` // messages per second
}

// saveTrace saves a trace to Redis
func (mt *MessageTracing) saveTrace(ctx context.Context, trace *MessageTrace) error {
	data, err := json.Marshal(trace)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:trace:%s", mt.queueName, trace.MessageID)
	tracesKey := fmt.Sprintf("%s:traces", mt.queueName)

	pipe := mt.rdb.Pipeline()
	pipe.Set(ctx, key, data, 24*time.Hour) // TTL for traces
	pipe.SAdd(ctx, tracesKey, trace.MessageID)
	_, err = pipe.Exec(ctx)

	return err
}

// getTrace retrieves a trace from Redis
func (mt *MessageTracing) getTrace(ctx context.Context, messageID string) (*MessageTrace, error) {
	key := fmt.Sprintf("%s:trace:%s", mt.queueName, messageID)

	result := mt.rdb.Get(ctx, key)
	data, err := result.Result()
	if err != nil {
		return nil, err
	}

	var trace MessageTrace
	err = json.Unmarshal([]byte(data), &trace)
	return &trace, err
}

// Helper functions for trace and span ID generation
func generateTraceID() string {
	return fmt.Sprintf("trace_%d_%s", time.Now().UnixNano(), generateShortID())
}

func generateSpanID() string {
	return fmt.Sprintf("span_%s", generateShortID())
}

func generateEventID() string {
	return fmt.Sprintf("event_%s", generateShortID())
}

func generateShortID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// Helper functions for duration calculations
func minDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

func maxDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}

func medianDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Simple bubble sort for small arrays
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1-i; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}
