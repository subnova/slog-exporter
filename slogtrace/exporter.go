package slogtrace

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
	"sync"
)

type Exporter struct {
	stoppedMu sync.RWMutex
	stopped   bool
}

func New() (*Exporter, error) {
	return &Exporter{}, nil
}

func (e *Exporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	e.stoppedMu.RLock()
	stopped := e.stopped
	e.stoppedMu.RUnlock()
	if stopped {
		return nil
	}

	if len(spans) == 0 {
		return nil
	}

	var records []slog.Record

	for _, span := range spans {
		var level = slog.LevelInfo
		if span.Status().Code == codes.Error {
			level = slog.LevelError
		}
		record := slog.NewRecord(span.StartTime(), level, span.Name(), 0)

		duration := span.EndTime().Sub(span.StartTime())

		var attrs []slog.Attr
		attrs = append(attrs, slog.String("duration", duration.String()))
		attrs = append(attrs, attributesToAttrs(span.Attributes())...)

		record.AddAttrs(attrs...)

		records = append(records, record)

		for _, event := range span.Events() {
			eventRecord := slog.NewRecord(event.Time, level, event.Name, 0)
			eventRecord.AddAttrs(attributesToAttrs(event.Attributes)...)

			records = append(records, eventRecord)
		}
	}

	slices.SortStableFunc(records, func(a, b slog.Record) int {
		if a.Time == b.Time {
			return 0
		}
		if a.Time.Before(b.Time) {
			return -1
		}
		return 1
	})

	for _, record := range records {
		err := slog.Default().Handler().Handle(ctx, record)
		if err != nil {
			return err
		}
	}

	return nil
}

func attributesToAttrs(attributes []attribute.KeyValue) []slog.Attr {
	var attrs []slog.Attr

	for _, attr := range attributes {
		key := string(attr.Key)

		switch attr.Value.Type() {
		case attribute.BOOL:
			attrs = append(attrs, slog.Bool(key, attr.Value.AsBool()))
		case attribute.INT64:
			attrs = append(attrs, slog.Int64(key, attr.Value.AsInt64()))
		case attribute.FLOAT64:
			attrs = append(attrs, slog.Float64(key, attr.Value.AsFloat64()))
		case attribute.STRING:
			attrs = append(attrs, slog.String(key, attr.Value.AsString()))
		case attribute.BOOLSLICE:
			attrs = append(attrs, slog.String(key, fmt.Sprintf("%+v", attr.Value.AsBoolSlice())))
		case attribute.INT64SLICE:
			attrs = append(attrs, slog.String(key, fmt.Sprintf("%+v", attr.Value.AsInt64Slice())))
		case attribute.FLOAT64SLICE:
			attrs = append(attrs, slog.String(key, fmt.Sprintf("%+v", attr.Value.AsFloat64Slice())))
		case attribute.STRINGSLICE:
			attrs = append(attrs, slog.String(key, fmt.Sprintf("%+v", attr.Value.AsStringSlice())))
		}
	}

	return attrs
}

func (e *Exporter) Shutdown(ctx context.Context) error {
	e.stoppedMu.Lock()
	e.stopped = true
	e.stoppedMu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return nil
}
