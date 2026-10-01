package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/pkg/pagination"
)

func journalReversalConnectionToModel(
	result *pagination.CursorListResult[*journalreversal.Reversal],
) (*gqlmodel.JournalReversalConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *journalreversal.Reversal, cursor string) *gqlmodel.JournalReversalEdge {
			return &gqlmodel.JournalReversalEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.JournalReversalEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.JournalReversalConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
