package dispatchconsoleservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// UrgencyBucket groups moves the way a dispatcher triages them, so the board can lead
// with what is already broken rather than with whatever sorted first.
type UrgencyBucket string

const (
	UrgencyLate     = UrgencyBucket("Late")
	UrgencyNow      = UrgencyBucket("Now")
	UrgencyToday    = UrgencyBucket("Today")
	UrgencyTomorrow = UrgencyBucket("Tomorrow")
	UrgencyPlanned  = UrgencyBucket("Planned")
)

// BoardMove is a move card enriched with the derived facts the console renders.
type BoardMove struct {
	*repositories.BoardMove

	Urgency         UrgencyBucket `json:"urgency"`
	MinutesToPickup int64         `json:"minutesToPickup"`
	IsCovered       bool          `json:"isCovered"`
	TotalStopCount  int           `json:"totalStopCount"`
}

// DriverAvailability is the derived capacity state shown on a capacity-rail row.
type DriverAvailability string

const (
	AvailabilityOpen      = DriverAvailability("Open")
	AvailabilityFinishing = DriverAvailability("Finishing")
	AvailabilityWorking   = DriverAvailability("Working")
	AvailabilityBlocked   = DriverAvailability("Blocked")
	AvailabilityTimeOff   = DriverAvailability("TimeOff")
)

// BoardDriver is a capacity-rail row: who they are, their live clocks, when they free
// up, and what they are already committed to over the horizon.
type BoardDriver struct {
	*repositories.BoardDriver

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

	ProjectedTimeAvailable int64                             `json:"projectedTimeAvailable"`
	Commitments            []*repositories.WorkerCommitment  `json:"commitments"`
	TimeOff                []*repositories.WorkerTimeOff     `json:"timeOff"`
	Findings               []dispatcheligibility.Finding     `json:"findings"`
	CommittedMiles         float64                           `json:"committedMiles"`
	CommittedRevenue       float64                           `json:"committedRevenue"`
}

// Board is everything one console render needs, in one response.
type Board struct {
	Moves       []*BoardMove                 `json:"moves"`
	Drivers     []*BoardDriver               `json:"drivers"`
	Summary     *repositories.BoardSummary   `json:"summary"`
	WindowStart int64                        `json:"windowStart"`
	WindowEnd   int64                        `json:"windowEnd"`
	GeneratedAt int64                        `json:"generatedAt"`
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
}

type MoveCandidatesRequest struct {
	TenantInfo     pagination.TenantInfo
	MoveID         pulid.ID
	FleetCodeIDs   []pulid.ID
	Limit          int
	IncludeBlocked bool
}

// DriverMoveMatch is the reverse view: for one driver, which open moves suit them. It
// carries the same score object so both directions of matching explain themselves
// identically.
type DriverMoveMatch struct {
	Move  *BoardMove                              `json:"move"`
	Score *dispatchcandidateservice.CandidateScore `json:"score"`
}

type DriverMovesRequest struct {
	TenantInfo  pagination.TenantInfo
	WorkerID    pulid.ID
	WindowStart int64
	WindowEnd   int64
	Limit       int
}

// AssignmentPreview is the pre-flight a dispatcher sees before a drop commits.
type AssignmentPreview struct {
	MoveID    pulid.ID                                 `json:"moveId"`
	WorkerID  pulid.ID                                 `json:"workerId"`
	TractorID pulid.ID                                 `json:"tractorId"`
	TrailerID pulid.ID                                 `json:"trailerId"`
	Score     *dispatchcandidateservice.CandidateScore `json:"score"`
	Blocked   bool                                     `json:"blocked"`
	// RequiresOverride is true when the organization enforces compliance as a hard block
	// and the pairing trips one, so the console can label the confirm action honestly.
	RequiresOverride bool `json:"requiresOverride"`
}

type PreviewAssignmentRequest struct {
	TenantInfo pagination.TenantInfo
	MoveID     pulid.ID
	WorkerID   pulid.ID
	TractorID  pulid.ID
	TrailerID  pulid.ID
}
