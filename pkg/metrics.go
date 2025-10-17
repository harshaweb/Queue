package pkg

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics provides comprehensive queue monitoring
type Metrics struct {
	queueName string

	// Counters
	messagesSent      int64
	messagesReceived  int64
	messagesProcessed int64
	processingErrors  int64

	// Timing
	totalProcessingTime int64
	minProcessingTime   int64
	maxProcessingTime   int64

	// Rate tracking
	lastResetTime time.Time
	rateWindow    time.Duration
	rateMu        sync.RWMutex

	// Error tracking
	errorsByType map[string]int64
	errorsMu     sync.RWMutex
}

// MetricsSnapshot provides a point-in-time view of metrics
type MetricsSnapshot struct {
	QueueName          string           `json:"queue_name"`
	MessagesSent       int64            `json:"messages_sent"`
	MessagesReceived   int64            `json:"messages_received"`
	MessagesProcessed  int64            `json:"messages_processed"`
	ProcessingErrors   int64            `json:"processing_errors"`
	AverageProcessTime time.Duration    `json:"average_process_time"`
	MinProcessTime     time.Duration    `json:"min_process_time"`
	MaxProcessTime     time.Duration    `json:"max_process_time"`
	Throughput         float64          `json:"throughput_per_second"`
	ErrorRate          float64          `json:"error_rate"`
	ErrorsByType       map[string]int64 `json:"errors_by_type"`
	Timestamp          time.Time        `json:"timestamp"`
}

// NewMetrics creates a new metrics instance
func NewMetrics(queueName string) *Metrics {
	return &Metrics{
		queueName:         queueName,
		lastResetTime:     time.Now(),
		rateWindow:        time.Minute,
		errorsByType:      make(map[string]int64),
		minProcessingTime: int64(^uint64(0) >> 1), // Max int64
	}
}

// IncrementSent increments the sent message counter
func (m *Metrics) IncrementSent() {
	atomic.AddInt64(&m.messagesSent, 1)
}

// AddSent adds to the sent message counter
func (m *Metrics) AddSent(count int) {
	atomic.AddInt64(&m.messagesSent, int64(count))
}

// IncrementReceived increments the received message counter
func (m *Metrics) IncrementReceived() {
	atomic.AddInt64(&m.messagesReceived, 1)
}

// IncrementProcessed increments the processed message counter
func (m *Metrics) IncrementProcessed() {
	atomic.AddInt64(&m.messagesProcessed, 1)
}

// IncrementErrors increments the error counter with type
func (m *Metrics) IncrementErrors(errorType string) {
	atomic.AddInt64(&m.processingErrors, 1)

	m.errorsMu.Lock()
	m.errorsByType[errorType]++
	m.errorsMu.Unlock()
}

