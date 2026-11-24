package driversettlementboardreader

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/settlementboard"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/settlementreleaseports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
)

type SettlementBatch struct{ release settlementreleaseports.Reader }

func NewSettlementBatch(p Params) *SettlementBatch {
	return &SettlementBatch{release: p.Release}
}

func (*SettlementBatch) Metric() string { return settlementboard.MetricSettlementBatch }

func (r *SettlementBatch) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	batch, err := r.release.Batch(ctx, &settlementreleaseports.BatchRequest{})
	if err != nil {
		return nil, err
	}

	return batchRead(ctx, batch, req.Location, req.Now), nil
}

func batchFigures() map[string]*boardcatalog.Value {
	figures := map[string]*boardcatalog.Value{}
	for idx := range settlementboard.Metrics {
		metric := &settlementboard.Metrics[idx]
		if metric.Name != settlementboard.MetricSettlementBatch {
			continue
		}
		for _, figure := range metric.Figures {
			figures[figure.Name] = nil
		}
	}

	return figures
}

func valueOf(value boardcatalog.Value) *boardcatalog.Value { return &value }

func dayText(ctx context.Context, at int64, loc *time.Location, now int64) string {
	return i18n.Day(
		i18n.FromContext(ctx),
		settlementreleaseports.Day(at, loc),
		settlementreleaseports.Day(now, loc),
	)
}

func batchRead(
	ctx context.Context,
	batch *settlementreleaseports.Batch,
	loc *time.Location,
	now int64,
) *boardcatalog.MetricRead {
	figures := batchFigures()
	if batch == nil {
		return &boardcatalog.MetricRead{
			Title:   i18n.T(ctx, "No pay batch yet"),
			Source:  i18n.T(ctx, "no driver pay batches"),
			Figures: figures,
		}
	}
	blocked := batch.BlockedTotal()
	held := batch.Blocked[driversettlement.ReleaseBlockHeld]
	critical := batch.Blocked[driversettlement.ReleaseBlockCriticalException]
	disputed := batch.Blocked[driversettlement.ReleaseBlockDisputeOpen]
	read := &boardcatalog.MetricRead{
		Title: batch.Name,
		Source: i18n.T(ctx,
			"{0, plural, one {# settlement} other {# settlements}}, paid {1}",
			batch.Settlements, dayText(ctx, batch.PayDate, loc, now)),
		Figures: figures,
	}
	figures[settlementboard.FigureBatchSettlements] = valueOf(boardcatalog.Count(batch.Settlements))
	figures[settlementboard.FigureBatchPayDate] = &boardcatalog.Value{
		Kind: boardcatalog.ValueText,
		Text: dayText(ctx, batch.PayDate, loc, now),
	}
	figures[settlementboard.FigureReadySettlements] = valueOf(boardcatalog.Count(batch.Ready.Count))
	figures[settlementboard.FigureBlockedSettlements] = valueOf(boardcatalog.Count(blocked.Count))
	figures[settlementboard.FigureHeldSettlements] = valueOf(boardcatalog.Count(held.Count))
	figures[settlementboard.FigureCriticalSettlements] = valueOf(boardcatalog.Count(critical.Count))
	figures[settlementboard.FigureDisputeSettlements] = valueOf(boardcatalog.Count(disputed.Count))
	if batch.Currencies > 1 || batch.Settlements == 0 {
		return read
	}

	money := func(minor int64) *boardcatalog.Value {
		return valueOf(boardcatalog.Money(minor, batch.Currency))
	}
	figures[settlementboard.FigureBatchNet] = money(batch.NetMinor)
	figures[settlementboard.FigureBatchGross] = money(batch.GrossMinor)
	figures[settlementboard.FigureBatchReimbursements] = money(batch.ReimbursementsMinor)
	figures[settlementboard.FigureBatchDeducted] = money(batch.DeductedMinor())
	figures[settlementboard.FigureReadyNet] = money(batch.Ready.NetMinor)
	figures[settlementboard.FigureBlockedNet] = money(blocked.NetMinor)
	read.Split = batchSplit(ctx, batch)

	return read
}

func batchSplit(ctx context.Context, batch *settlementreleaseports.Batch) *boardcatalog.Split {
	money := func(minor int64) boardcatalog.Value {
		return boardcatalog.Money(minor, batch.Currency)
	}
	parts := make([]boardcatalog.Part, 0, 6)
	add := func(label string, minor int64, count int64, kind boardcatalog.PartKind) {
		if count == 0 && minor == 0 {
			return
		}
		parts = append(parts, boardcatalog.Part{Label: label, Value: money(minor), Kind: kind})
	}
	add(i18n.T(ctx, "Posted or paid"), batch.Released.NetMinor, batch.Released.Count,
		boardcatalog.PartUsed)
	add(i18n.T(ctx, "Ready to release"), batch.Ready.NetMinor, batch.Ready.Count,
		boardcatalog.PartUsed)
	for _, block := range driversettlement.ReleaseBlocks() {
		tally := batch.Blocked[block]
		add(
			settlementreleaseports.BlockWords(ctx, block),
			tally.NetMinor,
			tally.Count,
			boardcatalog.PartOver,
		)
	}
	add(i18n.T(ctx, "Deducted"), batch.DeductedMinor(), 0, boardcatalog.PartCredit)

	return &boardcatalog.Split{
		Total: money(batch.NetMinor + batch.DeductedMinor()),
		Parts: parts,
	}
}
