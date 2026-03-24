package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListWorkersRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type WorkerSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
	IncludeProfile     bool                           `json:"includeProfile"`
}

type GetWorkerByIDRequest struct {
	ID             pulid.ID              `json:"id"`
	TenantInfo     pagination.TenantInfo `json:"tenantInfo"`
	IncludeProfile bool                  `json:"includeProfile"`
	IncludeState   bool                  `json:"includeState"`
}

type WorkerSyncReadinessCounts struct {
	TotalWorkers        int `json:"totalWorkers"        bun:"total_workers"`
	ActiveWorkers       int `json:"activeWorkers"       bun:"active_workers"`
	SyncedActiveWorkers int `json:"syncedActiveWorkers" bun:"synced_active_workers"`
}

const (
	WorkerSyncDriftTypeMissingMapping      = "missing_mapping"
	WorkerSyncDriftTypeMissingRemoteDriver = "missing_remote_driver"
	WorkerSyncDriftTypeMappingMismatch     = "mapping_mismatch"
	WorkerSyncDriftTypeRemoteDeactivated   = "remote_deactivated"
)

type WorkerSyncDriftRecord struct {
	WorkerID        string `json:"workerId"`
	WorkerName      string `json:"workerName"`
	DriftType       string `json:"driftType"`
	Message         string `json:"message"`
	LocalExternalID string `json:"localExternalId,omitempty"`
	RemoteDriverID  string `json:"remoteDriverId,omitempty"`
	DetectedAt      int64  `json:"detectedAt"`
}

type WorkerRepository interface {
	List(
		ctx context.Context,
		req *ListWorkersRequest,
	) (*pagination.ListResult[*worker.Worker], error)
	SelectOptions(
		ctx context.Context,
		req *WorkerSelectOptionsRequest,
	) (*pagination.ListResult[*worker.Worker], error)
	GetByID(
		ctx context.Context,
		req GetWorkerByIDRequest,
	) (*worker.Worker, error)
	Create(
		ctx context.Context,
		entity *worker.Worker,
	) (*worker.Worker, error)
	Update(
		ctx context.Context,
		entity *worker.Worker,
	) (*worker.Worker, error)
	GetWorkerSyncReadinessCounts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*WorkerSyncReadinessCounts, error)
	ReplaceWorkerSyncDrifts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		drifts []WorkerSyncDriftRecord,
	) error
	ListWorkerSyncDrifts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]WorkerSyncDriftRecord, error)
}
