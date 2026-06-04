package dbhelper

import "github.com/uptrace/bun"

func RelationColumns(columns []string) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(q *bun.SelectQuery) *bun.SelectQuery {
		if len(columns) == 0 {
			return q
		}

		return q.Column(columns...)
	}
}
