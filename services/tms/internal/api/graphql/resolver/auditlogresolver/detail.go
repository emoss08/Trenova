package auditlogresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/pkg/pagination"
)

var detailFields = []string{
	"edges.node.previousState",
	"edges.node.currentState",
	"edges.node.metadata",
}

func detailRequested(ctx context.Context) bool {
	for _, path := range detailFields {
		if graphql.FieldRequested(ctx, path) {
			return true
		}
	}
	return false
}

func changesRequested(ctx context.Context) bool {
	return graphql.FieldRequested(ctx, "edges.node.changes")
}

func requestCountRequested(ctx context.Context) bool {
	return graphql.FieldRequested(ctx, "edges.node.requestEntryCount")
}

func countFirstPageOnly(cursor pagination.CursorInfo) pagination.CursorInfo {
	if cursor.After != "" {
		cursor.IncludeTotalCount = false
	}
	return cursor
}
