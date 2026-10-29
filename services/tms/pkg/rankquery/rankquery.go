package rankquery

import (
	"slices"

	"github.com/emoss08/keel"
)

const rankColumn = "row_rank"

type Params struct {
	Name    string
	Columns []string
	With    []*keel.CTETable
	Rank    []keel.Order
	Limit   int
}

func Top[T any](grouped *keel.Query[T], p Params) *keel.Query[*keel.Row] {
	ranked := keel.CTE(
		p.Name,
		append(slices.Clone(p.Columns), rankColumn),
		grouped.Pick(keel.RowNumber().Over(nil, p.Rank...)),
	)

	picks := make([]keel.Expr, 0, len(p.Columns))
	for _, column := range p.Columns {
		picks = append(picks, ranked.Col(column))
	}
	rank := keel.Int64Of(ranked.Col(rankColumn))

	return ranked.Query().
		With(append(slices.Clone(p.With), ranked)...).
		Pick(picks...).
		Where(rank.Lte(int64(p.Limit))).
		Order(rank.Asc())
}
