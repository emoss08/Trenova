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

type UpdateShipmentCommentRequest struct {
	Entity      *shipment.ShipmentComment
	AsModerator bool
}

type DeleteShipmentCommentRequest struct {
	TenantInfo  pagination.TenantInfo
	ShipmentID  pulid.ID
	CommentID   pulid.ID
	AsModerator bool
}

type ToggleShipmentCommentRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
	CommentID  pulid.ID
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
		req *UpdateShipmentCommentRequest,
		actor *RequestActor,
	) (*shipment.ShipmentComment, error)
	Delete(
		ctx context.Context,
		req *DeleteShipmentCommentRequest,
		actor *RequestActor,
	) error
	Pin(
		ctx context.Context,
		req *ToggleShipmentCommentRequest,
		actor *RequestActor,
	) (*shipment.ShipmentComment, error)
	Unpin(
		ctx context.Context,
		req *ToggleShipmentCommentRequest,
		actor *RequestActor,
	) (*shipment.ShipmentComment, error)
	Resolve(
		ctx context.Context,
		req *ToggleShipmentCommentRequest,
		actor *RequestActor,
	) (*shipment.ShipmentComment, error)
	Unresolve(
		ctx context.Context,
		req *ToggleShipmentCommentRequest,
		actor *RequestActor,
	) (*shipment.ShipmentComment, error)
	Acknowledge(
		ctx context.Context,
		req *ToggleShipmentCommentRequest,
		actor *RequestActor,
	) (*shipment.ShipmentComment, error)
}
