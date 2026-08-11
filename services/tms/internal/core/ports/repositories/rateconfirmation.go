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

type MarkRateConfirmationTokenUsedRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TokenID    pulid.ID              `json:"tokenId"`
	UsedAt     int64                 `json:"usedAt"`
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
	CreateToken(
		ctx context.Context,
		entity *rateconfirmation.RateConfirmationToken,
	) (*rateconfirmation.RateConfirmationToken, error)
	GetTokenByHash(
		ctx context.Context,
		tokenHash string,
	) (*rateconfirmation.RateConfirmationToken, error)
	MarkTokenUsed(
		ctx context.Context,
		req *MarkRateConfirmationTokenUsedRequest,
	) (bool, error)
	RevokeTokensForRateConfirmation(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		rateConfirmationID pulid.ID,
		revokedAt int64,
	) error
}
