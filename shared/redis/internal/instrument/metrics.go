package instrument

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/shared/redis/internal/pool"
	"github.com/emoss08/trenova/shared/redis/internal/resp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type PoolSource struct {
	Stats   func() pool.Stats
	Size    int
	MinIdle int
}

type poolInstruments struct {
	idleMax       metric.Int64ObservableUpDownCounter
	idleMin       metric.Int64ObservableUpDownCounter
	connsMax      metric.Int64ObservableUpDownCounter
	usage         metric.Int64ObservableUpDownCounter
	waits         metric.Int64ObservableUpDownCounter
	waitsDuration metric.Int64ObservableUpDownCounter
	timeouts      metric.Int64ObservableUpDownCounter
	hits          metric.Int64ObservableUpDownCounter
	misses        metric.Int64ObservableUpDownCounter
}

func (i *Instrument) initMetrics(meter metric.Meter, addr string) error {
	var err error
	i.meter = meter
	i.attrs = []attribute.KeyValue{
		attribute.String("db.system", "redis"),
		attribute.String("pool.name", addr),
	}
	i.createTime, err = meter.Float64Histogram(
		"db.client.connections.create_time",
		metric.WithDescription("The time it took to create a new connection."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}
	i.useTime, err = meter.Float64Histogram(
		"db.client.connections.use_time",
		metric.WithDescription(
			"The time between borrowing a connection and returning it to the pool.",
		),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}
	i.measuring = true

	return nil
}

func (i *Instrument) ObservePool(src PoolSource) error {
	if !i.measuring {
		return nil
	}
	ins, err := newPoolInstruments(i.meter)
	if err != nil {
		return err
	}

	poolAttrs := attribute.NewSet(i.attrs...)
	idleAttrs := attribute.NewSet(append(poolAttrs.ToSlice(), attribute.String("state", "idle"))...)
	usedAttrs := attribute.NewSet(append(poolAttrs.ToSlice(), attribute.String("state", "used"))...)
	onPool := metric.WithAttributeSet(poolAttrs)

	i.registration, err = i.meter.RegisterCallback(
		func(_ context.Context, o metric.Observer) error {
			stats := src.Stats()
			o.ObserveInt64(ins.idleMax, 0, onPool)
			o.ObserveInt64(ins.idleMin, int64(src.MinIdle), onPool)
			o.ObserveInt64(ins.connsMax, int64(src.Size), onPool)
			o.ObserveInt64(ins.usage, stats.IdleConns, metric.WithAttributeSet(idleAttrs))
			o.ObserveInt64(
				ins.usage,
				stats.TotalConns-stats.IdleConns,
				metric.WithAttributeSet(usedAttrs),
			)
			o.ObserveInt64(ins.waits, stats.Waits, onPool)
			o.ObserveInt64(ins.waitsDuration, stats.WaitDurationNs, onPool)
			o.ObserveInt64(ins.timeouts, stats.Timeouts, onPool)
			o.ObserveInt64(ins.hits, stats.Hits, onPool)
			o.ObserveInt64(ins.misses, stats.Misses, onPool)
			return nil
		},
		ins.idleMax,
		ins.idleMin,
		ins.connsMax,
		ins.usage,
		ins.waits,
		ins.waitsDuration,
		ins.timeouts,
		ins.hits,
		ins.misses,
	)

	return err
}

func (i *Instrument) Close() error {
	if i.registration == nil {
		return nil
	}
	return i.registration.Unregister()
}

func newPoolInstruments(meter metric.Meter) (*poolInstruments, error) {
	var (
		ins  poolInstruments
		errs []error
	)
	counter := func(name, description string, unit ...string) metric.Int64ObservableUpDownCounter {
		opts := []metric.Int64ObservableUpDownCounterOption{metric.WithDescription(description)}
		if len(unit) > 0 {
			opts = append(opts, metric.WithUnit(unit[0]))
		}
		c, err := meter.Int64ObservableUpDownCounter(name, opts...)
		errs = append(errs, err)
		return c
	}

	ins.idleMax = counter(
		"db.client.connections.idle.max",
		"The maximum number of idle open connections allowed",
	)
	ins.idleMin = counter(
		"db.client.connections.idle.min",
		"The minimum number of idle open connections allowed",
	)
	ins.connsMax = counter(
		"db.client.connections.max",
		"The maximum number of open connections allowed",
	)
	ins.usage = counter(
		"db.client.connections.usage",
		"The number of connections that are currently in state described by the state attribute",
	)
	ins.waits = counter(
		"db.client.connections.waits",
		"The number of times a connection was waited for",
	)
	ins.waitsDuration = counter(
		"db.client.connections.waits_duration",
		"The total time spent for waiting a connection in nanoseconds",
		"ns",
	)
	ins.timeouts = counter(
		"db.client.connections.timeouts",
		"The number of connection timeouts that have occurred trying to obtain a connection from the pool",
	)
	ins.hits = counter(
		"db.client.connections.hits",
		"The number of times free connection was found in the pool",
	)
	ins.misses = counter(
		"db.client.connections.misses",
		"The number of times free connection was not found in the pool",
	)

	return &ins, errors.Join(errs...)
}

func (i *Instrument) recordCreate(ctx context.Context, d time.Duration, err error) {
	attrs := make([]attribute.KeyValue, 0, len(i.attrs)+2)
	attrs = append(attrs, i.attrs...)
	attrs = append(attrs, i.statusAttr(err), errorTypeAttr(err))
	i.createTime.Record(ctx, milliseconds(d), metric.WithAttributeSet(attribute.NewSet(attrs...)))
}

func (i *Instrument) recordUse(ctx context.Context, kind string, d time.Duration, err error) {
	attrs := make([]attribute.KeyValue, 0, len(i.attrs)+3)
	attrs = append(attrs, i.attrs...)
	attrs = append(attrs, attribute.String("type", kind), i.statusAttr(err), errorTypeAttr(err))
	i.useTime.Record(ctx, milliseconds(d), metric.WithAttributeSet(attribute.NewSet(attrs...)))
}

func milliseconds(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

func (i *Instrument) statusAttr(err error) attribute.KeyValue {
	switch {
	case err == nil:
		return attribute.String("status", "ok")
	case errors.Is(err, resp.Nil):
		return attribute.String("status", "nil")
	default:
		return attribute.String("status", "error")
	}
}

func errorTypeAttr(err error) attribute.KeyValue {
	switch {
	case err == nil:
		return attribute.String("error_type", "none")
	case errors.Is(err, context.Canceled):
		return attribute.String("error_type", "context_canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return attribute.String("error_type", "context_timeout")
	default:
		return attribute.String("error_type", "other")
	}
}
