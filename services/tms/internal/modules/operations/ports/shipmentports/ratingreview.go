package shipmentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ApproveRatingReviewRequest struct {
	ShipmentID pulid.ID              `json:"shipmentId"`
	TenantInfo pagination.TenantInfo `json:"-"`
}

type RatingReviewRepository interface {
	GetForRatingReview(
		ctx context.Context,
		req *ApproveRatingReviewRequest,
	) (*shipment.Shipment, error)
	SaveRatingReview(
		ctx context.Context,
		entity *shipment.Shipment,
	) (*shipment.Shipment, error)
}

type RatingReviewService interface {
	Approve(
		ctx context.Context,
		req *ApproveRatingReviewRequest,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
}
