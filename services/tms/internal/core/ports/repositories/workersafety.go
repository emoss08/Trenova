package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListWorkerSafetyEventsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	// Since limits the list to events on or after this instant; zero means all.
	Since           int64 `json:"since"`
	IncludeDocument bool  `json:"includeDocument"`
	IncludeActors   bool  `json:"includeActors"`
}

type GetWorkerSafetyEventByIDRequest struct {
	ID              pulid.ID              `json:"id"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	IncludeWorker   bool                  `json:"includeWorker"`
	IncludeDocument bool                  `json:"includeDocument"`
}

type ListWorkerDisciplinaryActionsRequest struct {
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	WorkerID      pulid.ID              `json:"workerId"`
	IncludeEvent  bool                  `json:"includeEvent"`
	IncludeActors bool                  `json:"includeActors"`
}

type GetWorkerDisciplinaryActionByIDRequest struct {
	ID           pulid.ID              `json:"id"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	IncludeEvent bool                  `json:"includeEvent"`
}

type ListWorkerRecognitionsRequest struct {
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	WorkerID      pulid.ID              `json:"workerId"`
	VisibleOnly   bool                  `json:"visibleOnly"`
	IncludeActors bool                  `json:"includeActors"`
}

type GetWorkerRecognitionByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

// ListWorkerSafetyViolationsRequest lists the violations cited on an event, or
// every violation a worker has collected.
type ListWorkerSafetyViolationsRequest struct {
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	SafetyEventID pulid.ID              `json:"safetyEventId"`
	WorkerID      pulid.ID              `json:"workerId"`
	// Since limits to violations whose event happened on or after this
	// instant; zero means all.
	Since        int64 `json:"since"`
	IncludeEvent bool  `json:"includeEvent"`
}

type GetWorkerSafetyViolationByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

// FleetSafetyRequest scopes the fleet roll-up. The window is how far back the
// counts and the trend reach; the BASIC scores always use the FMCSA's own
// twenty-four month look-back regardless, because a BASIC measured over
// anything else is not a BASIC.
type FleetSafetyRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	// WindowMonths is the counting window, defaulted by the service.
	WindowMonths int `json:"windowMonths"`
	// FleetCodeID narrows every section to one terminal.
	FleetCodeID pulid.ID `json:"fleetCodeId"`
	// RankLimit is how many drivers the ranking carries.
	RankLimit int `json:"rankLimit"`
	// RankBest flips the ranking from worst-first to best-first. The two lists
	// are the same query read from opposite ends, so they can never disagree
	// about where a driver sits.
	RankBest bool  `json:"rankBest"`
	Now      int64 `json:"now"`
}

// FleetSafetyRatingRow counts the roster in one safety rating.
type FleetSafetyRatingRow struct {
	Rating     worker.SafetyRating `bun:"rating"`
	Workers    int                 `bun:"workers"`
	TotalScore int                 `bun:"total_score"`
}

// FleetSafetyTerminalRow is one terminal's standing, read from the roster
// cache rather than recomputed: the cache is what the roster itself shows, so
// the fleet view and the list behind it cannot disagree.
type FleetSafetyTerminalRow struct {
	FleetCodeID          pulid.ID `bun:"fleet_code_id"`
	FleetCodeCode        string   `bun:"fleet_code_code"`
	FleetCodeDescription string   `bun:"fleet_code_description"`
	FleetCodeColor       string   `bun:"fleet_code_color"`
	Workers              int      `bun:"workers"`
	AtRisk               int      `bun:"at_risk"`
	Watch                int      `bun:"watch"`
	TotalScore           int      `bun:"total_score"`
}

// FleetSafetyKindRow counts events of one kind inside the window.
type FleetSafetyKindRow struct {
	Kind         worker.SafetyEventKind `bun:"kind"`
	Events       int                    `bun:"events"`
	Points       int                    `bun:"points"`
	Preventable  int                    `bun:"preventable"`
	OutOfService int                    `bun:"out_of_service"`
	Open         int                    `bun:"open_events"`
}

// FleetSafetyBasicRow is one BASIC's violations in one recency bucket. The
// bucket rather than the weighted total comes back from SQL so the weights
// stay in the domain, where they can be tested without a database.
type FleetSafetyBasicRow struct {
	Basic        worker.CSABasic `bun:"basic"`
	Bucket       int             `bun:"bucket"`
	Violations   int             `bun:"violations"`
	SeveritySum  int             `bun:"severity_sum"`
	OutOfService int             `bun:"out_of_service"`
}

// FleetSafetyEventBasicRow is the fallback for events nobody keyed violations
// into: the event's own points, grouped by the BASIC its kind suggests.
type FleetSafetyEventBasicRow struct {
	Kind             worker.SafetyEventKind  `bun:"kind"`
	InspectionResult worker.InspectionResult `bun:"inspection_result"`
	Bucket           int                     `bun:"bucket"`
	Events           int                     `bun:"events"`
	Points           int                     `bun:"points"`
	OutOfService     int                     `bun:"out_of_service"`
}

// FleetSafetyTrendRow is one month of the trend line.
type FleetSafetyTrendRow struct {
	PeriodStart  int64 `bun:"period_start"`
	Events       int   `bun:"events"`
	Accidents    int   `bun:"accidents"`
	Preventable  int   `bun:"preventable"`
	Citations    int   `bun:"citations"`
	Inspections  int   `bun:"inspections"`
	OutOfService int   `bun:"out_of_service"`
	Points       int   `bun:"points"`
}

