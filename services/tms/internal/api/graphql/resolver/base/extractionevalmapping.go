package base

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
)

func ProjectedSelection(
	ctx context.Context,
	spec projection.TypeSpec,
	nodePathPrefix string,
) projection.Selection {
	return projection.Select(
		spec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)
}
