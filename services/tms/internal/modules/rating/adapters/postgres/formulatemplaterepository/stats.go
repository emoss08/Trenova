package formulatemplaterepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/formulatemplatecol"
	"github.com/emoss08/trenova/pkg/buncolgen/rateagreementcol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type statsRow struct {
	ID         pulid.ID `bun:"id"`
	UsageCount int      `bun:"usage_count"`
}

type consumer struct {
	table    buncolgen.TableInfo
	template buncolgen.Column
	org      buncolgen.Column
	bu       buncolgen.Column
}

var templateConsumers = []consumer{
	{
		table:    shipmentcol.ShipmentTable,
		template: shipmentcol.ShipmentColumns.FormulaTemplateID,
		org:      shipmentcol.ShipmentColumns.OrganizationID,
		bu:       shipmentcol.ShipmentColumns.BusinessUnitID,
	},
	{
		table:    rateagreementcol.RateAgreementRuleTable,
		template: rateagreementcol.RateAgreementRuleColumns.FormulaTemplateID,
		org:      rateagreementcol.RateAgreementRuleColumns.OrganizationID,
		bu:       rateagreementcol.RateAgreementRuleColumns.BusinessUnitID,
	},
}

func statsQuery(
	q *bun.SelectQuery,
	req *formulatemplateports.GetFormulaTemplateStatsRequest,
) *bun.SelectQuery {
	ft := formulatemplatecol.FormulaTemplateColumns

	q = q.
		TableExpr(
			"? AS ?",
			bun.Ident(formulatemplatecol.FormulaTemplateTable.Name),
			bun.Ident(formulatemplatecol.FormulaTemplateTable.Alias),
		).
		ColumnExpr(ft.ID.As("id"))

	counts := make([]any, 0, len(templateConsumers))
	var expr strings.Builder
	for i := range templateConsumers {
		c := &templateConsumers[i]
		if i > 0 {
			expr.WriteString(" + ")
		}
		expr.WriteString("(?)")
		counts = append(counts, q.NewSelect().
			TableExpr("? AS ?", bun.Ident(c.table.Name), bun.Ident(c.table.Alias)).
			ColumnExpr("COUNT(*)").
			Where(c.template.EqColumn(ft.ID)).
			Where(c.org.EqColumn(ft.OrganizationID)).
			Where(c.bu.EqColumn(ft.BusinessUnitID)))
	}

	return q.
		ColumnExpr(expr.String()+" AS usage_count", counts...).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return formulatemplatecol.FormulaTemplateScopeTenant(sq, req.TenantInfo).
				Where(ft.ID.In(), bun.List(req.TemplateIDs))
		})
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

			var rows []statsRow
			if err := statsQuery(
				r.db.DBForContext(ctx).NewSelect(),
				req,
			).Scan(ctx, &rows); err != nil {
				r.l.Error(
					"failed to count formula template stats",
					zap.String("operation", "CountStatsByIDs"),
					zap.Error(err),
				)
				return nil, err
			}

			for _, row := range rows {
				stats[row.ID] = formulatemplateports.TemplateStats{UsageCount: row.UsageCount}
			}

			return stats, nil
		},
	)
}
