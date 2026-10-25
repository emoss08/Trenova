package formulatemplaterepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement/rateagreementdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type statsRow struct {
	ID         pulid.ID
	UsageCount int64
}

func statsQuery(templateIDs []pulid.ID) *keel.Query[formulatemplate.FormulaTemplate] {
	ft := formulatemplatedb.FormulaTemplates
	rr := rateagreementdb.RateAgreementRules
	rules := keel.Scalar(rr.Pick(keel.Count()).Where(rr.FormulaTemplateID.EqCol(ft.ID)))

	return ft.Pick(ft.ID, keel.Raw("? + ?", shipmentUsage(ft), rules)).
		Where(ft.ID.In(templateIDs...)).
		Limit(len(templateIDs))
}

func (r *repository) CountStatsByIDs(
	ctx context.Context,
	req *formulatemplateports.GetFormulaTemplateStatsRequest,
) (map[pulid.ID]formulatemplateports.TemplateStats, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]formulatemplateports.TemplateStats, error) {
			stats := make(map[pulid.ID]formulatemplateports.TemplateStats, len(req.TemplateIDs))
			if len(req.TemplateIDs) == 0 {
				return stats, nil
			}

			rows, err := keel.Into[statsRow](ctx, r.db.Keel(ctx), statsQuery(req.TemplateIDs))
			if err != nil {
				r.l.Error(
					"failed to count formula template stats",
					zap.String("operation", "CountStatsByIDs"),
					zap.Error(err),
				)
				return nil, err
			}

			for _, row := range rows {
				stats[row.ID] = formulatemplateports.TemplateStats{UsageCount: int(row.UsageCount)}
			}

			return stats, nil
		},
	)
}
