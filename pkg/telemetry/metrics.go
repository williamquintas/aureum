// Package telemetry provides OpenTelemetry metrics, tracing, and HTTP/gRPC middleware.
package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.Meter("github.com/aureum/pkg/telemetry")

	// RequestsTotal counts the total number of HTTP/gRPC requests.
	RequestsTotal metric.Int64Counter
	// RequestDuration measures request latency in milliseconds.
	RequestDuration metric.Float64Histogram
	// CacheHits tracks cache hit/miss counts.
	CacheHits metric.Int64Counter
)

func init() {
	var err error

	RequestsTotal, err = meter.Int64Counter("requests_total",
		metric.WithDescription("Total number of requests"),
		metric.WithUnit("{count}"),
	)
	if err != nil {
		panic(err)
	}

	RequestDuration, err = meter.Float64Histogram("request_duration_ms",
		metric.WithDescription("Request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		panic(err)
	}

	CacheHits, err = meter.Int64Counter("cache_hits_total",
		metric.WithDescription("Total number of cache hits"),
		metric.WithUnit("{count}"),
	)
	if err != nil {
		panic(err)
	}
}

// RecordRequest records a request metric with operation, status, and duration.
func RecordRequest(ctx context.Context, operation string, status string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.String("status", status),
	}
	RequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	RequestDuration.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(
		attribute.String("operation", operation),
	))
}

// RecordCacheHit records a cache hit or miss metric.
func RecordCacheHit(ctx context.Context, cacheName string, hit bool) {
	attrs := []attribute.KeyValue{
		attribute.String("cache_name", cacheName),
		attribute.Bool("hit", hit),
	}
	CacheHits.Add(ctx, 1, metric.WithAttributes(attrs...))
}
