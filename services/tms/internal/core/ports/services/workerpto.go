package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PTOBulkActionType string

const (
	PTOBulkActionApprove = PTOBulkActionType("Approve")
	PTOBulkActionReject  = PTOBulkActionType("Reject")
	PTOBulkActionCancel  = PTOBulkActionType("Cancel")
)

func (a PTOBulkActionType) IsValid() bool {
	switch a {
	case PTOBulkActionApprove, PTOBulkActionReject, PTOBulkActionCancel:
		return true
	default:
		return false
	}
}

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

type WorkerPTOService interface {
	List(
		ctx context.Context,
		req *repositories.ListPTORequest,
	) (*pagination.CursorListResult[*worker.WorkerPTO], error)
	ListUpcoming(
		ctx context.Context,
		req *repositories.ListUpcomingPTORequest,
	) (*pagination.CursorListResult[*worker.WorkerPTO], error)
	Get(ctx context.Context, req *repositories.GetPTOByIDRequest) (*worker.WorkerPTO, error)
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
	Approve(
		ctx context.Context,
		req *repositories.UpdatePTOStatusRequest,
	) (*worker.WorkerPTO, error)
	Reject(ctx context.Context, req *repositories.UpdatePTOStatusRequest) (*worker.WorkerPTO, error)
	Cancel(ctx context.Context, req *repositories.UpdatePTOStatusRequest) (*worker.WorkerPTO, error)
	CancelRequested(
		ctx context.Context,
		req *repositories.UpdatePTOStatusRequest,
	) (*worker.WorkerPTO, error)
	BulkAction(ctx context.Context, req *PTOBulkActionRequest) (*PTOBulkActionPayload, error)
	GetChartData(
		ctx context.Context,
		req *repositories.PTOChartRequest,
	) ([]*repositories.PTOChartDataPoint, error)
}
