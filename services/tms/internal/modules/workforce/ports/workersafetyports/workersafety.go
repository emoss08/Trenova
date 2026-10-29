package workersafetyports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/fleetsafety"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdiscipline"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListWorkerSafetyEventsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	// WorkerID narrows to one worker; nil lists every worker's, which a
	// caller bounds with Limit.
	WorkerID pulid.ID `json:"workerId"`
	OpenOnly bool     `json:"openOnly"`
	Limit    int      `json:"limit"`
	// Since limits the list to events on or after this instant; zero means all.
	Since           int64 `json:"since"`
	IncludeDocument bool  `json:"includeDocument"`
	IncludeActors   bool  `json:"includeActors"`
	IncludeWorker   bool  `json:"includeWorker"`
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

// ListWorkerSafetyViolationsRequest lists the violations cited on events, or
// every violation a worker has collected.
type ListWorkerSafetyViolationsRequest struct {
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	SafetyEventIDs []pulid.ID            `json:"safetyEventIds"`
	WorkerID       pulid.ID              `json:"workerId"`
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
	Rating     worker.SafetyRating
	Workers    int
	TotalScore int
}

// FleetSafetyTerminalRow is one terminal's standing, read from the roster
// cache rather than recomputed: the cache is what the roster itself shows, so
// the fleet view and the list behind it cannot disagree.
type FleetSafetyTerminalRow struct {
	FleetCodeID          pulid.ID
	FleetCodeCode        string
	FleetCodeDescription string
	FleetCodeColor       string
	Workers              int
	AtRisk               int
	Watch                int
	TotalScore           int
}

// FleetSafetyKindRow counts events of one kind inside the window.
type FleetSafetyKindRow struct {
	Kind         workersafety.SafetyEventKind
	Events       int
	Points       int
	Preventable  int
	OutOfService int
	Open         int
}

// FleetSafetyBasicRow is one BASIC's violations in one recency bucket. The
// bucket rather than the weighted total comes back from SQL so the weights
// stay in the domain, where they can be tested without a database.
type FleetSafetyBasicRow struct {
	Basic        fleetsafety.CSABasic
	Bucket       int
	Violations   int
	SeveritySum  int
	OutOfService int
}

// FleetSafetyEventBasicRow is the fallback for events nobody keyed violations
// into: the event's own points, grouped by the BASIC its kind suggests.
type FleetSafetyEventBasicRow struct {
	Kind             workersafety.SafetyEventKind
	InspectionResult workersafety.InspectionResult
	Bucket           int
	Events           int
	Points           int
	OutOfService     int
}

// FleetSafetyTrendRow is one month of the trend line.
type FleetSafetyTrendRow struct {
	PeriodStart  int64
	Events       int
	Accidents    int
	Preventable  int
	Citations    int
	Inspections  int
	OutOfService int
	Points       int
}

// FleetSafetyRankRow is one driver's line in the ranking.
type FleetSafetyRankRow struct {
	WorkerID       pulid.ID
	FirstName      string
	LastName       string
	FleetCodeID    pulid.ID
	FleetCodeCode  string
	FleetCodeColor string
	Rating         worker.SafetyRating
	Score          int
	ActivePoints   int
	Events         int
	LastEventAt    *int64
}

type WorkerSafetyRepository interface {
	ListEvents(
		ctx context.Context,
		req *ListWorkerSafetyEventsRequest,
	) ([]*workersafety.WorkerSafetyEvent, error)
	GetEventByID(
		ctx context.Context,
		req *GetWorkerSafetyEventByIDRequest,
	) (*workersafety.WorkerSafetyEvent, error)
	CreateEvent(
		ctx context.Context,
		entity *workersafety.WorkerSafetyEvent,
	) (*workersafety.WorkerSafetyEvent, error)
	UpdateEvent(
		ctx context.Context,
		entity *workersafety.WorkerSafetyEvent,
	) (*workersafety.WorkerSafetyEvent, error)
	DeleteEvent(ctx context.Context, req *GetWorkerSafetyEventByIDRequest) error

	ListActions(
		ctx context.Context,
		req *ListWorkerDisciplinaryActionsRequest,
	) ([]*workerdiscipline.WorkerDisciplinaryAction, error)
	GetActionByID(
		ctx context.Context,
		req *GetWorkerDisciplinaryActionByIDRequest,
	) (*workerdiscipline.WorkerDisciplinaryAction, error)
	CreateAction(
		ctx context.Context,
		entity *workerdiscipline.WorkerDisciplinaryAction,
	) (*workerdiscipline.WorkerDisciplinaryAction, error)
	UpdateAction(
		ctx context.Context,
		entity *workerdiscipline.WorkerDisciplinaryAction,
	) (*workerdiscipline.WorkerDisciplinaryAction, error)

	ListRecognitions(
		ctx context.Context,
		req *ListWorkerRecognitionsRequest,
	) ([]*workerdiscipline.WorkerRecognition, error)
	GetRecognitionByID(
		ctx context.Context,
		req *GetWorkerRecognitionByIDRequest,
	) (*workerdiscipline.WorkerRecognition, error)
	CreateRecognition(
		ctx context.Context,
		entity *workerdiscipline.WorkerRecognition,
	) (*workerdiscipline.WorkerRecognition, error)
	DeleteRecognition(ctx context.Context, req *GetWorkerRecognitionByIDRequest) error
	ListWorkersWithLapsedPoints(
		ctx context.Context,
		req *ListWorkersWithLapsedPointsRequest,
	) ([]workerports.WorkerTenantRef, error)

	ListViolations(
		ctx context.Context,
		req *ListWorkerSafetyViolationsRequest,
	) ([]*fleetsafety.WorkerSafetyViolation, error)
	GetViolationByID(
		ctx context.Context,
		req *GetWorkerSafetyViolationByIDRequest,
	) (*fleetsafety.WorkerSafetyViolation, error)
	CreateViolation(
		ctx context.Context,
		entity *fleetsafety.WorkerSafetyViolation,
	) (*fleetsafety.WorkerSafetyViolation, error)
	UpdateViolation(
		ctx context.Context,
		entity *fleetsafety.WorkerSafetyViolation,
	) (*fleetsafety.WorkerSafetyViolation, error)
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

// ListWorkersWithLapsedPointsRequest finds the workers whose safety standing
// may have moved with nobody writing anything, because points expired inside
// the window. Sweeping every worker nightly would cost six queries a head for
// no reason; only these can have changed.
type ListWorkersWithLapsedPointsRequest struct {
	Since int64 `json:"since"`
	Until int64 `json:"until"`
	Limit int   `json:"limit"`
}
