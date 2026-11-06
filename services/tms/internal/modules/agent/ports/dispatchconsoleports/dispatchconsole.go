package dispatchconsoleports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type DispatchPlannedAssignment struct {
	*dispatchplanports.Pairing

	Confidence     decimal.Decimal `json:"confidence"`
	Rationale      string          `json:"rationale"`
	AutoExecutable bool            `json:"autoExecutable"`
	ProposalID     pulid.ID        `json:"proposalId"`
}

type DispatchPlan struct {
	RunID          pulid.ID                       `json:"runId"`
	Mode           dispatchcontrol.PlanningMode   `json:"mode"`
	GeneratedAt    int64                          `json:"generatedAt"`
	Assignments    []*DispatchPlannedAssignment   `json:"assignments"`
	Uncovered      []*dispatchplanports.Uncovered `json:"uncovered"`
	Tours          []*dispatchplanports.Tour      `json:"tours"`
	CostCents      int64                          `json:"costCents"`
	UncoveredCents int64                          `json:"uncoveredCents"`
	ShadowMode     bool                           `json:"shadowMode"`
	AutonomyTier   agent.AutonomyTier             `json:"autonomyTier"`
}

type DispatchPlanRequest struct {
	TenantInfo   pagination.TenantInfo
	WindowStart  int64
	WindowEnd    int64
	FleetCodeIDs []pulid.ID
	MoveIDs      []pulid.ID
	Apply        bool
	Trigger      agent.RunTrigger
}

type CoverageRefreshRequester interface {
	RequestRefresh(ctx context.Context, tenantInfo pagination.TenantInfo)
}

type CoverageMove struct {
	MoveID      pulid.ID
	MoveVersion int64
}

type CoverageProposalsRequest struct {
	TenantInfo pagination.TenantInfo
	Moves      []CoverageMove
}

type PendingCoverage struct {
	ProposalID pulid.ID
	MoveID     pulid.ID
	WorkerID   pulid.ID
	WorkerName string
	TractorID  pulid.ID
	TrailerID  pulid.ID
	Confidence decimal.Decimal
	Rationale  string
	CreatedAt  int64
	ExpiresAt  int64
	Stale      bool
}

type FailedCoverage struct {
	ProposalID pulid.ID
	MoveID     pulid.ID
	WorkerID   pulid.ID
	WorkerName string
	Reason     string
	FailedAt   int64
}

type MoveCoverageProposals struct {
	Pending *PendingCoverage
	Failed  *FailedCoverage
}

type DispatchAutoAssignService interface {
	Plan(ctx context.Context, req *DispatchPlanRequest) (*DispatchPlan, error)
	CoverageProposals(
		ctx context.Context,
		req *CoverageProposalsRequest,
	) (map[pulid.ID]*MoveCoverageProposals, error)
}

type DispatchConsoleService interface {
	GetBoard(
		ctx context.Context,
		req *dispatchconsoleservice.GetBoardRequest,
	) (*dispatchconsoleservice.Board, error)
	GetBoardMoves(
		ctx context.Context,
		req *dispatchconsoleservice.GetBoardRequest,
	) ([]*dispatchconsoleservice.BoardMove, error)
	GetMove(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
	) (*dispatchconsoleservice.BoardMove, error)
	GetMoveCandidates(
		ctx context.Context,
		req *dispatchconsoleservice.MoveCandidatesRequest,
	) ([]*dispatchplanports.CandidateScore, error)
	GetDriverMoves(
		ctx context.Context,
		req *dispatchconsoleservice.DriverMovesRequest,
	) ([]*dispatchconsoleservice.DriverMoveMatch, error)
	PreviewAssignment(
		ctx context.Context,
		req *dispatchconsoleservice.PreviewAssignmentRequest,
	) (*dispatchconsoleservice.AssignmentPreview, error)
}
