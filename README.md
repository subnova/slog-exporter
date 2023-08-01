# Go slog exporter

This is a tiny library that provides an OpenTelemetry exporter that emits traces via Go's slog package. This is
intended to be used for development purposes when you want to see traces in your terminal.

Traces that are batched together will be sorted by their start time. However, traces that are emitted individually
will be printed in the order they are received. Traces emitted in different batches will be interleaved.

When using the default slog handler, timestamps are added by go's standard logger instead of using the timestamps
in the trace. Therefore, the timestamps will be for when the trace was emitted, not when the span started.