// RecordProcessingTime records message processing time
func (m *Metrics) RecordProcessingTime(duration time.Duration) {
	nanoseconds := duration.Nanoseconds()

	// Update total processing time
	atomic.AddInt64(&m.totalProcessingTime, nanoseconds)

	// Update min processing time
	for {
		current := atomic.LoadInt64(&m.minProcessingTime)
		if nanoseconds >= current {
			break
		}
		if atomic.CompareAndSwapInt64(&m.minProcessingTime, current, nanoseconds) {
			break
		}
	}

	// Update max processing time
	for {
		current := atomic.LoadInt64(&m.maxProcessingTime)
		if nanoseconds <= current {
			break
		}
		if atomic.CompareAndSwapInt64(&m.maxProcessingTime, current, nanoseconds) {
			break
		}
	}
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() *MetricsSnapshot {
	sent := atomic.LoadInt64(&m.messagesSent)
	received := atomic.LoadInt64(&m.messagesReceived)
	processed := atomic.LoadInt64(&m.messagesProcessed)
	errors := atomic.LoadInt64(&m.processingErrors)
	totalTime := atomic.LoadInt64(&m.totalProcessingTime)
	minTime := atomic.LoadInt64(&m.minProcessingTime)
	maxTime := atomic.LoadInt64(&m.maxProcessingTime)

	// Calculate average processing time
	var avgTime time.Duration
	if processed > 0 {
		avgTime = time.Duration(totalTime / processed)
	}

	// Calculate throughput
	m.rateMu.RLock()
	elapsed := time.Since(m.lastResetTime)
	m.rateMu.RUnlock()

	var throughput float64
	if elapsed.Seconds() > 0 {
		throughput = float64(processed) / elapsed.Seconds()
	}

	// Calculate error rate
	var errorRate float64
	if processed > 0 {
		errorRate = float64(errors) / float64(processed)
	}

	// Copy errors by type
	m.errorsMu.RLock()
	errorsByType := make(map[string]int64)
	for k, v := range m.errorsByType {
		errorsByType[k] = v
	}
	m.errorsMu.RUnlock()

	// Handle min time edge case
	minTimeDuration := time.Duration(minTime)
	if minTime == int64(^uint64(0)>>1) {
		minTimeDuration = 0
	}

	return &MetricsSnapshot{
		QueueName:          m.queueName,
		MessagesSent:       sent,
		MessagesReceived:   received,
		MessagesProcessed:  processed,
		ProcessingErrors:   errors,
		AverageProcessTime: avgTime,
		MinProcessTime:     minTimeDuration,
		MaxProcessTime:     time.Duration(maxTime),
		Throughput:         throughput,
		ErrorRate:          errorRate,
		ErrorsByType:       errorsByType,
		Timestamp:          time.Now(),
	}
}

// Reset resets all metrics counters
func (m *Metrics) Reset() {
	atomic.StoreInt64(&m.messagesSent, 0)
	atomic.StoreInt64(&m.messagesReceived, 0)
	atomic.StoreInt64(&m.messagesProcessed, 0)
	atomic.StoreInt64(&m.processingErrors, 0)
	atomic.StoreInt64(&m.totalProcessingTime, 0)
	atomic.StoreInt64(&m.minProcessingTime, int64(^uint64(0)>>1))
	atomic.StoreInt64(&m.maxProcessingTime, 0)

	m.rateMu.Lock()
	m.lastResetTime = time.Now()
	m.rateMu.Unlock()

	m.errorsMu.Lock()
	m.errorsByType = make(map[string]int64)
	m.errorsMu.Unlock()
}

// GetThroughput calculates current throughput
func (m *Metrics) GetThroughput() float64 {
	processed := atomic.LoadInt64(&m.messagesProcessed)

	m.rateMu.RLock()
	elapsed := time.Since(m.lastResetTime)
	m.rateMu.RUnlock()

	if elapsed.Seconds() > 0 {
		return float64(processed) / elapsed.Seconds()
	}
	return 0
}

// GetErrorRate calculates current error rate
func (m *Metrics) GetErrorRate() float64 {
	processed := atomic.LoadInt64(&m.messagesProcessed)
	errors := atomic.LoadInt64(&m.processingErrors)

	if processed > 0 {
		return float64(errors) / float64(processed)
	}
	return 0
}

// HealthStatus represents queue health
type HealthStatus struct {
	Healthy         bool      `json:"healthy"`
	QueueName       string    `json:"queue_name"`
	RedisConnected  bool      `json:"redis_connected"`
	ConsumerCount   int       `json:"consumer_count"`
	PendingMessages int64     `json:"pending_messages"`
	ProcessingRate  float64   `json:"processing_rate"`
	ErrorRate       float64   `json:"error_rate"`
	LastActivity    time.Time `json:"last_activity"`
	Issues          []string  `json:"issues,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

// PerformanceMonitor tracks performance trends
type PerformanceMonitor struct {
	metrics    *Metrics
	samples    []MetricsSnapshot
	maxSamples int
	sampleRate time.Duration
	mu         sync.RWMutex
	ticker     *time.Ticker
	stopCh     chan struct{}
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(metrics *Metrics, sampleRate time.Duration) *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics:    metrics,
		maxSamples: 1000, // Keep last 1000 samples
		sampleRate: sampleRate,
		stopCh:     make(chan struct{}),
	}
}

// Start begins performance monitoring
func (pm *PerformanceMonitor) Start() {
	pm.ticker = time.NewTicker(pm.sampleRate)

	go func() {
		for {
			select {
			case <-pm.ticker.C:
				pm.recordSample()
			case <-pm.stopCh:
				return
			}
		}
	}()
}

// Stop stops performance monitoring
func (pm *PerformanceMonitor) Stop() {
	if pm.ticker != nil {
		pm.ticker.Stop()
	}
	close(pm.stopCh)
}

// recordSample records a metrics snapshot
func (pm *PerformanceMonitor) recordSample() {
	snapshot := pm.metrics.GetSnapshot()

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.samples = append(pm.samples, *snapshot)

	// Keep only the most recent samples
	if len(pm.samples) > pm.maxSamples {
		pm.samples = pm.samples[1:]
	}
}

// GetTrend returns performance trend data
func (pm *PerformanceMonitor) GetTrend(duration time.Duration) []MetricsSnapshot {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	var trend []MetricsSnapshot

	for _, sample := range pm.samples {
		if sample.Timestamp.After(cutoff) {
			trend = append(trend, sample)
		}
	}

	return trend
}

// GetAverageThroughput calculates average throughput over duration
func (pm *PerformanceMonitor) GetAverageThroughput(duration time.Duration) float64 {
	trend := pm.GetTrend(duration)
	if len(trend) == 0 {
		return 0
	}

	var total float64
	for _, sample := range trend {
		total += sample.Throughput
	}

	return total / float64(len(trend))
}

// GetPeakThroughput returns the highest throughput in the given duration
func (pm *PerformanceMonitor) GetPeakThroughput(duration time.Duration) float64 {
	trend := pm.GetTrend(duration)
	if len(trend) == 0 {
		return 0
	}

	var peak float64
	for _, sample := range trend {
		if sample.Throughput > peak {
			peak = sample.Throughput
		}
	}

	return peak
}

// AlertThresholds defines thresholds for alerts
type AlertThresholds struct {
	MaxErrorRate       float64       `json:"max_error_rate"`
	MinThroughput      float64       `json:"min_throughput"`
	MaxProcessingTime  time.Duration `json:"max_processing_time"`
	MaxPendingMessages int64         `json:"max_pending_messages"`
}

// CheckAlerts checks if any alert thresholds are exceeded
func (m *Metrics) CheckAlerts(thresholds *AlertThresholds) []string {
	snapshot := m.GetSnapshot()
	var alerts []string

	if snapshot.ErrorRate > thresholds.MaxErrorRate {
		alerts = append(alerts, fmt.Sprintf("High error rate: %.2f%% (threshold: %.2f%%)",
			snapshot.ErrorRate*100, thresholds.MaxErrorRate*100))
	}

	if snapshot.Throughput < thresholds.MinThroughput {
		alerts = append(alerts, fmt.Sprintf("Low throughput: %.2f msg/s (threshold: %.2f msg/s)",
			snapshot.Throughput, thresholds.MinThroughput))
	}

	if snapshot.AverageProcessTime > thresholds.MaxProcessingTime {
		alerts = append(alerts, fmt.Sprintf("High processing time: %v (threshold: %v)",
			snapshot.AverageProcessTime, thresholds.MaxProcessingTime))
	}

	return alerts
}
