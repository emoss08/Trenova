package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type DispatchPlannedAssignment struct {
	MoveID         pulid.ID                                 `json:"moveId"`
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
	MoveID              pulid.ID                      `json:"moveId"`
	ProNumber           string                        `json:"proNumber"`
	Reason              string                        `json:"reason"`
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
}

type DispatchAutoAssignService interface {
	Plan(ctx context.Context, req *DispatchPlanRequest) (*DispatchPlan, error)
}

type DispatchConsoleService interface {
	GetBoard(
		ctx context.Context,
		req *dispatchconsoleservice.GetBoardRequest,
	) (*dispatchconsoleservice.Board, error)
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
