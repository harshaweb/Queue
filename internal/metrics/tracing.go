package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingConfig contains OpenTelemetry configuration
type TracingConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	JaegerEndpoint string
	SamplingRatio  float64
	Enabled        bool
}

// TracingProvider wraps OpenTelemetry functionality
type TracingProvider struct {
	tracer   trace.Tracer
	provider *tracesdk.TracerProvider
	config   *TracingConfig
}

// NewTracingProvider creates a new tracing provider
func NewTracingProvider(config *TracingConfig) (*TracingProvider, error) {
	if !config.Enabled {
		return &TracingProvider{
			tracer: otel.GetTracerProvider().Tracer("noop"),
			config: config,
		}, nil
	}

	// Create Jaeger exporter
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	provider := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(res),
		tracesdk.WithSampler(tracesdk.TraceIDRatioBased(config.SamplingRatio)),
	)

	// Set global provider
	otel.SetTracerProvider(provider)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := provider.Tracer(
		config.ServiceName,
		trace.WithInstrumentationVersion(config.ServiceVersion),
	)

	return &TracingProvider{
		tracer:   tracer,
		provider: provider,
		config:   config,
	}, nil
}

// StartSpan starts a new span
func (tp *TracingProvider) StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tp.tracer.Start(ctx, spanName, opts...)
}

// SpanFromContext returns the span from context
func (tp *TracingProvider) SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// InjectHeaders injects tracing headers into a map
func (tp *TracingProvider) InjectHeaders(ctx context.Context) map[string]string {
	headers := make(map[string]string)
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, &MapCarrier{m: headers})
	return headers
}

// ExtractHeaders extracts tracing context from headers
func (tp *TracingProvider) ExtractHeaders(ctx context.Context, headers map[string]string) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, &MapCarrier{m: headers})
}

// Close shuts down the tracing provider
func (tp *TracingProvider) Close(ctx context.Context) error {
	if tp.provider != nil {
		return tp.provider.Shutdown(ctx)
	}
	return nil
}

// MapCarrier implements the TextMapCarrier interface for map[string]string
type MapCarrier struct {
	m map[string]string
}

func (mc *MapCarrier) Get(key string) string {
	return mc.m[key]
}

func (mc *MapCarrier) Set(key, value string) {
	mc.m[key] = value
}

func (mc *MapCarrier) Keys() []string {
	keys := make([]string, 0, len(mc.m))
	for k := range mc.m {
		keys = append(keys, k)
	}
	return keys
}

// Queue operation tracing helpers

// TraceEnqueue traces an enqueue operation
func (tp *TracingProvider) TraceEnqueue(ctx context.Context, queueName string, messageID string) (context.Context, trace.Span) {
	return tp.StartSpan(ctx, "queue.enqueue",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("queue.name", queueName),
			attribute.String("message.id", messageID),
			attribute.String("operation", "enqueue"),
		),
	)
}

// TraceDequeue traces a dequeue operation
func (tp *TracingProvider) TraceDequeue(ctx context.Context, queueName string) (context.Context, trace.Span) {
	return tp.StartSpan(ctx, "queue.dequeue",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("queue.name", queueName),
			attribute.String("operation", "dequeue"),
		),
	)
}

// TraceProcessMessage traces message processing
func (tp *TracingProvider) TraceProcessMessage(ctx context.Context, queueName string, messageID string) (context.Context, trace.Span) {
	return tp.StartSpan(ctx, "queue.process_message",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("queue.name", queueName),
			attribute.String("message.id", messageID),
			attribute.String("operation", "process"),
		),
	)
}

// TraceAck traces an ack operation
func (tp *TracingProvider) TraceAck(ctx context.Context, queueName string, messageID string) (context.Context, trace.Span) {
	return tp.StartSpan(ctx, "queue.ack",
		trace.WithAttributes(
			attribute.String("queue.name", queueName),
			attribute.String("message.id", messageID),
			attribute.String("operation", "ack"),
		),
	)
}

// TraceNack traces a nack operation
func (tp *TracingProvider) TraceNack(ctx context.Context, queueName string, messageID string, requeue bool) (context.Context, trace.Span) {
	return tp.StartSpan(ctx, "queue.nack",
		trace.WithAttributes(
			attribute.String("queue.name", queueName),
			attribute.String("message.id", messageID),
			attribute.String("operation", "nack"),
			attribute.Bool("requeue", requeue),
		),
	)
}

// TraceMoveToLast traces a move-to-last operation
func (tp *TracingProvider) TraceMoveToLast(ctx context.Context, queueName string, messageID string) (context.Context, trace.Span) {
	return tp.StartSpan(ctx, "queue.move_to_last",
		trace.WithAttributes(
			attribute.String("queue.name", queueName),
			attribute.String("message.id", messageID),
			attribute.String("operation", "move_to_last"),
		),
	)
}

// TraceSkip traces a skip operation
func (tp *TracingProvider) TraceSkip(ctx context.Context, queueName string, messageID string) (context.Context, trace.Span) {
	return tp.StartSpan(ctx, "queue.skip",
		trace.WithAttributes(
			attribute.String("queue.name", queueName),
			attribute.String("message.id", messageID),
			attribute.String("operation", "skip"),
		),
	)
}

// TraceRedisOperation traces a Redis operation
func (tp *TracingProvider) TraceRedisOperation(ctx context.Context, command string, keys []string) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", command),
	}

	if len(keys) > 0 {
		attrs = append(attrs, attribute.StringSlice("db.redis.keys", keys))
	}

	return tp.StartSpan(ctx, fmt.Sprintf("redis.%s", command),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
}

// AddSpanAttributes adds attributes to the current span
func (tp *TracingProvider) AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// AddSpanEvent adds an event to the current span
func (tp *TracingProvider) AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanStatus sets the status of the current span
func (tp *TracingProvider) SetSpanStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(code, description)
}

// RecordError records an error on the current span
func (tp *TracingProvider) RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// Middleware for instrumenting HTTP handlers
func (tp *TracingProvider) HTTPMiddleware(operation string) func(next func()) func() {
	return func(next func()) func() {
		return func() {
			// This would be implemented as actual HTTP middleware
			// For now, it's a placeholder
			next()
		}
	}
}

// GetTracer returns the underlying tracer
func (tp *TracingProvider) GetTracer() trace.Tracer {
	return tp.tracer
}
