package carrierrepository

import (
	"github.com/emoss08/trenova/pkg/buncolgen/carriercol"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
)

func applyCarrierColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(carriercol.CarrierTable.All())
	}

	return q.Column(columns...)
}
