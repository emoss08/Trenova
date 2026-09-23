package resolver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func agentDefinitionColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AgentDefinitionSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)
	if !selection.HasSpecial("starters") {
		return selection.Columns
	}

	columns := sliceutils.AppendIfMissing(
		selection.Columns,
		buncolgen.DefinitionColumns.Template.Bare(),
	)

	return sliceutils.AppendIfMissing(columns, buncolgen.DefinitionColumns.ToolNames.Bare())
}

func agentDefinitionConnectionToModel(
	result *pagination.CursorListResult[*agentdefinition.Definition],
) (*gqlmodel.AgentDefinitionConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *agentdefinition.Definition, cursor string) *gqlmodel.AgentDefinitionEdge {
			return &gqlmodel.AgentDefinitionEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentDefinitionEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentDefinitionConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func agentDefinitionStats(
	ctx context.Context,
	obj *agentdefinition.Definition,
) (repositories.AgentDefinitionStats, error) {
	loadersForRequest, ok := loaders.FromContext(ctx)
	if !ok || loadersForRequest == nil {
		return repositories.AgentDefinitionStats{}, errortypes.NewDatabaseError(
			"Agent definition stats loader is not configured",
		)
	}

	return loadersForRequest.AgentDefinitionStatsByID.Load(ctx, obj.ID.String())
}
