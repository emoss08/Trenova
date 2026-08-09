package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetRateConfirmationByIDRequest struct {
	TenantInfo         pagination.TenantInfo `json:"-"`
	RateConfirmationID pulid.ID              `json:"rateConfirmationId"`
}

type ListRateConfirmationsByMoveRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ShipmentMoveID pulid.ID              `json:"shipmentMoveId"`
}

type RateConfirmationRepository interface {
	GetByID(
		ctx context.Context,
		req *GetRateConfirmationByIDRequest,
	) (*rateconfirmation.RateConfirmation, error)
	GetActiveByAssignmentID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		assignmentID pulid.ID,
	) (*rateconfirmation.RateConfirmation, error)
	ListByMoveID(
		ctx context.Context,
		req *ListRateConfirmationsByMoveRequest,
	) ([]*rateconfirmation.RateConfirmation, error)
	MaxRevisionForAssignment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		assignmentID pulid.ID,
	) (int64, error)
	Create(
		ctx context.Context,
		entity *rateconfirmation.RateConfirmation,
	) (*rateconfirmation.RateConfirmation, error)
	Update(
		ctx context.Context,
		entity *rateconfirmation.RateConfirmation,
	) (*rateconfirmation.RateConfirmation, error)
}
