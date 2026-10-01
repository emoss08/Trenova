package base

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
)

func AgentEvaluationColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AgentEvaluationSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func OptionalInt64(value *int) *int64 {
	if value == nil {
		return nil
	}
	converted := int64(*value)

	return &converted
}
