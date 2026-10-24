package listquery

import (
	"context"

	"github.com/emoss08/keel"
)

func UpdateReturning[T any](
	ctx context.Context,
	db *keel.DB,
	q *keel.Query[T],
	sets ...keel.Assignment,
) ([]*T, error) {
	read := q.Clone()
	n, err := q.Update(ctx, db, sets...)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return []*T{}, nil
	}

	return read.Limit(int(n)).List(ctx, db)
}
