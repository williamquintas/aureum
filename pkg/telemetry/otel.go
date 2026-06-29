package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// InitOTEL initializes OpenTelemetry tracing and metrics exporters with the given service identity.
func InitOTEL(serviceName, version string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("service.version", version),
		),
	)
	if err != nil {
		return fmt.Errorf("create resource: %w", err)
	}

	traceExporter, err := otlptrace.New(ctx, otlptracehttp.NewClient())
	if err != nil {
		return fmt.Errorf("create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	metricExporter, err := prometheus.New()
	if err != nil {
		return fmt.Errorf("create metric exporter: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metricExporter),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	return nil
}

// ShutdownOTEL gracefully shuts down the tracer and meter providers.
func ShutdownOTEL(ctx context.Context) error {
	tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	if ok {
		if err := tp.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown tracer provider: %w", err)
		}
	}

	mp, ok := otel.GetMeterProvider().(*metric.MeterProvider)
	if ok {
		if err := mp.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown meter provider: %w", err)
		}
	}

	return nil
}
