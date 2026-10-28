package workerdotports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdot"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListWorkerDOTTestsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	// WorkerID scopes the list to one worker; leave it nil for the whole
	// organisation, which is what the testing register reads.
	WorkerID  pulid.ID                  `json:"workerId"`
	TestTypes []workerdot.DOTTestType   `json:"testTypes"`
	Statuses  []workerdot.DOTTestStatus `json:"statuses"`
	// Since limits the list to collections on or after this instant; zero
	// means all.
	Since           int64 `json:"since"`
	OpenOnly        bool  `json:"openOnly"`
	IncludeWorker   bool  `json:"includeWorker"`
	IncludeDocument bool  `json:"includeDocument"`
	IncludeActors   bool  `json:"includeActors"`
	Limit           int   `json:"limit"`
}

type GetWorkerDOTTestByIDRequest struct {
	ID              pulid.ID              `json:"id"`
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	IncludeWorker   bool                  `json:"includeWorker"`
	IncludeDocument bool                  `json:"includeDocument"`
	IncludeActors   bool                  `json:"includeActors"`
}

type ListWorkerDOTViolationsRequest struct {
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	WorkerID       pulid.ID              `json:"workerId"`
	UnresolvedOnly bool                  `json:"unresolvedOnly"`
	IncludeTests   bool                  `json:"includeTests"`
	IncludeWorker  bool                  `json:"includeWorker"`
}

type GetWorkerDOTViolationByIDRequest struct {
	ID           pulid.ID              `json:"id"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	IncludeTests bool                  `json:"includeTests"`
}

type ListClearinghouseQueriesRequest struct {
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	WorkerID      pulid.ID              `json:"workerId"`
	PendingOnly   bool                  `json:"pendingOnly"`
	IncludeWorker bool                  `json:"includeWorker"`
}

type GetClearinghouseQueryByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListDOTRandomPoolsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Cursor pagination.CursorInfo    `json:"cursor"`
	Status string                   `json:"status"`
}

type GetDOTRandomPoolByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type DOTRandomPoolCodeExistsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Code       string                `json:"code"`
	ExcludeID  pulid.ID              `json:"excludeId"`
}

type ListDOTRandomDrawsRequest struct {
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	PoolID        pulid.ID              `json:"poolId"`
	IncludePool   bool                  `json:"includePool"`
	IncludeActors bool                  `json:"includeActors"`
	Limit         int                   `json:"limit"`
}

type GetDOTRandomDrawByIDRequest struct {
	ID             pulid.ID              `json:"id"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	IncludeEntries bool                  `json:"includeEntries"`
	IncludeWorkers bool                  `json:"includeWorkers"`
	IncludePool    bool                  `json:"includePool"`
}

type GetDOTRandomDrawByPeriodRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	PoolID     pulid.ID              `json:"poolId"`
	PeriodKey  string                `json:"periodKey"`
}

// ListPoolCandidatesRequest names the workers eligible for a draw: employed
// drivers whose type the pool covers. A terminated driver cannot be selected,
// and a driver already prohibited is not put back in the hat.
type ListPoolCandidatesRequest struct {
	TenantInfo  pagination.TenantInfo `json:"tenantInfo"`
	DriverTypes []string              `json:"driverTypes"`
}

type ListDOTRandomDrawEntriesRequest struct {
	TenantInfo      pagination.TenantInfo `json:"tenantInfo"`
	DrawID          pulid.ID              `json:"drawId"`
	WorkerID        pulid.ID              `json:"workerId"`
	OutstandingOnly bool                  `json:"outstandingOnly"`
	IncludeWorker   bool                  `json:"includeWorker"`
	IncludeDraw     bool                  `json:"includeDraw"`
}

type GetDOTRandomDrawEntryByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

// ListWorkersWithClearinghouseDueRequest finds the drivers whose annual limited
// query falls due inside the window, which is the only set the nightly sweep
// needs to look at.
type ListWorkersWithClearinghouseDueRequest struct {
	Until int64 `json:"until"`
	Limit int   `json:"limit"`
}

// DrugAlcoholRollup is the denormalised standing written back to the profile.
type DrugAlcoholRollup struct {
	Status                    worker.DrugAlcoholStatus
	ReturnToDuty              worker.ReturnToDutyStatus
	LastClearinghouseQueryAt  *int64
	NextClearinghouseQueryDue *int64
}

type UpdateDrugAlcoholRollupRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	Rollup     DrugAlcoholRollup     `json:"rollup"`
}

