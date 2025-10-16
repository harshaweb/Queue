package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the queue system
type Metrics struct {
	// Queue operation metrics
	OperationsTotal   *prometheus.CounterVec
	OperationDuration *prometheus.HistogramVec
	ErrorsTotal       *prometheus.CounterVec

	// Queue state metrics
	QueueLength      *prometheus.GaugeVec
	InFlightMessages *prometheus.GaugeVec
	ConsumerCount    *prometheus.GaugeVec
	DeadLetterSize   *prometheus.GaugeVec

	// Message processing metrics
	MessageProcessingTime *prometheus.HistogramVec
	MessageRetries        *prometheus.HistogramVec
	MessageAge            *prometheus.HistogramVec

	// Consumer metrics
	ConsumerConnections *prometheus.GaugeVec
	ConsumerLatency     *prometheus.HistogramVec

	// Redis metrics
	RedisConnections     *prometheus.GaugeVec
	RedisCommandDuration *prometheus.HistogramVec
	RedisErrors          *prometheus.CounterVec

	// System metrics
	ActiveQueues  prometheus.Gauge
	TotalMessages prometheus.Counter
	SystemUptime  prometheus.Gauge
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	return &Metrics{
		// Queue operation metrics
		OperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_operations_total",
				Help: "Total number of queue operations",
			},
			[]string{"operation", "queue", "status"},
		),

		OperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "queue_operation_duration_seconds",
				Help:    "Duration of queue operations",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"operation", "queue"},
		),

		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_errors_total",
				Help: "Total number of queue operation errors",
			},
			[]string{"operation", "queue", "error_type"},
		),

		// Queue state metrics
		QueueLength: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "queue_length",
				Help: "Current number of messages in queue",
			},
			[]string{"queue"},
		),

		InFlightMessages: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "queue_inflight_messages",
				Help: "Number of messages currently being processed",
			},
			[]string{"queue"},
		),

		ConsumerCount: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "queue_consumers",
				Help: "Number of active consumers for queue",
			},
			[]string{"queue"},
		),

		DeadLetterSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "queue_dead_letter_size",
				Help: "Number of messages in dead letter queue",
			},
			[]string{"queue"},
		),

		// Message processing metrics
		MessageProcessingTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "queue_message_processing_seconds",
				Help:    "Time spent processing messages",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
			},
			[]string{"queue", "status"},
		),

		MessageRetries: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "queue_message_retries",
				Help:    "Number of retries per message",
				Buckets: []float64{0, 1, 2, 3, 4, 5, 10, 20},
			},
			[]string{"queue"},
		),

		MessageAge: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "queue_message_age_seconds",
				Help:    "Age of messages when processed",
				Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600, 7200},
			},
			[]string{"queue"},
		),

		// Consumer metrics
		ConsumerConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "queue_consumer_connections",
				Help: "Number of active consumer connections",
			},
			[]string{"queue", "consumer_group"},
		),

		ConsumerLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "queue_consumer_latency_seconds",
				Help:    "Consumer polling latency",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"queue", "consumer_group"},
		),

		// Redis metrics
		RedisConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "queue_redis_connections",
				Help: "Number of active Redis connections",
			},
			[]string{"instance"},
		),

		RedisCommandDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "queue_redis_command_duration_seconds",
				Help:    "Duration of Redis commands",
				Buckets: []float64{.0001, .0005, .001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"command", "instance"},
		),

		RedisErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "queue_redis_errors_total",
				Help: "Total number of Redis errors",
			},
			[]string{"command", "instance", "error_type"},
		),

		// System metrics
		ActiveQueues: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "queue_active_queues",
				Help: "Number of active queues",
			},
		),

		TotalMessages: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "queue_messages_total",
				Help: "Total number of messages processed",
			},
		),

		SystemUptime: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "queue_system_uptime_seconds",
				Help: "System uptime in seconds",
			},
		),
	}
}

// RecordOperation records a queue operation with its duration and status
func (m *Metrics) RecordOperation(operation, queue, status string, duration time.Duration) {
	m.OperationsTotal.WithLabelValues(operation, queue, status).Inc()
	m.OperationDuration.WithLabelValues(operation, queue).Observe(duration.Seconds())
}

// RecordError records an error for a specific operation
func (m *Metrics) RecordError(operation, queue, errorType string) {
	m.ErrorsTotal.WithLabelValues(operation, queue, errorType).Inc()
}

