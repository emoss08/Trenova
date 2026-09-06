package dbhelper

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type idCountRow struct {
	ID    pulid.ID `bun:"id"`
	Total int      `bun:"total"`
}

func CountByID(
	ctx context.Context,
	q *bun.SelectQuery,
	groupCol buncolgen.Column,
	capacity int,
) (map[pulid.ID]int, error) {
	rows := make([]idCountRow, 0, capacity)
	if err := q.
		ColumnExpr(groupCol.As("id")).
		ColumnExpr(buncolgen.Count("total")).
		Group(groupCol.Qualified()).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}

	counts := make(map[pulid.ID]int, len(rows))
	for _, row := range rows {
		counts[row.ID] = row.Total
	}

	return counts, nil
}
