package auditlogresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
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
