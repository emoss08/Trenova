package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ShipmentCommentService interface {
	ListByShipmentID(
		ctx context.Context,
		req *repositories.ListShipmentCommentsRequest,
	) (*pagination.ListResult[*shipment.ShipmentComment], error)
	GetCountByShipmentID(
		ctx context.Context,
		req *repositories.GetShipmentCommentCountRequest,
	) (int, error)
	Create(
		ctx context.Context,
		entity *shipment.ShipmentComment,
		actor *RequestActor,
	) (*shipment.ShipmentComment, error)
	Update(
		ctx context.Context,
		entity *shipment.ShipmentComment,
		actor *RequestActor,
	) (*shipment.ShipmentComment, error)
	Delete(
		ctx context.Context,
		req *repositories.DeleteShipmentCommentRequest,
		actor *RequestActor,
	) error
}
