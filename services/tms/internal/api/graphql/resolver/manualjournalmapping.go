package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/pkg/pagination"
)

func manualJournalColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.ManualJournalSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func manualJournalConnectionToModel(
	result *pagination.CursorListResult[*manualjournal.Request],
) (*gqlmodel.ManualJournalConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *manualjournal.Request, cursor string) *gqlmodel.ManualJournalEdge {
			return &gqlmodel.ManualJournalEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.ManualJournalEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ManualJournalConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
