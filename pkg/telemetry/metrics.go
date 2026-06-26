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

	RequestsTotal   metric.Int64Counter
	RequestDuration metric.Float64Histogram
	CacheHits       metric.Int64Counter
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

func RecordCacheHit(ctx context.Context, cacheName string, hit bool) {
	attrs := []attribute.KeyValue{
		attribute.String("cache_name", cacheName),
		attribute.Bool("hit", hit),
	}
	CacheHits.Add(ctx, 1, metric.WithAttributes(attrs...))
}
