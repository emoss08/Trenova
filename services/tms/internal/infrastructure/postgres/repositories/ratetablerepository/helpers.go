package ratetablerepository

import (
	"github.com/emoss08/trenova/internal/core/domain/ratetable"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

func orderEntries(sq *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.RateTableEntryColumns
	return sq.Order(cols.SortOrder.OrderAsc()).
		Order(cols.RangeMin.OrderAsc()).
		Order(cols.MatchKey.OrderAsc())
}

func applyRateTableColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.RateTableTable.All())
	}

	return q.Column(columns...)
}

func stampEntries(entity *ratetable.RateTable, resetIDs bool) {
	for _, entry := range entity.Entries {
		if entry == nil {
			continue
		}

		if resetIDs {
			entry.ID = pulid.Nil
		}

		entry.RateTableID = entity.ID
		entry.OrganizationID = entity.OrganizationID
		entry.BusinessUnitID = entity.BusinessUnitID
	}
}
