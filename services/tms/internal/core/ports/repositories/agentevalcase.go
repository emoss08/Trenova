package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAgentEvalCaseByIDRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type GetAgentEvalCaseByContentRequest struct {
	AgentDefinitionID pulid.ID
	ContentHash       string
	TenantInfo        pagination.TenantInfo
}

type GetAgentEvalCaseByProposalRequest struct {
	ProposalID pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListAgentEvalCaseConnectionRequest struct {
	Filter            *pagination.QueryOptions  `json:"filter"`
	Cursor            pagination.CursorInfo     `json:"-"`
	Columns           []string                  `json:"-"`
	AgentDefinitionID pulid.ID                  `json:"agentDefinitionId"`
	Statuses          []agentquality.CaseStatus `json:"statuses"`
	Sources           []agentquality.CaseSource `json:"sources"`
}

type ListAgentEvalCasesRequest struct {
	AgentDefinitionID pulid.ID
	Statuses          []agentquality.CaseStatus
	TenantInfo        pagination.TenantInfo
	Limit             int
}

type ListEvalCaseCaptureCandidatesRequest struct {
	Since int64
	Limit int
}

type EvalCaseCaptureCandidate struct {
	OrganizationID    pulid.ID `bun:"organization_id"`
	BusinessUnitID    pulid.ID `bun:"business_unit_id"`
	ProposalID        pulid.ID `bun:"proposal_id"`
	DecisionID        pulid.ID `bun:"decision_id"`
	DecidedByUserID   pulid.ID `bun:"decided_by_user_id"`
	AgentDefinitionID pulid.ID `bun:"agent_definition_id"`
	DecidedAt         int64    `bun:"decided_at"`
}

func (c EvalCaseCaptureCandidate) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: c.OrganizationID, BuID: c.BusinessUnitID}
}

type PurgeExpiredEvalCasesRequest struct {
	TenantInfo    pagination.TenantInfo
	AllTenants    bool
	CreatedBefore int64
	Now           int64
	Limit         int
}

type PurgeOrphanedEvalCasesRequest struct {
	Limit int
}

type AgentEvalCaseRepository interface {
	Create(ctx context.Context, entity *agentquality.EvalCase) (*agentquality.EvalCase, error)
	Update(ctx context.Context, entity *agentquality.EvalCase) (*agentquality.EvalCase, error)
	GetByID(
		ctx context.Context,
		req GetAgentEvalCaseByIDRequest,
	) (*agentquality.EvalCase, error)
	GetByContent(
		ctx context.Context,
		req GetAgentEvalCaseByContentRequest,
	) (*agentquality.EvalCase, error)
	GetByProposal(
		ctx context.Context,
		req GetAgentEvalCaseByProposalRequest,
	) (*agentquality.EvalCase, error)
	ListConnection(
		ctx context.Context,
		req *ListAgentEvalCaseConnectionRequest,
	) (*pagination.CursorListResult[*agentquality.EvalCase], error)
	ListByAgent(
		ctx context.Context,
		req ListAgentEvalCasesRequest,
	) ([]*agentquality.EvalCase, error)
	ListCaptureCandidates(
		ctx context.Context,
		req ListEvalCaseCaptureCandidatesRequest,
	) ([]EvalCaseCaptureCandidate, error)
	PurgeExpired(ctx context.Context, req PurgeExpiredEvalCasesRequest) (int, error)
	PurgeOrphaned(ctx context.Context, req PurgeOrphanedEvalCasesRequest) (int, error)
}
