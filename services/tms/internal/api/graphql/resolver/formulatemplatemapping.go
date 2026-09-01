package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func formulaTemplateColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.FormulaTemplateSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func formulaTemplateConnectionToModel(
	result *pagination.CursorListResult[*formulatemplate.FormulaTemplate],
) (*gqlmodel.FormulaTemplateConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *formulatemplate.FormulaTemplate, cursor string) *gqlmodel.FormulaTemplateEdge {
			return &gqlmodel.FormulaTemplateEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.FormulaTemplateEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FormulaTemplateConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func formulaTemplateVariableDefinitionsToModel(
	definitions []*formulatypes.VariableDefinition,
) []*gqlmodel.FormulaTemplateVariableDefinition {
	items := make([]*gqlmodel.FormulaTemplateVariableDefinition, 0, len(definitions))
	for _, definition := range definitions {
		if definition == nil {
			continue
		}
		items = append(items, &gqlmodel.FormulaTemplateVariableDefinition{
			Name:         definition.Name,
			Type:         string(definition.Type),
			Description:  definition.Description,
			Required:     definition.Required,
			DefaultValue: definition.DefaultValue,
			Source:       sliceutils.StringPtrValue(definition.Source),
		})
	}
	return items
}

func formulaTemplateBreakdownDefinitionsToModel(
	definitions []*formulatypes.BreakdownDefinition,
) []*gqlmodel.FormulaTemplateBreakdownDefinition {
	items := make([]*gqlmodel.FormulaTemplateBreakdownDefinition, 0, len(definitions))
	for _, definition := range definitions {
		if definition == nil {
			continue
		}
		items = append(items, &gqlmodel.FormulaTemplateBreakdownDefinition{
			Name:       definition.Name,
			Label:      definition.Label,
			Expression: definition.Expression,
		})
	}
	return items
}

// formulaTemplateStats reads the per-row counts through the request's
// dataloader, so a page of templates resolves them in one query.
func formulaTemplateStats(
	ctx context.Context,
	obj *formulatemplate.FormulaTemplate,
) (repositories.TemplateStats, error) {
	loadersForRequest, ok := loaders.FromContext(ctx)
	if !ok || loadersForRequest == nil {
		return repositories.TemplateStats{}, errortypes.NewDatabaseError(
			"Formula template stats loader is not configured",
		)
	}

	return loadersForRequest.FormulaTemplateStatsByID.Load(ctx, obj.ID.String())()
}
