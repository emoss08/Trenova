package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type WorkerFilterOptions struct {
	Status         string `form:"status"`
	IncludeProfile bool   `form:"includeProfile"`
	IncludePTO     bool   `form:"includePTO"`
}

type ListWorkerRequest struct {
	Filter              *pagination.QueryOptions `json:"filter"              form:"filter"`
	WorkerFilterOptions `json:"workerFilterOptions" form:"workerFilterOptions"`
}

type ListWorkerPTORequest struct {
	Filter *pagination.QueryOptions `json:"filter" form:"filter"`
}

type GetWorkerByIDRequest struct {
	WorkerID      pulid.ID            `json:"workerId"      form:"worker_id"`
	BuID          pulid.ID            `json:"buId"          form:"bu_id"`
	OrgID         pulid.ID            `json:"orgId"         form:"org_id"`
	UserID        pulid.ID            `json:"userId"        form:"user_id"`
	FilterOptions WorkerFilterOptions `json:"filterOptions" form:"filterOptions"`
}

type UpdateWorkerOptions struct {
	OrgID pulid.ID `json:"orgId" form:"org_id"`
	BuID  pulid.ID `json:"buId"  form:"bu_id"`
}

type GetWorkerPTORequest struct {
	PtoID    pulid.ID `json:"ptoId"    form:"ptoId"`
	WorkerID pulid.ID `json:"workerId" form:"workerId"`
	BuID     pulid.ID `json:"buId"     form:"buId"`
	OrgID    pulid.ID `json:"orgId"    form:"orgId"`
}

type ListWorkerPTOFilterOptions struct {
	Status      string `json:"status"      form:"status"`
	Type        string `json:"type"        form:"type"`
	StartDate   int64  `json:"startDate"   form:"startDate"`
	EndDate     int64  `json:"endDate"     form:"endDate"`
	WorkerID    string `json:"workerId"    form:"workerId"`
	FleetCodeID string `json:"fleetCodeId" form:"fleetCodeId"`
}

type ListUpcomingWorkerPTORequest struct {
	Filter                     *pagination.QueryOptions `json:"filter"        form:"filter"`
	ListWorkerPTOFilterOptions `json:"filterOptions" form:"filterOptions"`
}

type ApprovePTORequest struct {
	PtoID      pulid.ID `json:"ptoId"      form:"ptoId"`
	BuID       pulid.ID `json:"buId"       form:"buId"`
	OrgID      pulid.ID `json:"orgId"      form:"orgId"`
	ApproverID pulid.ID `json:"approverId" form:"approverId"`
}

type RejectPTORequest struct {
	PtoID      pulid.ID `json:"ptoId"      form:"ptoId"`
	BuID       pulid.ID `json:"buId"       form:"buId"`
	OrgID      pulid.ID `json:"orgId"      form:"orgId"`
	RejectorID pulid.ID `json:"rejectorId" form:"rejectorId"`
	Reason     string   `json:"reason"     form:"reason"`
}

type PTOChartDataRequest struct {
	Filter    *pagination.QueryOptions `json:"filter"    form:"filter"`
	StartDate int64                    `json:"startDate" form:"startDate"`
	EndDate   int64                    `json:"endDate"   form:"endDate"`
	Type      string                   `json:"type"      form:"type"`
	Timezone  string                   `json:"timezone"  form:"timezone"`
	WorkerID  string                   `json:"workerId"  form:"workerId"`
}

type PTOChartDataPoint struct {
	Date        string                    `json:"date"`
	Vacation    int                       `json:"vacation"`
	Sick        int                       `json:"sick"`
	Holiday     int                       `json:"holiday"`
	Bereavement int                       `json:"bereavement"`
	Maternity   int                       `json:"maternity"`
	Paternity   int                       `json:"paternity"`
	Personal    int                       `json:"personal"`
	Workers     map[string][]WorkerDetail `json:"workers"`
}

type WorkerDetail struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	PTOType   string `json:"ptoType"`
}

type PTOCalendarDataRequest struct {
	Filter    *pagination.QueryOptions `json:"filter"    form:"filter"`
	StartDate int64                    `json:"startDate" form:"startDate"`
	EndDate   int64                    `json:"endDate"   form:"endDate"`
	Type      string                   `json:"type"      form:"type"`
}

type PTOCalendarEvent struct {
	ID         string `json:"id"               bun:"id"          form:"id"`
	WorkerID   string `json:"workerId"         bun:"worker_id"   form:"workerId"`
	WorkerName string `json:"workerName"       bun:"worker_name" form:"workerName"`
	StartDate  int64  `json:"startDate"        bun:"start_date"  form:"startDate"`
	EndDate    int64  `json:"endDate"          bun:"end_date"    form:"endDate"`
	Type       string `json:"type"             bun:"type"        form:"type"`
	Status     string `json:"status"           bun:"status"      form:"status"`
	Reason     string `json:"reason,omitempty" bun:"reason"      form:"reason"`
}

type WorkerRepository interface {
	List(
		ctx context.Context,
		req *ListWorkerRequest,
	) (*pagination.ListResult[*worker.Worker], error)
	GetByID(ctx context.Context, req *GetWorkerByIDRequest) (*worker.Worker, error)
	Create(ctx context.Context, wrk *worker.Worker) (*worker.Worker, error)
	Update(ctx context.Context, wrk *worker.Worker) (*worker.Worker, error)
	GetWorkerPTO(
		ctx context.Context,
		req *GetWorkerPTORequest,
	) (*worker.WorkerPTO, error)
	ListUpcomingPTO(
		ctx context.Context,
		req *ListUpcomingWorkerPTORequest,
	) (*pagination.ListResult[*worker.WorkerPTO], error)
	ApprovePTO(ctx context.Context, req *ApprovePTORequest) error
	RejectPTO(ctx context.Context, req *RejectPTORequest) error
	ListWorkerPTO(
		ctx context.Context,
		req *ListWorkerPTORequest,
	) (*pagination.ListResult[*worker.WorkerPTO], error)
	GetPTOChartData(
		ctx context.Context,
		req *PTOChartDataRequest,
	) ([]*PTOChartDataPoint, error)
	GetPTOCalendarData(
		ctx context.Context,
		req *PTOCalendarDataRequest,
	) ([]*PTOCalendarEvent, error)
	CreateWorkerPTO(
		ctx context.Context,
		pto *worker.WorkerPTO,
	) (*worker.WorkerPTO, error)
}
