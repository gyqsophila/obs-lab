
module demo-go

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.12
	go.opentelemetry.io/contrib/instrumentation/github.com/go-chi/chi/otelchi v0.53.0
	go.opentelemetry.io/otel v1.26.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.26.0
	go.opentelemetry.io/otel/sdk v1.26.0
	go.opentelemetry.io/otel/trace v1.26.0
	github.com/prometheus/client_golang v1.17.0
	github.com/rs/zerolog v1.33.0
)
