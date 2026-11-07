package auditlogresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

var detailFields = []string{
	"edges.node.changes",
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
