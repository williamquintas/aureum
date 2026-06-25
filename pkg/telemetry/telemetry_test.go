package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

// setupTestMeterProvider creates a new MeterProvider with a ManualReader,
// replaces the global meter provider, recreates the package-level instruments
// so they use the test provider, and registers a cleanup to restore the original.
func setupTestMeterProvider(t *testing.T) *sdkmetric.ManualReader {
	t.Helper()

	prev := otel.GetMeterProvider()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(mp)

	// Recreate instruments so they use the test meter provider
	meter = otel.Meter("github.com/aureum/pkg/telemetry")

	var err error
	RequestsTotal, err = meter.Int64Counter("requests_total",
		metric.WithDescription("Total number of requests"),
		metric.WithUnit("{count}"),
	)
	require.NoError(t, err)

	RequestDuration, err = meter.Float64Histogram("request_duration_ms",
		metric.WithDescription("Request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	require.NoError(t, err)

	CacheHits, err = meter.Int64Counter("cache_hits_total",
		metric.WithDescription("Total number of cache hits"),
		metric.WithUnit("{count}"),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		otel.SetMeterProvider(prev)
	})

	return reader
}

// setupTestTracerProvider creates a new TracerProvider with an InMemoryExporter,
// sets it as the global tracer provider, and registers a cleanup.
func setupTestTracerProvider(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()

	prev := otel.GetTracerProvider()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)

	// Also set the global propagator to W3C TraceContext for extraction
	prevProp := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})

	t.Cleanup(func() {
		otel.SetTracerProvider(prev)
		otel.SetTextMapPropagator(prevProp)
	})

	return exporter
}

// getAttrVal is a helper to safely extract a string attribute value from a Set.
func getAttrVal(t *testing.T, attrs attribute.Set, key string) string {
	t.Helper()
	v, ok := attrs.Value(attribute.Key(key))
	if !ok {
		return ""
	}
	return v.AsString()
}

// getAttrBool is a helper to safely extract a bool attribute value from a Set.
func getAttrBool(t *testing.T, attrs attribute.Set, key string) bool {
	t.Helper()
	v, ok := attrs.Value(attribute.Key(key))
	if !ok {
		return false
	}
	return v.AsBool()
}

// ── CC-31: RecordRequest ─────────────────────────────────────────────────────

func TestRecordRequest_IncrementsCounterAndRecordsDuration(t *testing.T) {
	ctx := context.Background()
	reader := setupTestMeterProvider(t)

	RecordRequest(ctx, "create", "success", 150*time.Millisecond)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	require.Len(t, rm.ScopeMetrics, 1)
	sm := rm.ScopeMetrics[0]

	var foundCounter, foundHistogram bool
	for _, m := range sm.Metrics {
		switch m.Name {
		case "requests_total":
			foundCounter = true
			data, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok, "requests_total should be Sum[int64]")
			require.Len(t, data.DataPoints, 1)
			assert.Equal(t, int64(1), data.DataPoints[0].Value, "counter should be 1")
			assert.Equal(t, "create", getAttrVal(t, data.DataPoints[0].Attributes, "operation"))
			assert.Equal(t, "success", getAttrVal(t, data.DataPoints[0].Attributes, "status"))

		case "request_duration_ms":
			foundHistogram = true
			data, ok := m.Data.(metricdata.Histogram[float64])
			require.True(t, ok, "request_duration_ms should be Histogram[float64]")
			require.Len(t, data.DataPoints, 1)
			assert.InDelta(t, 150.0, data.DataPoints[0].Sum, 1.0, "duration ~150ms")
			assert.Equal(t, "create", getAttrVal(t, data.DataPoints[0].Attributes, "operation"))
		}
	}
	assert.True(t, foundCounter, "requests_total metric not found")
	assert.True(t, foundHistogram, "request_duration_ms metric not found")
}

func TestRecordRequest_MultipleCallsAccumulate(t *testing.T) {
	ctx := context.Background()
	reader := setupTestMeterProvider(t)

	RecordRequest(ctx, "read", "success", 10*time.Millisecond)
	RecordRequest(ctx, "read", "success", 20*time.Millisecond)
	RecordRequest(ctx, "read", "error", 30*time.Millisecond)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	require.Len(t, rm.ScopeMetrics, 1)
	sm := rm.ScopeMetrics[0]

	for _, m := range sm.Metrics {
		if m.Name == "requests_total" {
			data := m.Data.(metricdata.Sum[int64])
			var total int64
			for _, dp := range data.DataPoints {
				total += dp.Value
			}
			assert.Equal(t, int64(3), total, "total across all attribute sets should be 3")
			break
		}
	}
}

func TestRecordRequest_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		RecordRequest(context.Background(), "op", "ok", time.Second)
	})
	assert.NotPanics(t, func() {
		RecordRequest(context.Background(), "", "", 0)
	})
}

// ── CC-32: RecordCacheHit ────────────────────────────────────────────────────

