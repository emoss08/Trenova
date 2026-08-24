package ratequoterepository

import (
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
)

func applyRateQuoteColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.RateQuoteTable.All())
	}

	return q.Column(columns...)
}
