package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CreateSystemShipmentCommentRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
	Comment    string
	Type       shipment.CommentType
	Visibility shipment.CommentVisibility
	Priority   shipment.CommentPriority
	Metadata   map[string]any
}

type ShipmentCommentService interface {
	ListByShipmentID(
		ctx context.Context,
		req *repositories.ListShipmentCommentsRequest,
	) (*pagination.CursorListResult[*shipment.ShipmentComment], error)
	GetCountByShipmentID(
		ctx context.Context,
		req *repositories.GetShipmentCommentCountRequest,
	) (int, error)
	Create(
		ctx context.Context,
		entity *shipment.ShipmentComment,
		actor *RequestActor,
	) (*shipment.ShipmentComment, error)
	CreateSystem(
		ctx context.Context,
		req *CreateSystemShipmentCommentRequest,
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
