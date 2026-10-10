package carrierrepository

import (
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
)

func applyCarrierColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.CarrierTable.All())
	}

	return q.Column(columns...)
}
