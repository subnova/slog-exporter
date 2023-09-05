package slogtrace_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"github.com/subnova/slog-exporter/slogtrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slog"
	"testing"
	"time"
)

func initLogProvider(filter attribute.Filter) (func(context.Context) error, error) {
	traceExporter, err := slogtrace.New(filter)
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(traceExporter, sdktrace.WithBatchTimeout(100*time.Millisecond), sdktrace.WithExportTimeout(100*time.Millisecond)))
	otel.SetTracerProvider(tracerProvider)

	return tracerProvider.Shutdown, nil
}

func parseLogLines(buf *bytes.Buffer) ([]map[string]any, error) {
	var data []map[string]any

	jsonlines := bytes.Split(buf.Bytes(), []byte("\n"))
	jsonlines = jsonlines[:len(jsonlines)-1] // remove last empty line

	for _, line := range jsonlines {
		var row map[string]any
		err := json.Unmarshal(line, &row)
		if err != nil {
			return nil, err
		}
		data = append(data, row)
	}

	return data, nil
}

func TestLogLevels(t *testing.T) {
	// setup slog to output JSON data
	buf := bytes.Buffer{}
	w := bufio.NewWriter(&buf)
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))

	// initialize tracer
	shutdown, err := initLogProvider(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = shutdown(context.Background())
	}()

	// emit a trace
	tracer := otel.Tracer("test")

	_, span := tracer.Start(context.Background(), "test", trace.WithSpanKind(trace.SpanKindInternal))
	span.End()

	_, span = tracer.Start(context.Background(), "test error", trace.WithSpanKind(trace.SpanKindInternal))
	span.SetStatus(codes.Error, "test error")
	span.End()

	// flush the buffer
	time.Sleep(200 * time.Millisecond)
	_ = w.Flush()

	// check the output
	logLines, err := parseLogLines(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if len(logLines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(logLines))
	}

	if logLines[0]["level"] != "INFO" {
		t.Errorf("expected level to be INFO, got %v", logLines[0]["level"])
	}
	if logLines[1]["level"] != "ERROR" {
		t.Errorf("expected level to be ERROR, got %v", logLines[1]["level"])
	}
}

func TestAttributesAreCorrectlyFormatted(t *testing.T) {
	// setup slog to output JSON data
	buf := bytes.Buffer{}
	w := bufio.NewWriter(&buf)
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))

	// initialize tracer
	shutdown, err := initLogProvider(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = shutdown(context.Background())
	}()

	// emit a trace
	tracer := otel.Tracer("test")
	_, span := tracer.Start(context.Background(), "test", trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.Bool("bool", true),
			attribute.Int("int", 42),
			attribute.Float64("float", 3.14),
			attribute.String("string", "hello world"),
			attribute.BoolSlice("bools", []bool{true, false}),
			attribute.IntSlice("ints", []int{1, 1, 2, 3, 5, 8, 13}),
			attribute.Float64Slice("floats", []float64{3.14, 2.71828}),
			attribute.StringSlice("strings", []string{"hello", "world"})))
	span.End()

	// flush the buffer
	time.Sleep(200 * time.Millisecond)
	_ = w.Flush()

	// check the output
	logLines, err := parseLogLines(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if len(logLines) != 1 {
		t.Fatalf("expected 1 log line, got %d", len(logLines))
	}

	data := logLines[0]

	if data["bool"] != true {
		t.Errorf("expected bool to be true, got %v", data["bool"])
	}
	if data["int"] != 42.0 { // unmarshaller unmarshals all numbers as float64
		t.Errorf("expected int to be 42, got %v", data["int"])
	}
	if data["float"] != 3.14 {
		t.Errorf("expected float to be 3.14, got %v", data["float"])
	}
	if data["string"] != "hello world" {
		t.Errorf("expected string to be hello world, got %v", data["string"])
	}
	if data["bools"] != "[true false]" {
		t.Errorf("expected bools to be [true, false], got %v", data["bools"])
	}
	if data["ints"] != "[1 1 2 3 5 8 13]" {
		t.Errorf("expected ints to be [1 1 2 3 5 8 13], got %v", data["ints"])
	}
	if data["floats"] != "[3.14 2.71828]" {
		t.Errorf("expected floats to be [3.14 2.71828], got %v", data["floats"])
	}
	if data["strings"] != "[hello world]" {
		t.Errorf("expected strings to be [hello world], got %v", data["strings"])
	}
}

func TestEventsAreEmitted(t *testing.T) {
	// setup slog to output JSON data
	buf := bytes.Buffer{}
	w := bufio.NewWriter(&buf)
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))

	// initialize tracer
	shutdown, err := initLogProvider(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = shutdown(context.Background())
	}()

	// emit a trace
	tracer := otel.Tracer("test")
	_, span := tracer.Start(context.Background(), "test", trace.WithSpanKind(trace.SpanKindInternal))
	span.AddEvent("test event", trace.WithAttributes(attribute.String("key", "value")))
	span.End()

	// flush the buffer
	time.Sleep(200 * time.Millisecond)
	_ = w.Flush()

	// check the output
	logLines, err := parseLogLines(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if len(logLines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(logLines))
	}

	if logLines[0]["msg"] != "test" {
		t.Errorf("expected msg to be test, got %v", logLines[0]["msg"])
	}
	if logLines[1]["msg"] != "test event" {
		t.Errorf("expected msg to be test event, got %v", logLines[1]["msg"])
	}
	if logLines[1]["key"] != "value" {
		t.Errorf("expected key to be value, got %v", logLines[1]["key"])
	}
}

func TestMultipleSpanAreOrderedByTime(t *testing.T) {
	// setup slog to output JSON data
	buf := bytes.Buffer{}
	w := bufio.NewWriter(&buf)
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))

	// initialize tracer
	shutdown, err := initLogProvider(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = shutdown(context.Background())
	}()

	// emit some traces
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test", trace.WithSpanKind(trace.SpanKindInternal))

	_, span2 := tracer.Start(ctx, "test2", trace.WithSpanKind(trace.SpanKindInternal))

	span2.End()
	span.End()

	// flush the buffer
	time.Sleep(200 * time.Millisecond)
	_ = w.Flush()

	// check the output
	logLines, err := parseLogLines(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if len(logLines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(logLines))
	}

	if logLines[0]["msg"] != "test" {
		t.Errorf("expected msg to be test for first log, got %v", logLines[0]["msg"])
	}
	if logLines[1]["msg"] != "test2" {
		t.Errorf("expected msg to be test2 for second log, got %v", logLines[1]["msg"])
	}
}

func TestAttributesAreFiltered(t *testing.T) {
	// setup slog to output JSON data
	buf := bytes.Buffer{}
	w := bufio.NewWriter(&buf)
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))

	// initialize tracer
	shutdown, err := initLogProvider(func(kv attribute.KeyValue) bool {
		return kv.Key == "string"
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = shutdown(context.Background())
	}()

	// emit a trace
	tracer := otel.Tracer("test")
	_, span := tracer.Start(context.Background(), "test", trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("string", "hello world"),
			attribute.String("string2", "hello world 2")))
	span.End()

	// flush the buffer
	time.Sleep(200 * time.Millisecond)
	_ = w.Flush()

	// check the output
	logLines, err := parseLogLines(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if len(logLines) != 1 {
		t.Fatalf("expected 1 log line, got %d", len(logLines))
	}

	data := logLines[0]

	if data["string"] != "hello world" {
		t.Errorf("expected string to be hello world, got %v", data["string"])
	}
	if data["string2"] != nil {
		t.Errorf("expected string2 to be filtered, got %v", data["string2"])
	}
}
