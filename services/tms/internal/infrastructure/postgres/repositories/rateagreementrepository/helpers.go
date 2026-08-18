package rateagreementrepository

import (
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

func applyRateAgreementColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.RateAgreementTable.All())
	}

	return q.Column(columns...)
}

// applyRuleWindow narrows a rule set to the moment asked for. A zero moment
// loads every rule, which is what an editor showing the history of a lane
// needs; a rating only ever wants the live ones.
func applyRuleWindow(q *bun.SelectQuery, asOf int64) *bun.SelectQuery {
	if asOf == 0 {
		return q
	}

	cols := buncolgen.RateAgreementRuleColumns

	return q.
		Where(cols.EffectiveFrom.Lte(), asOf).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(cols.EffectiveTo.IsNull()).
				WhereOr(cols.EffectiveTo.Gt(), asOf)
		})
}

func stampAccessorials(entity *rateagreement.RateAgreement, resetIDs bool) {
	for _, accessorial := range entity.Accessorials {
		if accessorial == nil {
			continue
		}

		if resetIDs {
			accessorial.ID = pulid.Nil
		}

		accessorial.RateAgreementID = entity.ID
		accessorial.OrganizationID = entity.OrganizationID
		accessorial.BusinessUnitID = entity.BusinessUnitID
	}
}
