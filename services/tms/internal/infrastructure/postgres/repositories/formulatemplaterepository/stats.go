package formulatemplaterepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type statsRow struct {
	ID            pulid.ID `bun:"id"`
	UsageCount    int      `bun:"usage_count"`
	ScenarioCount int      `bun:"scenario_count"`
}

// statsQuery counts, per template, everything that rates with it and every
// scenario that guards it. Each consumer is its own tenant-scoped subselect so
// the page costs one round trip however many rows it holds.
func statsQuery(
	q *bun.SelectQuery,
	req *repositories.GetFormulaTemplateStatsRequest,
) *bun.SelectQuery {
	ft := buncolgen.FormulaTemplateColumns

	consumer := func(table string) string {
		return "(SELECT COUNT(*) FROM " + table + " c " +
			"WHERE c.formula_template_id = ft.id " +
			"AND c.organization_id = ft.organization_id " +
			"AND c.business_unit_id = ft.business_unit_id)"
	}

	return q.
		TableExpr("formula_templates AS ft").
		ColumnExpr("ft.id AS id").
		ColumnExpr(
			consumer("shipments")+" + "+
				consumer("accessorial_charges")+" + "+
				consumer("rate_agreement_rules")+" + "+
				consumer("rate_agreement_accessorials")+" AS usage_count",
		).
		ColumnExpr(
			"(SELECT COUNT(*) FROM formula_template_test_cases tc "+
				"WHERE tc.template_id = ft.id "+
				"AND tc.organization_id = ft.organization_id "+
				"AND tc.business_unit_id = ft.business_unit_id) AS scenario_count",
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FormulaTemplateScopeTenant(sq, req.TenantInfo).
				Where(ft.ID.In(), bun.In(req.TemplateIDs))
		})
}

func (r *repository) CountStatsByIDs(
	ctx context.Context,
	req *repositories.GetFormulaTemplateStatsRequest,
) (map[pulid.ID]repositories.TemplateStats, error) {
	stats := make(map[pulid.ID]repositories.TemplateStats, len(req.TemplateIDs))
	if len(req.TemplateIDs) == 0 {
		return stats, nil
	}

	log := r.l.With(zap.String("operation", "CountStatsByIDs"))

	var rows []statsRow
	if err := statsQuery(r.db.DBForContext(ctx).NewSelect(), req).Scan(ctx, &rows); err != nil {
		log.Error("failed to count formula template stats", zap.Error(err))
		return nil, err
	}

	for _, row := range rows {
		stats[row.ID] = repositories.TemplateStats{
			UsageCount:    row.UsageCount,
			ScenarioCount: row.ScenarioCount,
		}
	}

	return stats, nil
}