// FleetSafetyRankRow is one driver's line in the ranking.
type FleetSafetyRankRow struct {
	WorkerID       pulid.ID            `bun:"worker_id"`
	FirstName      string              `bun:"first_name"`
	LastName       string              `bun:"last_name"`
	FleetCodeID    pulid.ID            `bun:"fleet_code_id"`
	FleetCodeCode  string              `bun:"fleet_code_code"`
	FleetCodeColor string              `bun:"fleet_code_color"`
	Rating         worker.SafetyRating `bun:"rating"`
	Score          int                 `bun:"score"`
	ActivePoints   int                 `bun:"active_points"`
	Events         int                 `bun:"events"`
	LastEventAt    *int64              `bun:"last_event_at"`
}

type WorkerSafetyRepository interface {
	ListEvents(
		ctx context.Context,
		req *ListWorkerSafetyEventsRequest,
	) ([]*worker.WorkerSafetyEvent, error)
	GetEventByID(
		ctx context.Context,
		req *GetWorkerSafetyEventByIDRequest,
	) (*worker.WorkerSafetyEvent, error)
	CreateEvent(ctx context.Context, entity *worker.WorkerSafetyEvent) (*worker.WorkerSafetyEvent, error)
	UpdateEvent(ctx context.Context, entity *worker.WorkerSafetyEvent) (*worker.WorkerSafetyEvent, error)
	DeleteEvent(ctx context.Context, req *GetWorkerSafetyEventByIDRequest) error

	ListActions(
		ctx context.Context,
		req *ListWorkerDisciplinaryActionsRequest,
	) ([]*worker.WorkerDisciplinaryAction, error)
	GetActionByID(
		ctx context.Context,
		req *GetWorkerDisciplinaryActionByIDRequest,
	) (*worker.WorkerDisciplinaryAction, error)
	CreateAction(
		ctx context.Context,
		entity *worker.WorkerDisciplinaryAction,
	) (*worker.WorkerDisciplinaryAction, error)
	UpdateAction(
		ctx context.Context,
		entity *worker.WorkerDisciplinaryAction,
	) (*worker.WorkerDisciplinaryAction, error)

	ListRecognitions(
		ctx context.Context,
		req *ListWorkerRecognitionsRequest,
	) ([]*worker.WorkerRecognition, error)
	GetRecognitionByID(
		ctx context.Context,
		req *GetWorkerRecognitionByIDRequest,
	) (*worker.WorkerRecognition, error)
	CreateRecognition(
		ctx context.Context,
		entity *worker.WorkerRecognition,
	) (*worker.WorkerRecognition, error)
	DeleteRecognition(ctx context.Context, req *GetWorkerRecognitionByIDRequest) error
	ListWorkersWithLapsedPoints(
		ctx context.Context,
		req *ListWorkersWithLapsedPointsRequest,
	) ([]WorkerTenantRef, error)

	ListViolations(
		ctx context.Context,
		req *ListWorkerSafetyViolationsRequest,
	) ([]*worker.WorkerSafetyViolation, error)
	GetViolationByID(
		ctx context.Context,
		req *GetWorkerSafetyViolationByIDRequest,
	) (*worker.WorkerSafetyViolation, error)
	CreateViolation(
		ctx context.Context,
		entity *worker.WorkerSafetyViolation,
	) (*worker.WorkerSafetyViolation, error)
	UpdateViolation(
		ctx context.Context,
		entity *worker.WorkerSafetyViolation,
	) (*worker.WorkerSafetyViolation, error)
	DeleteViolation(ctx context.Context, req *GetWorkerSafetyViolationByIDRequest) error

	// The fleet roll-up is six grouped queries rather than one walk of every
	// worker: the numbers are aggregates, and pulling a few thousand drivers
	// into memory to count them would be the wrong shape at any fleet size.
	FleetRatings(ctx context.Context, req *FleetSafetyRequest) ([]FleetSafetyRatingRow, error)
	FleetTerminals(ctx context.Context, req *FleetSafetyRequest) ([]FleetSafetyTerminalRow, error)
	FleetKinds(ctx context.Context, req *FleetSafetyRequest) ([]FleetSafetyKindRow, error)
	FleetBasics(ctx context.Context, req *FleetSafetyRequest) ([]FleetSafetyBasicRow, error)
	FleetEventBasics(
		ctx context.Context,
		req *FleetSafetyRequest,
	) ([]FleetSafetyEventBasicRow, error)
	FleetTrend(ctx context.Context, req *FleetSafetyRequest) ([]FleetSafetyTrendRow, error)
	FleetRanking(ctx context.Context, req *FleetSafetyRequest) ([]FleetSafetyRankRow, error)
}

// WorkerTenantRef names a worker and the tenant they belong to, for sweeps
// that work across tenants.
type WorkerTenantRef struct {
	WorkerID       pulid.ID `bun:"worker_id"`
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

// ListWorkersWithLapsedPointsRequest finds the workers whose safety standing
// may have moved with nobody writing anything, because points expired inside
// the window. Sweeping every worker nightly would cost six queries a head for
// no reason; only these can have changed.
type ListWorkersWithLapsedPointsRequest struct {
	Since int64 `json:"since"`
	Until int64 `json:"until"`
	Limit int   `json:"limit"`
}
