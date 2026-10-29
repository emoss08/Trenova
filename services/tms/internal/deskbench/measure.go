package deskbench

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
)

func (b *Bench) priceOf(ctx context.Context, session *Session) benchrecord.CallPricing {
	return func(call *benchrecord.ModelCall) benchrecord.CallPrice {
		tokens := benchrecord.CallTokens(call)

		return b.pricer.Price(ctx, session.Tenant, &tokens)
	}
}

func (b *Bench) Measure(
	ctx context.Context,
	session *Session,
	record *benchrecord.TurnRecord,
	from time.Time,
) {
	ctx = session.Context(ctx)
	priceOf := b.priceOf(ctx, session)
	ownerID := record.TurnID
	if record.RunID.IsNotNil() {
		ownerID = record.RunID
	}

	record.ModelCalls = b.recorder.For(record.ThreadID, ownerID, from, record.FinishedAt)
	benchrecord.SortCalls(record.ModelCalls)
	record.Calls = benchrecord.SummarizeCalls(record.ModelCalls, priceOf)
	record.Usage, record.Models = benchrecord.UsageOf(record.ModelCalls, priceOf)
	record.Unattributed = benchrecord.SummarizeCalls(
		b.recorder.Unattributed(from, record.FinishedAt),
		priceOf,
	)
	record.Disclosures = b.disclosures.For(record.ThreadID, ownerID, from, record.FinishedAt)
	record.FirstTokenMs = firstTokenOf(record)
}

func firstTokenOf(record *benchrecord.TurnRecord) *int64 {
	var first time.Time
	for _, call := range record.ModelCalls {
		if call.FirstTokenAt.IsZero() {
			continue
		}
		if first.IsZero() || call.FirstTokenAt.Before(first) {
			first = call.FirstTokenAt
		}
	}
	if first.IsZero() {
		return nil
	}
	elapsed := first.Sub(record.StartedAt).Milliseconds()

	return &elapsed
}
