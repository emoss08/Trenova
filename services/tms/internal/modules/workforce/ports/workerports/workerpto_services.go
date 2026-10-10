package workerports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PTOBulkActionType string

func (a PTOBulkActionType) TargetStatus() worker.PTOStatus {
	switch a {
	case PTOBulkActionApprove:
		return worker.PTOStatusApproved
	case PTOBulkActionReject:
		return worker.PTOStatusRejected
	case PTOBulkActionCancel:
		return worker.PTOStatusCancelled
	default:
		return ""
	}
}

func (a PTOBulkActionType) IsValid() bool {
	switch a {
	case PTOBulkActionApprove, PTOBulkActionReject, PTOBulkActionCancel:
		return true
	default:
		return false
	}
}

const PTOBulkActionApprove = PTOBulkActionType("Approve")

const PTOBulkActionReject = PTOBulkActionType("Reject")

const PTOBulkActionCancel = PTOBulkActionType("Cancel")

type PTOBulkActionRequest struct {
	TenantInfo pagination.TenantInfo
	PTOIDs     []pulid.ID
	Action     PTOBulkActionType
	Reason     string
	UserID     pulid.ID
}

type PTOBulkActionResult struct {
	PTOID   pulid.ID `json:"ptoId"`
	Success bool     `json:"success"`
	Error   string   `json:"error"`
}

type PTOBulkActionPayload struct {
	Results      []*PTOBulkActionResult `json:"results"`
	SuccessCount int                    `json:"successCount"`
	FailureCount int                    `json:"failureCount"`
}

// WorkerPTOTransitionPreview is a time-off request before and after a
// decision, whether the decision returns booked days to the balance, and
// what the driver would be told in Dash and by text. Driver is nil when no
// Dash notifier is configured, SMS when no text would be sent.
type WorkerPTOTransitionPreview struct {
	Before        *worker.WorkerPTO
	After         *worker.WorkerPTO
	ReturnsLedger bool
	Driver        *notificationports.DriverNotificationPreview
	SMS           *notificationports.DriverSMSPreview
	// Location is the organization's, the zone a request's days are booked
	// in, so the days a preview names are the days the ledger counts.
	Location *time.Location
}

type WorkerPTOService interface {
	List(
		ctx context.Context,
		req *ListPTORequest,
	) (*pagination.CursorListResult[*worker.WorkerPTO], error)
	ListUpcoming(
		ctx context.Context,
		req *ListUpcomingPTORequest,
	) (*pagination.CursorListResult[*worker.WorkerPTO], error)
	Get(ctx context.Context, req *GetPTOByIDRequest) (*worker.WorkerPTO, error)
	Create(
		ctx context.Context,
		entity *worker.WorkerPTO,
		userID pulid.ID,
	) (*worker.WorkerPTO, error)
	Update(
		ctx context.Context,
		entity *worker.WorkerPTO,
		userID pulid.ID,
	) (*worker.WorkerPTO, error)
	PlanCreate(
		ctx context.Context,
		entity *worker.WorkerPTO,
		userID pulid.ID,
	) (*worker.WorkerPTO, error)
	PlanUpdate(
		ctx context.Context,
		entity *worker.WorkerPTO,
	) (*kernelports.RecordChange[worker.WorkerPTO], error)
	Approve(
		ctx context.Context,
		req *UpdatePTOStatusRequest,
	) (*worker.WorkerPTO, error)
	PreviewApprove(
		ctx context.Context,
		req *UpdatePTOStatusRequest,
	) (*WorkerPTOTransitionPreview, error)
	Reject(ctx context.Context, req *UpdatePTOStatusRequest) (*worker.WorkerPTO, error)
	Cancel(ctx context.Context, req *UpdatePTOStatusRequest) (*worker.WorkerPTO, error)
	// PreviewReject and PreviewCancel check the decision as Reject and Cancel
	// do and return what it would change and send, writing nothing.
	PreviewReject(
		ctx context.Context,
		req *UpdatePTOStatusRequest,
	) (*WorkerPTOTransitionPreview, error)
	PreviewCancel(
		ctx context.Context,
		req *UpdatePTOStatusRequest,
	) (*WorkerPTOTransitionPreview, error)
	CancelRequested(
		ctx context.Context,
		req *UpdatePTOStatusRequest,
	) (*worker.WorkerPTO, error)
	BulkAction(ctx context.Context, req *PTOBulkActionRequest) (*PTOBulkActionPayload, error)
	GetChartData(
		ctx context.Context,
		req *PTOChartRequest,
	) ([]*PTOChartDataPoint, error)
}
