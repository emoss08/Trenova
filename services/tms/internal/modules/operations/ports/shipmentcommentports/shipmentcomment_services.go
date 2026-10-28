package shipmentcommentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
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

type ShipmentCommentChange struct {
	Before *shipment.ShipmentComment
	After  *shipment.ShipmentComment
}

type ShipmentCommentDeletion struct {
	Comment    *shipment.ShipmentComment
	Tombstoned bool
}

type ShipmentCommentService interface {
	GetByID(
		ctx context.Context,
		req *GetShipmentCommentByIDRequest,
	) (*shipment.ShipmentComment, error)
	ListByShipmentID(
		ctx context.Context,
		req *ListShipmentCommentsRequest,
	) (*pagination.CursorListResult[*shipment.ShipmentComment], error)
	GetCountByShipmentID(
		ctx context.Context,
		req *GetShipmentCommentCountRequest,
	) (int, error)
	Create(
		ctx context.Context,
		entity *shipment.ShipmentComment,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentComment, error)
	CreateSystem(
		ctx context.Context,
		req *CreateSystemShipmentCommentRequest,
	) (*shipment.ShipmentComment, error)
	Update(
		ctx context.Context,
		req *UpdateShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Delete(
		ctx context.Context,
		req *DeleteShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) error
	Pin(
		ctx context.Context,
		req *ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Unpin(
		ctx context.Context,
		req *ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Resolve(
		ctx context.Context,
		req *ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Unresolve(
		ctx context.Context,
		req *ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentComment, error)
	Acknowledge(
		ctx context.Context,
		req *ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentComment, error)
	PreviewPin(
		ctx context.Context,
		req *ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
		pinned bool,
	) (*ShipmentCommentChange, error)
	PreviewResolve(
		ctx context.Context,
		req *ToggleShipmentCommentRequest,
		actor *permissionports.RequestActor,
		resolved bool,
	) (*ShipmentCommentChange, error)
	PreviewUpdate(
		ctx context.Context,
		req *UpdateShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*ShipmentCommentChange, error)
	PreviewDelete(
		ctx context.Context,
		req *DeleteShipmentCommentRequest,
		actor *permissionports.RequestActor,
	) (*ShipmentCommentDeletion, error)
}
