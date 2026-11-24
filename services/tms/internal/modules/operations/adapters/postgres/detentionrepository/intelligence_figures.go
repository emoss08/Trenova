package detentionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"go.uber.org/zap"
)

func (r *intelligenceRepository) FacilityFigures(
	ctx context.Context,
	req *detentionports.IntelligenceRequest,
) ([]detentionports.FacilityIntelligence, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]detentionports.FacilityIntelligence, error) {
			current := span{IntelligenceWindow: req.Current, locations: req.LocationIDs}
			var rows []facilityRow
			if err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.IntoSlice(r.facilities(current, req.Limit), &rows),
			); err != nil {
				r.l.Error("failed to read detention facility figures", zap.Error(err))
				return nil, err
			}

			return facilityIntelligence(rows), nil
		},
	)
}

func (r *intelligenceRepository) TotalFigures(
	ctx context.Context,
	req *detentionports.IntelligenceRequest,
) (*detentionports.IntelligenceSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*detentionports.IntelligenceSummary, error) {
			current := span{IntelligenceWindow: req.Current, locations: req.LocationIDs}
			figures := figuresTable(req.LocationIDs)
			out := &detentionports.IntelligenceSummary{}
			var recovery recoveryRow
			if err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.IntoValue(r.totals(figures, current), &out.Totals),
				keel.IntoValue(r.recovery(figures, current), &recovery),
			); err != nil {
				r.l.Error("failed to read detention totals", zap.Error(err))
				return nil, err
			}
			out.Recovery = recovery.buckets(&invoicedRow{})

			return out, nil
		},
	)
}

func (r *intelligenceRepository) ArrivalFigures(
	ctx context.Context,
	req *detentionports.IntelligenceRequest,
) ([]detentionports.ArrivalCell, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]detentionports.ArrivalCell, error) {
			current := span{IntelligenceWindow: req.Current, locations: req.LocationIDs}
			var days []arrivalRow
			var cells []detentionports.ArrivalCell
			if err := keel.Batch(ctx, r.db.Keel(ctx),
				r.arrivals(current, req.Timezone, &days, &cells),
			); err != nil {
				r.l.Error("failed to read detention arrivals", zap.Error(err))
				return nil, err
			}
			if len(req.LocationIDs) == 0 {
				return arrivalCells(days), nil
			}

			return cells, nil
		},
	)
}
