package formulalibraryresolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
)

func formulaLibraryColumns(ctx context.Context, nodePathPrefix string) []string {
	return projection.Select(
		projection.FormulaLibrarySpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	).Columns
}

func formulaLibraryConnectionToModel(
	result *pagination.CursorListResult[*formulalibrary.FormulaLibrary],
) (*gqlmodel.FormulaLibraryConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *formulalibrary.FormulaLibrary, cursor string) *gqlmodel.FormulaLibraryEdge {
			return &gqlmodel.FormulaLibraryEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.FormulaLibraryEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FormulaLibraryConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func changeSummaryToModel(summary map[string]jsonutils.FieldChange) map[string]any {
	if len(summary) == 0 {
		return nil
	}
	out := make(map[string]any, len(summary))
	for path, change := range summary {
		out[path] = change
	}
	return out
}
