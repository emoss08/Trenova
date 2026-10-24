package listquery

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/pagination"
)

func Offset[T any](
	ctx context.Context,
	db *keel.DB,
	q *keel.Query[T],
	info pagination.Info,
) (*pagination.ListResult[*T], error) {
	counted := q.Clone()
	q.Limit(info.SafeLimit()).Offset(info.SafeOffset())

	return listAndCount(ctx, db, q, counted)
}
