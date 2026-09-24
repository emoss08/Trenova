package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CreateEvalCaseFromProposalRequest struct {
	ProposalID pulid.ID
	Title      string
	TenantInfo pagination.TenantInfo
}

type CreateEvalCaseFromMessageRequest struct {
	ThreadID   pulid.ID
	MessageID  pulid.ID
	Title      string
	TenantInfo pagination.TenantInfo
}

type CreateEvalCaseFromFeedbackRequest struct {
	FeedbackID pulid.ID
	Title      string
	TenantInfo pagination.TenantInfo
}

type CreateCuratedEvalCaseRequest struct {
	AgentDefinitionID pulid.ID
	Title             string
	Trigger           agent.RunTrigger
	Input             string
	PageContext       *agent.PageContext
	Mentions          []agent.EntityRef
	SubjectType       agent.SubjectType
	SubjectID         pulid.ID
	HeldTools         []string
	Expected          agentquality.Expected
	Rubric            string
	ExpiresAt         *int64
	TenantInfo        pagination.TenantInfo
}

type UpdateEvalCaseRequest struct {
	ID          pulid.ID
	Version     int64
	Title       *string
	Input       *string
	HeldTools   []string
	Expected    *agentquality.Expected
	Rubric      *string
	ExpiresAt   *int64
	ClearExpiry bool
	TenantInfo  pagination.TenantInfo
}

type SetEvalCaseStatusRequest struct {
	ID         pulid.ID
	Status     agentquality.CaseStatus
	TenantInfo pagination.TenantInfo
}

type EvalCaseCapture struct {
	Case      *agentquality.EvalCase
	Duplicate bool
}

type CaptureEvalCaseCandidatesRequest struct {
	Since int64
	Limit int
}

type CaptureEvalCaseCandidatesResult struct {
	Captured   int `json:"captured"`
	Duplicates int `json:"duplicates"`
	Failed     int `json:"failed"`
	Scanned    int `json:"scanned"`
}

type PurgeEvalCasesRequest struct {
	Now   int64
	Limit int
}

type PurgeEvalCasesResult struct {
	Retained int `json:"retained"`
	Expired  int `json:"expired"`
	Orphaned int `json:"orphaned"`
}

type AgentEvalCaseService interface {
	CreateFromProposal(
		ctx context.Context,
		req *CreateEvalCaseFromProposalRequest,
		actor *RequestActor,
	) (*EvalCaseCapture, error)
	CreateFromMessage(
		ctx context.Context,
		req *CreateEvalCaseFromMessageRequest,
		actor *RequestActor,
	) (*EvalCaseCapture, error)
	CreateFromFeedback(
		ctx context.Context,
		req *CreateEvalCaseFromFeedbackRequest,
		actor *RequestActor,
	) (*EvalCaseCapture, error)
	CreateCurated(
		ctx context.Context,
		req *CreateCuratedEvalCaseRequest,
		actor *RequestActor,
	) (*EvalCaseCapture, error)
	Update(
		ctx context.Context,
		req *UpdateEvalCaseRequest,
		actor *RequestActor,
	) (*agentquality.EvalCase, error)
	SetStatus(
		ctx context.Context,
		req *SetEvalCaseStatusRequest,
		actor *RequestActor,
	) (*agentquality.EvalCase, error)
	GetByID(
		ctx context.Context,
		req repositories.GetAgentEvalCaseByIDRequest,
	) (*agentquality.EvalCase, error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAgentEvalCaseConnectionRequest,
	) (*pagination.CursorListResult[*agentquality.EvalCase], error)
	CaptureCandidates(
		ctx context.Context,
		req CaptureEvalCaseCandidatesRequest,
	) (*CaptureEvalCaseCandidatesResult, error)
	Purge(ctx context.Context, req PurgeEvalCasesRequest) (*PurgeEvalCasesResult, error)
}
