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

	return loadDelegates(ctx, loadersForRequest.AgentDefinitionByID, obj, nil)
}

// myAgentDelegates reads the agents a definition may hand work to that the
// person asking may use themselves, through the request's usable-agent
// loader, so a page of agents costs one query and one read of what the
// person may use for all their delegates. A delegate the agent could not be
// handed a task by now is left out as well.
func myAgentDelegates(
	ctx context.Context,
	obj *agentdefinition.Definition,
) ([]*agentdefinition.Definition, error) {
	if len(obj.DelegateIDs) == 0 {
		return []*agentdefinition.Definition{}, nil
	}

	loadersForRequest, ok := loaders.FromContext(ctx)
	if !ok || loadersForRequest == nil {
		return nil, errortypes.NewDatabaseError("Agent loader is not configured")
	}

	return loadDelegates(
		ctx,
		loadersForRequest.UsableAgentByID,
		obj,
		func(delegate *agentdefinition.Definition) bool {
			return obj.DelegateRefusal(delegate) == ""
		},
	)
}

type definitionLoader interface {
	LoadThunk(ctx context.Context, key string) func() (*agentdefinition.Definition, error)
}

// loadDelegates reads obj's delegates through loader in the order they are
// configured, queueing every id before waiting on any so they are read in
// one batch. An id the loader does not find is left out, and so is one keep
// refuses when keep is set.
func loadDelegates(
	ctx context.Context,
	loader definitionLoader,
	obj *agentdefinition.Definition,
	keep func(*agentdefinition.Definition) bool,
) ([]*agentdefinition.Definition, error) {
	thunks := make([]func() (*agentdefinition.Definition, error), 0, len(obj.DelegateIDs))
	for _, id := range obj.DelegateIDs {
		thunks = append(thunks, loader.LoadThunk(ctx, id.String()))
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
		if delegate == nil || (keep != nil && !keep(delegate)) {
			continue
		}
		delegates = append(delegates, delegate)
	}

	return delegates, nil
}
