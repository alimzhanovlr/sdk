package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds tracing configuration
type Config struct {
	Enabled     bool
	ServiceName string
	Endpoint    string
	SampleRate  float64
}

// Tracer wraps OpenTelemetry tracer
type Tracer struct {
	provider *tracesdk.TracerProvider
	tracer   trace.Tracer
	enabled  bool
}

// New creates a new tracer
func New(cfg Config) (*Tracer, error) {
	if !cfg.Enabled {
		return &Tracer{enabled: false}, nil
	}

	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.Endpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create jaeger exporter: %w", err)
	}

	// Create trace provider
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
		)),
		tracesdk.WithSampler(tracesdk.TraceIDRatioBased(cfg.SampleRate)),
	)

	otel.SetTracerProvider(tp)

	tracer := tp.Tracer(cfg.ServiceName)

	return &Tracer{
		provider: tp,
		tracer:   tracer,
		enabled:  true,
	}, nil
}

// Start starts a new span
func (t *Tracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !t.enabled {
		return ctx, trace.SpanFromContext(ctx)
	}
	return t.tracer.Start(ctx, name, opts...)
}

// StartSpanFromContext starts a span from context
func (t *Tracer) StartSpanFromContext(ctx context.Context, operation string) (context.Context, trace.Span) {
	return t.Start(ctx, operation)
}

// AddEvent adds an event to the current span
func (t *Tracer) AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	if !t.enabled {
		return
	}
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetAttributes sets attributes on the current span
func (t *Tracer) SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	if !t.enabled {
		return
	}
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span
func (t *Tracer) RecordError(ctx context.Context, err error) {
	if !t.enabled {
		return
	}
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// Shutdown shuts down the tracer provider
func (t *Tracer) Shutdown(ctx context.Context) error {
	if !t.enabled || t.provider == nil {
		return nil
	}
	return t.provider.Shutdown(ctx)
}

// GetTraceID returns trace ID from context
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}
