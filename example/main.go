package main

import (
	"context"
	"github.com/subnova/slog-exporter/slogtrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
	"time"
)

func initLogProvider() (func(context.Context) error, error) {
	traceExporter, err := slogtrace.New()
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(traceExporter))
	otel.SetTracerProvider(tracerProvider)

	return tracerProvider.Shutdown, nil
}

func main() {
	shutdown, err := initLogProvider()
	if err != nil {
		panic(err)
	}
	defer func() {
		err = shutdown(context.Background())
		if err != nil {
			slog.Error("shutting down exporter", slog.String("error", err.Error()))
		}
	}()

	tracer := otel.Tracer("example")

	ctx, span := tracer.Start(context.Background(), "example", trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("example.key", "example.value")))
	span.AddEvent("example event", trace.WithAttributes(attribute.Int("example.event.key", 1)))
	defer span.End()

	time.Sleep(2 * time.Second)

	_, span2 := tracer.Start(ctx, "another example", trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attribute.StringSlice("errors", []string{"error1", "error2"})))
	span2.SetStatus(codes.Error, "example error")
	defer span2.End()
}
