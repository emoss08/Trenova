package querybudget

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	maxLoggedRepeats   = 5
	maxLoggedSQLLength = 2000
)

type Params struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

type Budgets struct {
	maxRepeats int
	logger     *observability.ContextLogger
}

func New(p Params) *Budgets {
	return &Budgets{
		maxRepeats: p.Config.Database.Budget.GetMaxRepeats(),
		logger:     observability.NewContextLogger(p.Logger.Named("database.budget")),
	}
}

func (b *Budgets) Start(ctx context.Context, fields ...zap.Field) context.Context {
	if b == nil {
		return ctx
	}

	return keel.Budget(ctx,
		keel.MaxRepeats(b.maxRepeats),
		keel.OnExceeded(func(ctx context.Context, report keel.BudgetReport) {
			b.exceeded(ctx, report, fields)
		}),
	)
}

func (b *Budgets) exceeded(ctx context.Context, report keel.BudgetReport, fields []zap.Field) {
	out := make([]zap.Field, 0, len(fields)+4)
	out = append(out, fields...)
	out = append(out,
		zap.Int("max_repeats", b.maxRepeats),
		zap.Int("round_trips", report.RoundTrips),
		zap.Int("statements", report.Statements),
		zap.Array("repeats", repeats(overBudget(report, b.maxRepeats))),
	)

	b.logger.Warn(
		ctx,
		"A request sent one statement in more round trips than its budget allows, the shape of an N+1 read",
		out...,
	)
}

func overBudget(report keel.BudgetReport, maxRepeats int) []keel.Repeat {
	out := make([]keel.Repeat, 0, min(len(report.Repeats), maxLoggedRepeats))
	for _, repeat := range report.Repeats {
		if repeat.Count <= maxRepeats {
			continue
		}
		out = append(out, repeat)
		if len(out) == maxLoggedRepeats {
			break
		}
	}

	return out
}

type repeats []keel.Repeat

func (r repeats) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for _, repeat := range r {
		if err := enc.AppendObject(loggedRepeat(repeat)); err != nil {
			return err
		}
	}

	return nil
}

type loggedRepeat keel.Repeat

func (r loggedRepeat) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("sql", truncate(r.SQL, maxLoggedSQLLength))
	enc.AddInt("count", r.Count)

	return enc.AddArray("call_sites", zapcore.ArrayMarshalerFunc(
		func(arr zapcore.ArrayEncoder) error {
			for _, site := range r.CallSites {
				arr.AppendString(site)
			}
			return nil
		},
	))
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}

	return s[:limit] + "…"
}
