package postgres

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	slowWaitLogInterval = 10 * time.Second
	slowWaitStackFrames = 12
)

type slowWaitLog struct {
	logger    *observability.ContextLogger
	metrics   *metrics.Database
	component string
	pool      string
	mu        sync.Mutex
	last      map[keel.WaitKind]time.Time
	skipped   map[keel.WaitKind]int
}

func (c *Connection) slowWaits(pool scopePool) keel.SlowWaitConfig {
	after := c.cfg.Database.SlowWaitWarning
	if after <= 0 {
		return keel.SlowWaitConfig{}
	}

	l := &slowWaitLog{
		logger:    c.logger.Named("slow-wait"),
		component: c.settings.component,
		pool:      string(pool),
		last:      map[keel.WaitKind]time.Time{},
		skipped:   map[keel.WaitKind]int{},
	}
	if c.metrics != nil {
		l.metrics = c.metrics.Database
	}

	return keel.SlowWaitConfig{After: after, Report: l.report}
}

func (l *slowWaitLog) report(ctx context.Context, w *keel.SlowWait) {
	if l.metrics != nil {
		l.metrics.RecordSlowWait(l.component, l.pool, w.Kind.String())
	}

	suppressed, ok := l.admit(w.Kind, time.Now())
	if !ok {
		return
	}

	fields := []zap.Field{
		zap.String("kind", w.Kind.String()),
		zap.String("pool", l.pool),
		zap.String("tenant", w.Tenant),
		zap.Duration("waited", w.Waited),
		zap.Strings("stack", trimStack(w.Stack)),
		zap.Int("suppressed_since_last", suppressed),
	}
	switch w.Kind {
	case keel.WaitTenantSlot:
		fields = append(fields,
			zap.Int("limit", w.Limit),
			zap.Int("waiters", w.Waiters),
			zap.Array("holders", slotHolders(w.Holders)),
		)
	case keel.WaitConnection:
		fields = append(fields,
			zap.Int32("pool_acquired", w.Pool.Acquired),
			zap.Int32("pool_idle", w.Pool.Idle),
			zap.Int32("pool_total", w.Pool.Total),
			zap.Int32("pool_max", w.Pool.Max),
		)
	case keel.WaitCacheFill:
	}

	l.logger.Warn(ctx, slowWaitMessage(w.Kind), fields...)
}

func (l *slowWaitLog) admit(kind keel.WaitKind, now time.Time) (int, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if last, ok := l.last[kind]; ok && now.Sub(last) < slowWaitLogInterval {
		l.skipped[kind]++
		return 0, false
	}

	suppressed := l.skipped[kind]
	l.last[kind], l.skipped[kind] = now, 0

	return suppressed, true
}

func slowWaitMessage(kind keel.WaitKind) string {
	switch kind {
	case keel.WaitTenantSlot:
		return "Database work is waiting for its organization's concurrency slot"
	case keel.WaitConnection:
		return "Database work is waiting for a pool connection"
	case keel.WaitCacheFill:
		return "Database read is waiting for another request's read of the same cache entry"
	}

	return "Database work is waiting"
}

func trimStack(stack []string) []string {
	return stack[:min(len(stack), slowWaitStackFrames)]
}

type slotHolders []keel.SlotHolder

func (h slotHolders) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for _, holder := range h {
		if err := enc.AppendObject(slotHolder(holder)); err != nil {
			return err
		}
	}

	return nil
}

type slotHolder keel.SlotHolder

func (h slotHolder) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddDuration("held", h.Held)

	return enc.AddArray("stack", zapcore.ArrayMarshalerFunc(func(arr zapcore.ArrayEncoder) error {
		for _, frame := range trimStack(h.Stack) {
			arr.AppendString(frame)
		}
		return nil
	}))
}