// UpdateQueueLength updates the current queue length
func (m *Metrics) UpdateQueueLength(queue string, length float64) {
	m.QueueLength.WithLabelValues(queue).Set(length)
}

// UpdateInFlightMessages updates the number of in-flight messages
func (m *Metrics) UpdateInFlightMessages(queue string, count float64) {
	m.InFlightMessages.WithLabelValues(queue).Set(count)
}

// UpdateConsumerCount updates the number of active consumers
func (m *Metrics) UpdateConsumerCount(queue string, count float64) {
	m.ConsumerCount.WithLabelValues(queue).Set(count)
}

// UpdateDeadLetterSize updates the dead letter queue size
func (m *Metrics) UpdateDeadLetterSize(queue string, size float64) {
	m.DeadLetterSize.WithLabelValues(queue).Set(size)
}

// RecordMessageProcessing records message processing time and status
func (m *Metrics) RecordMessageProcessing(queue, status string, duration time.Duration) {
	m.MessageProcessingTime.WithLabelValues(queue, status).Observe(duration.Seconds())
}

// RecordMessageRetries records the number of retries for a message
func (m *Metrics) RecordMessageRetries(queue string, retries float64) {
	m.MessageRetries.WithLabelValues(queue).Observe(retries)
}

// RecordMessageAge records the age of a message when processed
func (m *Metrics) RecordMessageAge(queue string, age time.Duration) {
	m.MessageAge.WithLabelValues(queue).Observe(age.Seconds())
}

// UpdateConsumerConnections updates the number of consumer connections
func (m *Metrics) UpdateConsumerConnections(queue, consumerGroup string, count float64) {
	m.ConsumerConnections.WithLabelValues(queue, consumerGroup).Set(count)
}

// RecordConsumerLatency records consumer polling latency
func (m *Metrics) RecordConsumerLatency(queue, consumerGroup string, latency time.Duration) {
	m.ConsumerLatency.WithLabelValues(queue, consumerGroup).Observe(latency.Seconds())
}

// UpdateRedisConnections updates the number of Redis connections
func (m *Metrics) UpdateRedisConnections(instance string, count float64) {
	m.RedisConnections.WithLabelValues(instance).Set(count)
}

// RecordRedisCommand records Redis command execution time
func (m *Metrics) RecordRedisCommand(command, instance string, duration time.Duration) {
	m.RedisCommandDuration.WithLabelValues(command, instance).Observe(duration.Seconds())
}

// RecordRedisError records a Redis error
func (m *Metrics) RecordRedisError(command, instance, errorType string) {
	m.RedisErrors.WithLabelValues(command, instance, errorType).Inc()
}

// UpdateActiveQueues updates the number of active queues
func (m *Metrics) UpdateActiveQueues(count float64) {
	m.ActiveQueues.Set(count)
}

// IncrementTotalMessages increments the total message counter
func (m *Metrics) IncrementTotalMessages() {
	m.TotalMessages.Inc()
}

// UpdateSystemUptime updates the system uptime
func (m *Metrics) UpdateSystemUptime(uptime time.Duration) {
	m.SystemUptime.Set(uptime.Seconds())
}

// MetricsCollector provides an interface for collecting custom metrics
type MetricsCollector interface {
	Collect() map[string]float64
}

// CustomMetrics allows for application-specific metrics
type CustomMetrics struct {
	collectors []MetricsCollector
	gauges     map[string]*prometheus.GaugeVec
}

// NewCustomMetrics creates a new custom metrics instance
func NewCustomMetrics() *CustomMetrics {
	return &CustomMetrics{
		collectors: make([]MetricsCollector, 0),
		gauges:     make(map[string]*prometheus.GaugeVec),
	}
}

// RegisterCollector registers a custom metrics collector
func (cm *CustomMetrics) RegisterCollector(collector MetricsCollector) {
	cm.collectors = append(cm.collectors, collector)
}

// RegisterGauge registers a custom gauge metric
func (cm *CustomMetrics) RegisterGauge(name, help string, labels []string) *prometheus.GaugeVec {
	gauge := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
	cm.gauges[name] = gauge
	return gauge
}

// CollectCustomMetrics collects all custom metrics
func (cm *CustomMetrics) CollectCustomMetrics() {
	for _, collector := range cm.collectors {
		metrics := collector.Collect()
		for name, value := range metrics {
			if gauge, exists := cm.gauges[name]; exists {
				gauge.WithLabelValues().Set(value)
			}
		}
	}
}