func TestRecordCacheHit_RecordsMetricWithAttributes(t *testing.T) {
	ctx := context.Background()
	reader := setupTestMeterProvider(t)

	RecordCacheHit(ctx, "credit_cards", true)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	require.Len(t, rm.ScopeMetrics, 1)
	sm := rm.ScopeMetrics[0]

	found := false
	for _, m := range sm.Metrics {
		if m.Name == "cache_hits_total" {
			found = true
			data, ok := m.Data.(metricdata.Sum[int64])
			require.True(t, ok)
			require.Len(t, data.DataPoints, 1)
			assert.Equal(t, int64(1), data.DataPoints[0].Value)
			assert.Equal(t, "credit_cards", getAttrVal(t, data.DataPoints[0].Attributes, "cache_name"))
			assert.Equal(t, true, getAttrBool(t, data.DataPoints[0].Attributes, "hit"))
		}
	}
	assert.True(t, found, "cache_hits_total metric not found")
}

func TestRecordCacheHit_HitAndMiss(t *testing.T) {
	ctx := context.Background()
	reader := setupTestMeterProvider(t)

	RecordCacheHit(ctx, "invoices", true)
	RecordCacheHit(ctx, "invoices", false)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	require.Len(t, rm.ScopeMetrics, 1)
	sm := rm.ScopeMetrics[0]

	for _, m := range sm.Metrics {
		if m.Name == "cache_hits_total" {
			data := m.Data.(metricdata.Sum[int64])
			require.Len(t, data.DataPoints, 2, "should have 2 data points (hit=true, hit=false)")
			for _, dp := range data.DataPoints {
				hit := getAttrBool(t, dp.Attributes, "hit")
				if hit {
					assert.Equal(t, int64(1), dp.Value, "hit count should be 1")
				} else {
					assert.Equal(t, int64(1), dp.Value, "miss count should be 1")
				}
			}
			break
		}
	}
}

func TestRecordCacheHit_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		RecordCacheHit(context.Background(), "test", true)
	})
	assert.NotPanics(t, func() {
		RecordCacheHit(context.Background(), "test", false)
	})
	assert.NotPanics(t, func() {
		RecordCacheHit(context.Background(), "", true)
	})
}

// ── CC-33: Error metrics recording ──────────────────────────────────────────
// RecordRequest with error status verifies error attribute tagging.
func TestRecordRequest_ErrorAttributes(t *testing.T) {
	ctx := context.Background()
	reader := setupTestMeterProvider(t)

	RecordRequest(ctx, "create", "error", 200*time.Millisecond)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	require.Len(t, rm.ScopeMetrics, 1)
	sm := rm.ScopeMetrics[0]

	for _, m := range sm.Metrics {
		if m.Name == "requests_total" {
			data := m.Data.(metricdata.Sum[int64])
			require.Len(t, data.DataPoints, 1)
			status := getAttrVal(t, data.DataPoints[0].Attributes, "status")
			assert.Equal(t, "error", status, "error status attribute")
			break
		}
	}
}

// ── CC-34: HTTP Middleware ───────────────────────────────────────────────────

func TestHTTPMiddleware_CreatesSpan(t *testing.T) {
	exporter := setupTestTracerProvider(t)

	handler := HTTPMiddleware("test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		assert.True(t, span.SpanContext().IsValid(), "span should be valid in handler")
		assert.True(t, span.SpanContext().HasTraceID(), "span should have trace ID")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1, "should create exactly one span")
	assert.Equal(t, "GET /api/v1/test", spans[0].Name)
}

func TestHTTPMiddleware_ContextPropagation(t *testing.T) {
	exporter := setupTestTracerProvider(t)

	handler := HTTPMiddleware("test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		sc := span.SpanContext()
		w.Header().Set("X-Trace-ID", sc.TraceID().String())
		w.WriteHeader(http.StatusOK)
	}))

	// Set up a remote span context to simulate an incoming trace
	traceID := trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	spanID := trace.SpanID{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.TraceFlags(1),
		Remote:     true,
	})

	// Use the W3C TraceContext propagator to inject the span context into HTTP headers
	req := httptest.NewRequest(http.MethodPost, "/api/v1/data", nil)
	ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
	propagator := propagation.TraceContext{}
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	responseTraceID := rec.Header().Get("X-Trace-ID")
	assert.Equal(t, traceID.String(), responseTraceID, "trace context should propagate")

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, traceID, spans[0].SpanContext.TraceID(), "child span should have same trace ID")
}

func TestHTTPMiddleware_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		mw := HTTPMiddleware("test")
		handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	})
}

// ── gRPC Unary Interceptor ──────────────────────────────────────────────────

func TestGRPCUnaryInterceptor_ReturnsServerOption(t *testing.T) {
	opt := GRPCUnaryInterceptor()
	assert.NotNil(t, opt, "should return a ServerOption")
	assert.NotPanics(t, func() {
		_ = GRPCUnaryInterceptor()
	})
}

// ── InitOTEL / ShutdownOTEL ─────────────────────────────────────────────────

func TestInitOTEL_InvalidConfig_ReturnsError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping InitOTEL test in short mode (requires real OTLP endpoint)")
	}
	err := InitOTEL("", "")
	assert.Error(t, err, "should fail with empty service name")
}

func TestShutdownOTEL_NoopProvider_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_ = ShutdownOTEL(context.Background())
	})
}

// ── Package-level instrument metadata ───────────────────────────────────────

func TestPackageInstruments_AreDefined(t *testing.T) {
	assert.NotNil(t, RequestsTotal, "RequestsTotal counter should be defined")
	assert.NotNil(t, RequestDuration, "RequestDuration histogram should be defined")
	assert.NotNil(t, CacheHits, "CacheHits counter should be defined")
}
