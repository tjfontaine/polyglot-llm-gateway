package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// InitTracer initializes OpenTelemetry tracing
func InitTracer(serviceName string, logger *slog.Logger) (func(context.Context) error, error) {
	// Create resource with service name
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"",
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	var tp *sdktrace.TracerProvider

	if os.Getenv("OTEL_TRACES_STDOUT") == "1" || os.Getenv("OTEL_TRACES_STDOUT") == "true" {
		// Create stdout exporter for development
		exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
		logger.Info("OpenTelemetry initialized", slog.String("service", serviceName), slog.String("exporter", "stdout"))
	} else {
		// No-op exporter; keep resource for future exporters
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
		)
		logger.Info("OpenTelemetry initialized", slog.String("service", serviceName), slog.String("exporter", "none"))
	}

	otel.SetTracerProvider(tp)

	// Return shutdown function
	return tp.Shutdown, nil
}
