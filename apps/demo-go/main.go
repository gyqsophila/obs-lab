
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/github.com/go-chi/chi/otelchi"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

var (
	requests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_server_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	latency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_server_request_duration_seconds",
			Help:    "HTTP request latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(requests, latency)
	zerolog.TimeFieldFormat = time.RFC3339Nano
}

func tracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		return nil, err
	}
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("demo-go"),
			semconv.DeploymentEnvironment("dev"),
		),
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

func main() {
	ctx := context.Background()
	tp, err := tracerProvider(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init tracer")
	}
	defer func() { _ = tp.Shutdown(ctx) }()

	r := chi.NewRouter()
	r.Use(otelchi.Middleware("demo-go"))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sleepMs, _ := strconv.Atoi(r.URL.Query().Get("sleep_ms"))
		if sleepMs > 0 {
			time.Sleep(time.Duration(sleepMs) * time.Millisecond)
		}
		status := http.StatusOK
		if rand.Float64() < 0.03 {
			status = http.StatusInternalServerError
		}
		elapsed := time.Since(start).Seconds()
		latency.WithLabelValues("GET", "/hello").Observe(elapsed)
		requests.WithLabelValues("GET", "/hello", fmt.Sprintf("%d", status)).Inc()

		span := otel.Tracer("demo-go").Start(r.Context(), "work").Span
		defer span.End()

		// Structured log with trace_id
		traceID := ""
		if sc := span.SpanContext(); sc.IsValid() {
			traceID = sc.TraceID().String()
		}
		log.Info().Str("trace_id", traceID).Str("path", "/hello").Float64("latency_s", elapsed).Msg("hello served")
		w.WriteHeader(status)
		_, _ = w.Write([]byte("hello"))
	})

	r.Get("/boom", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		latency.WithLabelValues("GET", "/boom").Observe(time.Since(start).Seconds())
		requests.WithLabelValues("GET", "/boom", "500").Inc()

		span := otel.Tracer("demo-go").Start(r.Context(), "boom").Span
		defer span.End()
		traceID := ""
		if sc := span.SpanContext(); sc.IsValid() {
			traceID = sc.TraceID().String()
		}
		log.Error().Str("trace_id", traceID).Str("path", "/boom").Msg("boom error")
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	// metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/", r)

	addr := ":8080"
	log.Info().Msgf("starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}
