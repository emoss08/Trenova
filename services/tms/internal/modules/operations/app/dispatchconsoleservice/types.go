package dispatchconsoleservice

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchurgency"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type MoveTenderSummary struct {
	ID                    pulid.ID      `json:"id"`
	Status                tender.Status `json:"status"`
	Mode                  tender.Mode   `json:"mode"`
	CurrentRank           int16         `json:"currentRank"`
	OfferCount            int           `json:"offerCount"`
	CurrentCarrierName    string        `json:"currentCarrierName"`
	CurrentOfferExpiresAt *int64        `json:"currentOfferExpiresAt"`
}

type BoardMove struct {
	*dispatchboardports.BoardMove

	Urgency         dispatchurgency.Bucket               `json:"urgency"`
	MinutesToPickup int64                                `json:"minutesToPickup"`
	IsCovered       bool                                 `json:"isCovered"`
	TotalStopCount  int                                  `json:"totalStopCount"`
	LiveTender      *MoveTenderSummary                   `json:"liveTender"`
	PlanPrice       *dispatchplanprice.DispatchPlanPrice `json:"planPrice"`
}

type DriverAvailability string

const (
	AvailabilityOpen      = DriverAvailability("Open")
	AvailabilityFinishing = DriverAvailability("Finishing")
	AvailabilityWorking   = DriverAvailability("Working")
	AvailabilityBlocked   = DriverAvailability("Blocked")
	AvailabilityTimeOff   = DriverAvailability("TimeOff")
)

type BoardDriver struct {
	*dispatchboardports.BoardDriver

	Availability     DriverAvailability `json:"availability"`
	DutyStatus       string             `json:"dutyStatus"`
	DriveRemainingMs int64              `json:"driveRemainingMs"`
	ShiftRemainingMs int64              `json:"shiftRemainingMs"`
	CycleRemainingMs int64              `json:"cycleRemainingMs"`
	BreakRemainingMs int64              `json:"breakRemainingMs"`
	HOSRecordedAt    int64              `json:"hosRecordedAt"`
	HOSIsStale       bool               `json:"hosIsStale"`

	Latitude          *float64 `json:"latitude"`
	Longitude         *float64 `json:"longitude"`
	FormattedLocation string   `json:"formattedLocation"`
	PositionAt        int64    `json:"positionRecordedAt"`

	ProjectedTimeAvailable int64                                  `json:"projectedTimeAvailable"`
	Commitments            []*dispatchboardports.WorkerCommitment `json:"commitments"`
	TimeOff                []*dispatchboardports.WorkerTimeOff    `json:"timeOff"`
	Findings               []dispatchfinding.Finding              `json:"findings"`
	CommittedMiles         float64                                `json:"committedMiles"`
	CommittedRevenue       float64                                `json:"committedRevenue"`
	PlannedMove            *PlannedMove                           `json:"plannedMove"`
}

type PlannedMove struct {
	MoveID      pulid.ID `json:"moveId"`
	ProNumber   string   `json:"proNumber"`
	CostCents   int64    `json:"costCents"`
	ValueCents  int64    `json:"valueCents"`
	GeneratedAt int64    `json:"generatedAt"`
}

type Board struct {
	Moves            []*BoardMove                     `json:"moves"`
	Drivers          []*BoardDriver                   `json:"drivers"`
	DriversTruncated bool                             `json:"driversTruncated"`
	Summary          *dispatchboardports.BoardSummary `json:"summary"`
	WindowStart      int64                            `json:"windowStart"`
	WindowEnd        int64                            `json:"windowEnd"`
	GeneratedAt      int64                            `json:"generatedAt"`
}

type GetBoardRequest struct {
	TenantInfo     pagination.TenantInfo
	WindowStart    int64
	WindowEnd      int64
	FleetCodeIDs   []pulid.ID
	CustomerIDs    []pulid.ID
	ServiceTypeIDs []pulid.ID
	WorkerIDs      []pulid.ID
	Query          string
	IncludeCovered bool
	Limit          int
	// LateOnly lists only the moves late now; the summary still counts
	// every move in the window.
	LateOnly bool
	// DueFirst orders the moves by when their next stop is due, furthest
	// behind first, and Offset skips that many of them for the next page.
	DueFirst bool
	Offset   int
}

type MoveCandidatesRequest struct {
	TenantInfo     pagination.TenantInfo
	MoveID         pulid.ID
	FleetCodeIDs   []pulid.ID
	Limit          int
	IncludeBlocked bool
}

type DriverMoveMatch struct {
	Move  *BoardMove                        `json:"move"`
	Score *dispatchplanports.CandidateScore `json:"score"`
}

type DriverMovesRequest struct {
	TenantInfo  pagination.TenantInfo
	WorkerID    pulid.ID
	WindowStart int64
	WindowEnd   int64
	Limit       int
}

type AssignmentPreview struct {
	MoveID           pulid.ID                          `json:"moveId"`
	WorkerID         pulid.ID                          `json:"workerId"`
	TractorID        pulid.ID                          `json:"tractorId"`
	TrailerID        pulid.ID                          `json:"trailerId"`
	Score            *dispatchplanports.CandidateScore `json:"score"`
	Blocked          bool                              `json:"blocked"`
	RequiresOverride bool                              `json:"requiresOverride"`
}

type PreviewAssignmentRequest struct {
	TenantInfo pagination.TenantInfo
	MoveID     pulid.ID
	WorkerID   pulid.ID
	TractorID  pulid.ID
	TrailerID  pulid.ID
}
