package postgres

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const keelInstrumentation = "github.com/emoss08/keel"

type tracingObserver struct {
	tracer    trace.Tracer
	histogram metric.Int64Histogram
	attrs     []attribute.KeyValue
}

var _ keel.Observer = (*tracingObserver)(nil)

func newTracingObserver(database string) *tracingObserver {
	histogram, _ := otel.Meter(keelInstrumentation).Int64Histogram(
		"go.sql.query_timing",
		metric.WithDescription("Timing of processed queries"),
		metric.WithUnit("milliseconds"),
	)

	return &tracingObserver{
		tracer:    otel.GetTracerProvider().Tracer(keelInstrumentation),
		histogram: histogram,
		attrs: []attribute.KeyValue{
			semconv.DBSystemPostgreSQL,
			semconv.DBNameKey.String(database),
		},
	}
}

func (o *tracingObserver) BeforeQuery(ctx context.Context, e *keel.QueryEvent) context.Context {
	ctx, _ = o.tracer.Start( //nolint:spancheck // AfterQuery ends the span it reads back from this context
		ctx,
		spanName(e),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	return ctx
}

func (o *tracingObserver) AfterQuery(ctx context.Context, e *keel.QueryEvent) {
	attrs := make([]attribute.KeyValue, 0, len(o.attrs)+2)
	attrs = append(attrs, o.attrs...)
	attrs = append(attrs, semconv.DBOperationKey.String(e.Op))
	if e.Table != "" {
		attrs = append(attrs, semconv.DBSQLTableKey.String(e.Table))
	}

	o.histogram.Record(ctx, e.Duration.Milliseconds(), metric.WithAttributes(attrs...))

	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	defer span.End()

	attrs = append(attrs,
		semconv.DBStatementKey.String(e.SQL),
		attribute.Int64("db.rows_affected", e.Rows),
	)
	if e.Replica != "" {
		attrs = append(attrs, attribute.String("db.replica", e.Replica))
	}
	span.SetAttributes(attrs...)

	if e.Err != nil && !errors.Is(e.Err, keel.ErrNotFound) {
		span.RecordError(e.Err)
		span.SetStatus(codes.Error, e.Err.Error())
	}
}

func spanName(e *keel.QueryEvent) string {
	if e.Table == "" {
		return e.Op
	}
	return e.Op + " " + e.Table
}

type debugObserver struct {
	logger  *observability.ContextLogger
	verbose bool
}

var _ keel.Observer = (*debugObserver)(nil)

func (o *debugObserver) BeforeQuery(ctx context.Context, _ *keel.QueryEvent) context.Context {
	return ctx
}

func (o *debugObserver) AfterQuery(ctx context.Context, e *keel.QueryEvent) {
	if e.Err == nil && !o.verbose {
		return
	}

	fields := []zap.Field{
		zap.String("operation", e.Op),
		zap.String("table", e.Table),
		zap.Duration("duration", e.Duration),
		zap.Int64("rows", e.Rows),
		zap.String("query", e.SQL),
	}
	if e.Err != nil {
		fields = append(fields, zap.Error(e.Err))
	}

	o.logger.Debug(ctx, "keel query", fields...)
}
