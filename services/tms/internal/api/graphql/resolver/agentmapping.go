package resolver

import (
	"context"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func agentRunColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AgentRunSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func agentProposalColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AgentProposalSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func agentMemoryColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AgentMemorySpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func agentMemoryConnectionToModel(
	result *pagination.CursorListResult[*agent.Memory],
) (*gqlmodel.AgentMemoryConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *agent.Memory, cursor string) *gqlmodel.AgentMemoryEdge {
			return &gqlmodel.AgentMemoryEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentMemoryEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentMemoryConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

// agentMemorySubject reads the optional subject pair from an input,
// refusing half of one.
func agentMemorySubject(input gqlmodel.AgentMemoryInput) (agent.MemorySubjectType, pulid.ID, error) {
	hasType := input.SubjectType != nil && *input.SubjectType != ""
	hasID := input.SubjectID != nil && strings.TrimSpace(*input.SubjectID) != ""
	switch {
	case !hasType && !hasID:
		return "", pulid.Nil, nil
	case hasType != hasID:
		return "", pulid.Nil, errortypes.NewValidationError(
			"subjectId",
			errortypes.ErrInvalid,
			"A subject type and id go together; give both or neither",
		)
	}

	subjectID, err := pulid.MustParse(*input.SubjectID)
	if err != nil {
		return "", pulid.Nil, err
	}

	return *input.SubjectType, subjectID, nil
}

func optionalInt64(value *int) *int64 {
	if value == nil {
		return nil
	}
	converted := int64(*value)

	return &converted
}

func agentPlanColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AgentPlanSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func agentPlanConnectionToModel(
	result *pagination.CursorListResult[*agent.AgentPlan],
) (*gqlmodel.AgentPlanConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *agent.AgentPlan, cursor string) *gqlmodel.AgentPlanEdge {
			return &gqlmodel.AgentPlanEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentPlanEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentPlanConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func agentExceptionColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AgentExceptionSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func agentRunConnectionToModel(
	result *pagination.CursorListResult[*agent.AgentRun],
) (*gqlmodel.AgentRunConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *agent.AgentRun, cursor string) *gqlmodel.AgentRunEdge {
			return &gqlmodel.AgentRunEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentRunEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentRunConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func agentProposalConnectionToModel(
	result *pagination.CursorListResult[*agent.AgentProposal],
) (*gqlmodel.AgentProposalConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *agent.AgentProposal, cursor string) *gqlmodel.AgentProposalEdge {
			return &gqlmodel.AgentProposalEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentProposalEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentProposalConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func agentExceptionConnectionToModel(
	result *pagination.CursorListResult[*agent.AgentException],
) (*gqlmodel.AgentExceptionConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *agent.AgentException, cursor string) *gqlmodel.AgentExceptionEdge {
			return &gqlmodel.AgentExceptionEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentExceptionEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentExceptionConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
