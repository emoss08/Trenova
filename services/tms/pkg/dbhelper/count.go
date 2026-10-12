package dbhelper

import (
	"context"

	"github.com/uptrace/bun"
)

func Count(ctx context.Context, query *bun.SelectQuery) (int, error) {
	count, err := query.Count(ctx)
	return int(count), err
}
