package agentresolver

import (
	"context"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/agentloaders"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/api/graphql/projection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory/agentmemorydb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentapprovalports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const agentMemoryTaintsSpecial = "taints"

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

func runTranscript(ctx context.Context, run *agent.AgentRun) (*agent.RunTranscript, error) {
	if run.Transcript != nil || !loadercore.Available(ctx) {
		return run.Transcript, nil
	}

	loaded, err := agentloaders.AgentRunTranscriptByID(ctx, run.ID.String())
	switch {
	case errortypes.IsNotFoundError(err):
		return run.Transcript, nil
	case err != nil:
		return nil, err
	default:
		return loaded.Transcript, nil
	}
}

func reflectionSignals(reflection *agentreflection.Reflection) []*agentreflection.ReflectionSignal {
	if reflection == nil {
		return []*agentreflection.ReflectionSignal{}
	}

	out := make([]*agentreflection.ReflectionSignal, 0, len(reflection.Signals))
	for idx := range reflection.Signals {
		out = append(out, &reflection.Signals[idx])
	}

	return out
}

func supersededMemory(
	ctx context.Context,
	memory *agentmemory.Memory,
) (*agentmemory.Memory, error) {
	if !loadercore.Available(ctx) || !memory.Replaces() {
		return nil, nil
	}

	loaded, err := agentloaders.AgentMemoryByID(ctx, memory.SupersedesID.String())
	switch {
	case errortypes.IsNotFoundError(err):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return loaded, nil
	}
}

func replacingMemory(ctx context.Context, memory *agentmemory.Memory) (*agentmemory.Memory, error) {
	if !loadercore.Available(ctx) || memory == nil || memory.ID.IsNil() {
		return nil, nil
	}

	return agentloaders.AgentMemoryReplacement(ctx, memory.ID.String())
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

func agentEvaluationConnectionToModel(
	result *pagination.CursorListResult[*agentevaluation.Evaluation],
) (*gqlmodel.AgentEvaluationConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *agentevaluation.Evaluation, cursor string) *gqlmodel.AgentEvaluationEdge {
			return &gqlmodel.AgentEvaluationEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentEvaluationEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentEvaluationConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func agentMemoryColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := gqlconnection.ProjectedSelection(ctx, projection.AgentMemorySpec, nodePathPrefix)
	if !selection.HasSpecial(agentMemoryTaintsSpecial) || len(selection.Columns) == 0 {
		return selection.Columns
	}

	m := agentmemorydb.Memories
	columns := selection.Columns
	for _, column := range projection.ColumnNames(m.Tainted.Col, m.ReviewedByUserID.Col, m.ReviewedAt.Col) {
		columns = sliceutils.AppendIfMissing(columns, column)
	}

	return columns
}

func agentMemoryConnectionToModel(
	result *pagination.CursorListResult[*agentmemory.Memory],
) (*gqlmodel.AgentMemoryConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *agentmemory.Memory, cursor string) *gqlmodel.AgentMemoryEdge {
			return &gqlmodel.AgentMemoryEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentMemoryEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentMemoryConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func agentReflectionColumns(ctx context.Context, nodePathPrefix string) []string {
	selection := projection.Select(
		projection.AgentReflectionSpec,
		func(path string) bool {
			return graphql.FieldRequested(ctx, path)
		},
		projection.SelectOptions{PathPrefix: nodePathPrefix},
	)

	return selection.Columns
}

func agentReflectionConnectionToModel(
	result *pagination.CursorListResult[*agentreflection.Reflection],
) (*gqlmodel.AgentReflectionConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *agentreflection.Reflection, cursor string) *gqlmodel.AgentReflectionEdge {
			return &gqlmodel.AgentReflectionEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentReflectionEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentReflectionConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

// agentMemorySubject reads the optional subject pair from an input,
// refusing half of one.
func agentMemorySubject(
	input gqlmodel.AgentMemoryInput,
) (agentmemory.MemorySubjectType, pulid.ID, error) {
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

func recordedEgressClass(proposal *agent.AgentProposal) *agent.EgressClass {
	if proposal == nil || !proposal.EgressClass.IsValid() {
		return nil
	}
	class := proposal.EgressClass

	return &class
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
	result *pagination.CursorListResult[*agentplan.AgentPlan],
) (*gqlmodel.AgentPlanConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *agentplan.AgentPlan, cursor string) *gqlmodel.AgentPlanEdge {
			return &gqlmodel.AgentPlanEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentPlanEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentPlanConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
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
	page, err := gqlconnection.EntityCursorConnection(
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
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func agentProposalConnectionToModel(
	result *pagination.CursorListResult[*agent.AgentProposal],
) (*gqlmodel.AgentProposalConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
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
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func agentExceptionConnectionToModel(
	result *pagination.CursorListResult[*agentexception.AgentException],
) (*gqlmodel.AgentExceptionConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *agentexception.AgentException, cursor string) *gqlmodel.AgentExceptionEdge {
			return &gqlmodel.AgentExceptionEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.AgentExceptionEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.AgentExceptionConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

// settleApprovalRequest reads which approval an undo or a commit-now names:
// a proposal or a plan, exactly one.
func settleApprovalRequest(
	proposalID, planID *string,
	tenant pagination.TenantInfo,
) (*agentapprovalports.SettleApprovalRequest, error) {
	req := &agentapprovalports.SettleApprovalRequest{TenantInfo: tenant}
	hasProposal := proposalID != nil && *proposalID != ""
	hasPlan := planID != nil && *planID != ""
	if hasProposal == hasPlan {
		return nil, errortypes.NewValidationError(
			"proposalId", errortypes.ErrInvalid, "Name a proposal or a plan, not both or neither",
		)
	}

	var err error
	if hasProposal {
		req.ProposalID, err = pulid.MustParse(*proposalID)
	} else {
		req.PlanID, err = pulid.MustParse(*planID)
	}
	if err != nil {
		return nil, err
	}

	return req, nil
}
