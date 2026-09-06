package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListShiftTemplatesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	ActiveOnly bool                  `json:"activeOnly"`
	Limit      int                   `json:"limit"`
}

type GetShiftTemplateByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CountShiftTemplateAssignmentsRequest struct {
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	TemplateIDs []pulid.ID            `json:"templateIds"`
}

type ListShiftAssignmentsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	// ActiveAt limits to the assignment in force at that instant; zero returns
	// the whole history, which is what a worker's own page shows.
	ActiveAt        int64 `json:"activeAt"`
	IncludeTemplate bool  `json:"includeTemplate"`
	Limit           int   `json:"limit"`
}

type GetShiftAssignmentByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListAvailabilityPreferencesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
}

type ListShiftSwapsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	// OpenOnly narrows to the swaps still going somewhere, which is the queue
	// a manager works.
	OpenOnly       bool  `json:"openOnly"`
	Since          int64 `json:"since"`
	IncludeWorkers bool  `json:"includeWorkers"`
	Limit          int   `json:"limit"`
}

type GetShiftSwapByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

// RotaWorkerRow is one worker's roster line: who they are, where they work,
// and which shift pattern they are on for the week being drawn.
type RotaWorkerRow struct {
	WorkerID         pulid.ID `bun:"worker_id"`
	FirstName        string   `bun:"first_name"`
	LastName         string   `bun:"last_name"`
	FleetCode        string   `bun:"fleet_code"`
	FleetColor       string   `bun:"fleet_color"`
	ShiftTemplateID  pulid.ID `bun:"shift_template_id"`
	CycleOffsetWeeks int16    `bun:"cycle_offset_weeks"`
}

// RotaDayRow is a count of something on one worker's one day.
type RotaDayRow struct {
	WorkerID pulid.ID `bun:"worker_id"`
	DayStart int64    `bun:"day_start"`
	Count    int      `bun:"count"`
}

// RotaRangeRow is a span that overrides the pattern — time off, or a leave
// case. It comes back as the range it is stored as and is expanded into days
// by the composer: expanding in SQL would need generate_series, which is
// Postgres-only, and a week is seven days.
//
// EndsAt of zero means an open-ended span, which covers the rest of the week.
type RotaRangeRow struct {
	WorkerID pulid.ID `bun:"worker_id"`
	StartsAt int64    `bun:"starts_at"`
	EndsAt   int64    `bun:"ends_at"`
}

// RotaQuery scopes the week being drawn.
type RotaQuery struct {
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	WeekStart   int64                 `json:"weekStart"`
	WeekEnd     int64                 `json:"weekEnd"`
	FleetCodeID pulid.ID              `json:"fleetCodeId"`
	// ManagerIDs narrows the rota to the people a manager answers for. Empty
	// draws the whole roster.
	ManagerIDs []pulid.ID `json:"managerIds"`
	Limit      int        `json:"limit"`
}

type SchedulingRepository interface {
	ListTemplates(
		ctx context.Context,
		req *ListShiftTemplatesRequest,
	) ([]*worker.ShiftTemplate, error)
	GetTemplateByID(
		ctx context.Context,
		req *GetShiftTemplateByIDRequest,
	) (*worker.ShiftTemplate, error)
	CreateTemplate(
		ctx context.Context,
		entity *worker.ShiftTemplate,
	) (*worker.ShiftTemplate, error)
	UpdateTemplate(
		ctx context.Context,
		entity *worker.ShiftTemplate,
	) (*worker.ShiftTemplate, error)
	CountTemplateAssignments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		templateID pulid.ID,
	) (int, error)
	CountTemplateAssignmentsByIDs(
		ctx context.Context,
		req *CountShiftTemplateAssignmentsRequest,
	) (map[pulid.ID]int, error)

	ListAssignments(
		ctx context.Context,
		req *ListShiftAssignmentsRequest,
	) ([]*worker.WorkerShiftAssignment, error)
	GetAssignmentByID(
		ctx context.Context,
		req *GetShiftAssignmentByIDRequest,
	) (*worker.WorkerShiftAssignment, error)
	CreateAssignment(
		ctx context.Context,
		entity *worker.WorkerShiftAssignment,
	) (*worker.WorkerShiftAssignment, error)
	UpdateAssignment(
		ctx context.Context,
		entity *worker.WorkerShiftAssignment,
	) (*worker.WorkerShiftAssignment, error)

	ListPreferences(
		ctx context.Context,
		req *ListAvailabilityPreferencesRequest,
	) ([]*worker.WorkerAvailabilityPreference, error)
	// UpsertPreference writes one weekday's preference, replacing whatever was
	// there. A worker has exactly one statement per weekday, so an insert that
	// collided would be a duplicate rather than a second opinion.
	UpsertPreference(
		ctx context.Context,
		entity *worker.WorkerAvailabilityPreference,
	) (*worker.WorkerAvailabilityPreference, error)

	ListSwaps(ctx context.Context, req *ListShiftSwapsRequest) ([]*worker.ShiftSwapRequest, error)
	GetSwapByID(
		ctx context.Context,
		req *GetShiftSwapByIDRequest,
	) (*worker.ShiftSwapRequest, error)
	CreateSwap(
		ctx context.Context,
		entity *worker.ShiftSwapRequest,
	) (*worker.ShiftSwapRequest, error)
	UpdateSwap(
		ctx context.Context,
		entity *worker.ShiftSwapRequest,
	) (*worker.ShiftSwapRequest, error)

	// The rota's four reads. Each is a grouped query over a week rather than a
	// walk: a fifty-driver rota is four queries, not two hundred.
	RotaWorkers(ctx context.Context, req *RotaQuery) ([]RotaWorkerRow, error)
	RotaTimeOffRanges(ctx context.Context, req *RotaQuery) ([]RotaRangeRow, error)
	RotaLeaveRanges(ctx context.Context, req *RotaQuery) ([]RotaRangeRow, error)
	RotaAssignedDays(ctx context.Context, req *RotaQuery) ([]RotaDayRow, error)
}