// WorkerDrugAlcoholRepository owns every table in the testing programme. They
// travel together because the standing on a worker is derived from all of them
// at once, and splitting them would mean four repositories reading each other.
type WorkerDrugAlcoholRepository interface {
	ListTests(
		ctx context.Context,
		req *ListWorkerDOTTestsRequest,
	) ([]*workerdot.WorkerDOTTest, error)
	GetTestByID(
		ctx context.Context,
		req *GetWorkerDOTTestByIDRequest,
	) (*workerdot.WorkerDOTTest, error)
	CreateTest(
		ctx context.Context,
		entity *workerdot.WorkerDOTTest,
	) (*workerdot.WorkerDOTTest, error)
	UpdateTest(
		ctx context.Context,
		entity *workerdot.WorkerDOTTest,
	) (*workerdot.WorkerDOTTest, error)

	ListViolations(
		ctx context.Context,
		req *ListWorkerDOTViolationsRequest,
	) ([]*workerdot.WorkerDOTViolation, error)
	GetViolationByID(
		ctx context.Context,
		req *GetWorkerDOTViolationByIDRequest,
	) (*workerdot.WorkerDOTViolation, error)
	GetOpenViolation(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) (*workerdot.WorkerDOTViolation, error)
	CreateViolation(
		ctx context.Context,
		entity *workerdot.WorkerDOTViolation,
	) (*workerdot.WorkerDOTViolation, error)
	UpdateViolation(
		ctx context.Context,
		entity *workerdot.WorkerDOTViolation,
	) (*workerdot.WorkerDOTViolation, error)

	ListQueries(
		ctx context.Context,
		req *ListClearinghouseQueriesRequest,
	) ([]*workerdot.WorkerClearinghouseQuery, error)
	GetQueryByID(
		ctx context.Context,
		req *GetClearinghouseQueryByIDRequest,
	) (*workerdot.WorkerClearinghouseQuery, error)
	CreateQuery(
		ctx context.Context,
		entity *workerdot.WorkerClearinghouseQuery,
	) (*workerdot.WorkerClearinghouseQuery, error)
	UpdateQuery(
		ctx context.Context,
		entity *workerdot.WorkerClearinghouseQuery,
	) (*workerdot.WorkerClearinghouseQuery, error)

	ListPools(
		ctx context.Context,
		req *ListDOTRandomPoolsRequest,
	) (*pagination.CursorListResult[*workerdot.DOTRandomPool], error)
	GetPoolByID(
		ctx context.Context,
		req *GetDOTRandomPoolByIDRequest,
	) (*workerdot.DOTRandomPool, error)
	GetDefaultPool(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*workerdot.DOTRandomPool, error)
	PoolCodeExists(ctx context.Context, req *DOTRandomPoolCodeExistsRequest) (bool, error)
	CreatePool(
		ctx context.Context,
		entity *workerdot.DOTRandomPool,
	) (*workerdot.DOTRandomPool, error)
	UpdatePool(
		ctx context.Context,
		entity *workerdot.DOTRandomPool,
	) (*workerdot.DOTRandomPool, error)
	ClearDefaultPool(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		exceptID pulid.ID,
	) error

	ListDraws(
		ctx context.Context,
		req *ListDOTRandomDrawsRequest,
	) ([]*workerdot.DOTRandomDraw, error)
	GetDrawByID(
		ctx context.Context,
		req *GetDOTRandomDrawByIDRequest,
	) (*workerdot.DOTRandomDraw, error)
	GetDrawByPeriod(
		ctx context.Context,
		req *GetDOTRandomDrawByPeriodRequest,
	) (*workerdot.DOTRandomDraw, error)
	// CreateDrawWithEntries writes the round and the names it produced in one
	// transaction: a draw whose entries failed to land would be evidence of a
	// selection that never happened.
	CreateDrawWithEntries(
		ctx context.Context,
		draw *workerdot.DOTRandomDraw,
		entries []*workerdot.DOTRandomDrawEntry,
	) (*workerdot.DOTRandomDraw, error)
	UpdateDraw(
		ctx context.Context,
		entity *workerdot.DOTRandomDraw,
	) (*workerdot.DOTRandomDraw, error)
	ListPoolCandidates(ctx context.Context, req *ListPoolCandidatesRequest) ([]pulid.ID, error)

	ListDrawEntries(
		ctx context.Context,
		req *ListDOTRandomDrawEntriesRequest,
	) ([]*workerdot.DOTRandomDrawEntry, error)
	GetDrawEntryByID(
		ctx context.Context,
		req *GetDOTRandomDrawEntryByIDRequest,
	) (*workerdot.DOTRandomDrawEntry, error)
	UpdateDrawEntry(
		ctx context.Context,
		entity *workerdot.DOTRandomDrawEntry,
	) (*workerdot.DOTRandomDrawEntry, error)

	UpdateProfileRollup(ctx context.Context, req *UpdateDrugAlcoholRollupRequest) error
	ListWorkersWithClearinghouseDue(
		ctx context.Context,
		req *ListWorkersWithClearinghouseDueRequest,
	) ([]workerports.WorkerTenantRef, error)
}
