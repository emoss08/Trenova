package dispatchconsoleports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatcheligibility"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type DispatchPlannedAssignment struct {
	MoveID         pulid.ID                                 `json:"moveId"`
	MoveVersion    int64                                    `json:"moveVersion"`
	ProNumber      string                                   `json:"proNumber"`
	WorkerID       pulid.ID                                 `json:"workerId"`
	WorkerName     string                                   `json:"workerName"`
	TractorID      pulid.ID                                 `json:"tractorId"`
	TrailerID      pulid.ID                                 `json:"trailerId"`
	Score          *dispatchcandidateservice.CandidateScore `json:"score"`
	Confidence     decimal.Decimal                          `json:"confidence"`
	Rationale      string                                   `json:"rationale"`
	AutoExecutable bool                                     `json:"autoExecutable"`
	ProposalID     pulid.ID                                 `json:"proposalId"`

	TourID                pulid.ID `json:"tourId"`
	SequenceIndex         int      `json:"sequenceIndex"`
	ProjectedStartAt      int64    `json:"projectedStartAt"`
	ProjectedCompleteAt   int64    `json:"projectedCompleteAt"`
	ProjectedDriveRemains int64    `json:"projectedDriveRemainingMs"`
}

type DispatchTour struct {
	TourID             pulid.ID   `json:"tourId"`
	WorkerID           pulid.ID   `json:"workerId"`
	WorkerName         string     `json:"workerName"`
	MoveIDs            []pulid.ID `json:"moveIds"`
	StartsAt           int64      `json:"startsAt"`
	EndsAt             int64      `json:"endsAt"`
	TotalScore         int        `json:"totalScore"`
	TotalDeadheadMiles float64    `json:"totalDeadheadMiles"`
}

type DispatchUncoveredMove struct {
	MoveID    pulid.ID `json:"moveId"`
	ProNumber string   `json:"proNumber"`
	Reason    string   `json:"reason"`
	// StartsAt is the origin window, which is what decides whether an
	// uncovered move is still a planning problem or has become somebody's
	// afternoon.
	StartsAt            int64                         `json:"startsAt"`
	BestBlockedFindings []dispatcheligibility.Finding `json:"bestBlockedFindings"`
}

type DispatchPlan struct {
	RunID        pulid.ID                     `json:"runId"`
	Assignments  []*DispatchPlannedAssignment `json:"assignments"`
	Uncovered    []*DispatchUncoveredMove     `json:"uncovered"`
	Tours        []*DispatchTour              `json:"tours"`
	PlanningMode string                       `json:"planningMode"`
	ShadowMode   bool                         `json:"shadowMode"`
	AutonomyTier agent.AutonomyTier           `json:"autonomyTier"`
	TotalScore   int                          `json:"totalScore"`
	GeneratedAt  int64                        `json:"generatedAt"`
}

type DispatchPlanRequest struct {
	TenantInfo   pagination.TenantInfo
	WindowStart  int64
	WindowEnd    int64
	FleetCodeIDs []pulid.ID
	MoveIDs      []pulid.ID
	Apply        bool
	// Trigger is why the plan ran: a person asking for it from the console
	// (Manual), the board changing (Event) or the hourly catch-up (Scheduled).
	// It is recorded on the run.
	Trigger agent.RunTrigger
}

// CoverageRefreshRequester asks for an organization's coverage to be planned
// again soon. Requests close together collapse into one plan.
type CoverageRefreshRequester interface {
	RequestRefresh(ctx context.Context, tenantInfo pagination.TenantInfo)
}

// CoverageMove is a move on the board as the board read it: the version tells a
// proposal planned against an older move from one still current.
type CoverageMove struct {
	MoveID      pulid.ID
	MoveVersion int64
}

type CoverageProposalsRequest struct {
	TenantInfo pagination.TenantInfo
	Moves      []CoverageMove
}

// PendingCoverage is the coverage proposal waiting on a move: who the agent
// would put on it and why. Stale is true when the move changed after it was
// planned, so approving it would be refused.
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

// FailedCoverage is a coverage proposal someone approved whose assignment was
// refused, kept on its move while the move is as it was when it failed, so the
// board says why the move is still uncovered instead of forgetting the approval.
type FailedCoverage struct {
	ProposalID pulid.ID
	MoveID     pulid.ID
	WorkerID   pulid.ID
	WorkerName string
	Reason     string
	FailedAt   int64
}

// MoveCoverageProposals is what the agent has on one move: the proposal waiting
// on it, and the last approval of one that failed, when either exists.
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
	) ([]*dispatchcandidateservice.CandidateScore, error)
	GetDriverMoves(
		ctx context.Context,
		req *dispatchconsoleservice.DriverMovesRequest,
	) ([]*dispatchconsoleservice.DriverMoveMatch, error)
	PreviewAssignment(
		ctx context.Context,
		req *dispatchconsoleservice.PreviewAssignmentRequest,
	) (*dispatchconsoleservice.AssignmentPreview, error)
}
