package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListPTORequest struct {
	Filter        *pagination.QueryOptions `json:"filter"`
	Status        string                   `json:"status"`        // Filter by PTO status (Requested, Approved, Rejected, Cancelled)
	Type          string                   `json:"type"`          // Filter by PTO type (Vacation, Sick, etc.)
	StartDateFrom int64                    `json:"startDateFrom"` // Filter PTO starting from this date
	StartDateTo   int64                    `json:"startDateTo"`   // Filter PTO starting up to this date
	WorkerID      pulid.ID                 `json:"workerId"`      // Filter by specific worker
	IncludeWorker bool                     `json:"includeWorker"` // Include worker details in response
}

type GetPTOByIDRequest struct {
	ID            pulid.ID              `json:"id"`
	TenantInfo    pagination.TenantInfo `json:"tenantInfo"`
	IncludeWorker bool                  `json:"includeWorker"`
}

type ListWorkerPTOFilterOptions struct {
	Status      string `json:"status"      form:"status"`
	Type        string `json:"type"        form:"type"`
	StartDate   int64  `json:"startDate"   form:"startDate"`
	EndDate     int64  `json:"endDate"     form:"endDate"`
	WorkerID    string `json:"workerId"    form:"workerId"`
	FleetCodeID string `json:"fleetCodeId" form:"fleetCodeId"`
	Timezone    string `json:"timezone"    form:"timezone"`
}
type ListUpcomingPTORequest struct {
	Filter                     *pagination.QueryOptions `json:"filter"`
	ListWorkerPTOFilterOptions `json:"filterOptions"`
}

type UpdatePTOStatusRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Status     worker.PTOStatus      `json:"status"`
	UserID     pulid.ID              `json:"userId"` // User approving/rejecting
}

type PTOChartWorker struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	PTOType   string `json:"ptoType"`
}

type PTOChartDataPoint struct {
	Date        string                      `json:"date"`
	Vacation    int                         `json:"vacation"`
	Sick        int                         `json:"sick"`
	Holiday     int                         `json:"holiday"`
	Bereavement int                         `json:"bereavement"`
	Maternity   int                         `json:"maternity"`
	Paternity   int                         `json:"paternity"`
	Personal    int                         `json:"personal"`
	Workers     map[string][]PTOChartWorker `json:"workers"`
}

type PTOChartRequest struct {
	Filter        *pagination.QueryOptions `json:"filter"`
	StartDateFrom int64                    `json:"startDateFrom"`
	StartDateTo   int64                    `json:"startDateTo"`
	Type          string                   `json:"type"`
	WorkerID      string                   `json:"workerId"`
	Timezone      string                   `json:"timezone"`
}

type WorkerPTORepository interface {
	List(
		ctx context.Context,
		req *ListPTORequest,
	) (*pagination.ListResult[*worker.WorkerPTO], error)
	ListUpcoming(
		ctx context.Context,
		req *ListUpcomingPTORequest,
	) (*pagination.ListResult[*worker.WorkerPTO], error)
	GetByID(
		ctx context.Context,
		req *GetPTOByIDRequest,
	) (*worker.WorkerPTO, error)
	Create(
		ctx context.Context,
		entity *worker.WorkerPTO,
	) (*worker.WorkerPTO, error)
	UpdateStatus(
		ctx context.Context,
		req *UpdatePTOStatusRequest,
	) (*worker.WorkerPTO, error)
	GetChartData(
		ctx context.Context,
		req *PTOChartRequest,
	) ([]*PTOChartDataPoint, error)
}
