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
	return definitionColumns(ctx, projection.AgentDefinitionSpec, nodePathPrefix)
}

// definitionColumns is the columns a selection of an agent needs. Its
// opening questions are worked out from its template and its tools, so
// asking for them reads both.
func definitionColumns(
	ctx context.Context,
	spec projection.TypeSpec,
	nodePathPrefix string,
) []string {
	selection := projection.Select(
		spec,
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

// agentDefinitionDelegates reads the agents a definition may hand work to
// through the request's definition loader, so a page of agents costs one
// query for all their delegates. One deleted since it was added is left out.
func agentDefinitionDelegates(
	ctx context.Context,
	obj *agentdefinition.Definition,
) ([]*agentdefinition.Definition, error) {
	if len(obj.DelegateIDs) == 0 {
		return []*agentdefinition.Definition{}, nil
	}

	loadersForRequest, ok := loaders.FromContext(ctx)
	if !ok || loadersForRequest == nil {
		return nil, errortypes.NewDatabaseError("Agent definition loader is not configured")
	}

	thunks := make([]func() (*agentdefinition.Definition, error), 0, len(obj.DelegateIDs))
	for _, id := range obj.DelegateIDs {
		thunks = append(thunks, loadersForRequest.AgentDefinitionByID.LoadThunk(ctx, id.String()))
	}

	delegates := make([]*agentdefinition.Definition, 0, len(thunks))
	for _, thunk := range thunks {
		delegate, err := thunk()
		if err != nil {
			if errortypes.IsNotFoundError(err) {
				continue
			}

			return nil, err
		}
		if delegate != nil {
			delegates = append(delegates, delegate)
		}
	}

	return delegates, nil
}
