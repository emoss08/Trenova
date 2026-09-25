package aitrace

import (
	"context"
	"sync"

	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type completionTallyKey struct{}

type callOriginKey struct{}

type CompletionTally struct {
	mu               sync.Mutex
	attempts         int
	failover         bool
	midReplyRestarts int
	providerID       string
	providerName     string
	model            string
	costUSD          decimal.Decimal
	priced           bool
}

type AttemptTally struct {
	ProviderID   string
	ProviderName string
	Model        string
	Failover     bool
	CostUSD      *decimal.Decimal
}

func WithCompletionTally(ctx context.Context) (context.Context, *CompletionTally) {
	tally := new(CompletionTally)

	return context.WithValue(ctx, completionTallyKey{}, tally), tally
}

func TallyFrom(ctx context.Context) *CompletionTally {
	if ctx == nil {
		return nil
	}

	tally, _ := ctx.Value(completionTallyKey{}).(*CompletionTally)

	return tally
}

func (t *CompletionTally) Attempt(a AttemptTally) {
	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.attempts++
	t.failover = t.failover || a.Failover
	t.providerID = a.ProviderID
	t.providerName = a.ProviderName
	if a.Model != "" {
		t.model = a.Model
	}
	if a.CostUSD != nil {
		t.costUSD = t.costUSD.Add(*a.CostUSD)
		t.priced = true
	}
}

func (t *CompletionTally) Restarted() {
	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.midReplyRestarts++
}

func (t *CompletionTally) Attempts() int {
	if t == nil {
		return 0
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return t.attempts
}

func (t *CompletionTally) Record(span trace.Span) {
	if t == nil || !span.IsRecording() {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	attrs := make([]attribute.KeyValue, 0, 7)
	attrs = append(attrs,
		AIAttempts.Int(t.attempts),
		AIFailover.Bool(t.failover),
		AIMidReplyRestarts.Int(t.midReplyRestarts),
	)
	attrs = appendString(attrs, AIProviderID, t.providerID)
	attrs = appendString(attrs, GenAIProviderName, t.providerName)
	attrs = appendString(attrs, GenAIResponseModel, t.model)
	if t.priced {
		attrs = append(attrs, AICostUSD.Float64(t.costUSD.InexactFloat64()))
	}

	span.SetAttributes(attrs...)
}

type CallOrigin struct {
	ActivityAttempt int
	Stream          bool
}

func WithCallOrigin(ctx context.Context, origin CallOrigin) context.Context {
	return context.WithValue(ctx, callOriginKey{}, origin)
}

func CallOriginFrom(ctx context.Context) CallOrigin {
	if ctx == nil {
		return CallOrigin{}
	}

	origin, _ := ctx.Value(callOriginKey{}).(CallOrigin)

	return origin
}
