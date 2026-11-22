package instrument

import (
	"context"
	"errors"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/redis/internal/resp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const scopeName = "github.com/redis/go-redis/extra/redisotel"

type Params struct {
	Addr           string
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

type Instrument struct {
	tracer       trace.Tracer
	spanOpts     []trace.SpanStartOption
	attrs        []attribute.KeyValue
	meter        metric.Meter
	registration metric.Registration
	createTime   metric.Float64Histogram
	useTime      metric.Float64Histogram
	tracing      bool
	measuring    bool
}

func New(p Params) (*Instrument, error) {
	inst := &Instrument{}
	if p.TracerProvider != nil {
		inst.tracing = true
		inst.tracer = p.TracerProvider.Tracer(scopeName)
		inst.spanOpts = []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(tracingAttrs(p.Addr)...),
		}
	}
	if p.MeterProvider != nil {
		if err := inst.initMetrics(p.MeterProvider.Meter(scopeName), p.Addr); err != nil {
			return nil, err
		}
	}

	return inst, nil
}

func tracingAttrs(addr string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("db.system", "redis"),
		attribute.String("db.connection_string", "redis://"+addr),
	}
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return attrs
	}
	attrs = append(attrs, attribute.String("server.address", host))
	if port, convErr := strconv.Atoi(portText); convErr == nil {
		attrs = append(attrs, attribute.Int("server.port", port))
	}

	return attrs
}

func (i *Instrument) Dial(
	ctx context.Context,
	dial func(context.Context) (net.Conn, error),
) (net.Conn, error) {
	start := time.Now()
	var span trace.Span
	if i.tracing {
		ctx, span = i.tracer.Start(ctx, "redis.dial", i.spanOpts...)
	}

	nc, err := dial(ctx)

	if span != nil {
		i.recordError(span, err)
		span.End()
	}
	if i.measuring {
		i.recordCreate(ctx, time.Since(start), err)
	}

	return nc, err
}

func (i *Instrument) Command(
	ctx context.Context,
	args []any,
) (spanCtx context.Context, end func(error)) {
	var attrs []attribute.KeyValue
	if i.tracing {
		attrs = i.callerAttrs(make([]attribute.KeyValue, 0, 4))
		attrs = append(attrs, attribute.String("db.statement", statement(args)))
	}
	return i.start(ctx, commandName(args), "command", attrs)
}

func (i *Instrument) Pipeline(
	ctx context.Context,
	cmds [][]any,
) (spanCtx context.Context, end func(error)) {
	var (
		name  string
		attrs []attribute.KeyValue
	)
	if i.tracing {
		summary, stmt := pipelineStrings(cmds)
		name = "redis.pipeline " + summary
		attrs = make([]attribute.KeyValue, 0, 5)
		attrs = append(attrs, attribute.Int("db.redis.num_cmd", len(cmds)))
		attrs = i.callerAttrs(attrs)
		attrs = append(attrs, attribute.String("db.statement", stmt))
	}
	return i.start(ctx, name, "pipeline", attrs)
}

func (i *Instrument) start(
	ctx context.Context,
	name, kind string,
	attrs []attribute.KeyValue,
) (spanCtx context.Context, end func(error)) {
	begun := time.Now()
	var span trace.Span
	if i.tracing {
		opts := make([]trace.SpanStartOption, 0, len(i.spanOpts)+1)
		opts = append(opts, i.spanOpts...)
		opts = append(opts, trace.WithAttributes(attrs...))
		ctx, span = i.tracer.Start(ctx, name, opts...)
	}

	return ctx, func(err error) {
		if span != nil {
			i.recordError(span, err)
			span.End()
		}
		if i.measuring {
			i.recordUse(ctx, kind, time.Since(begun), err)
		}
	}
}

func (i *Instrument) recordError(span trace.Span, err error) {
	if err == nil || errors.Is(err, resp.Nil) {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func pipelineStrings(cmds [][]any) (summary, stmt string) {
	const nameLimit = 10

	names := make([]string, 0, nameLimit)
	b := make([]byte, 0, 32*len(cmds))
	for idx, cmd := range cmds {
		if idx > 0 {
			b = append(b, '\n')
		}
		b = appendCmd(b, cmd)
		if name := commandName(cmd); len(names) < nameLimit && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}

	return strings.Join(names, " "), string(b)
}

func commandName(args []any) string {
	if len(args) == 0 {
		return ""
	}
	name, _ := args[0].(string)
	return strings.ToLower(name)
}

func statement(args []any) string {
	return string(appendCmd(make([]byte, 0, 32), args))
}
