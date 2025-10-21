package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetCommentByIDRequest struct {
	CommentID pulid.ID `json:"commentId"      form:"commentId"`
	OrgID     pulid.ID `json:"organizationId" form:"organizationId"`
	BuID      pulid.ID `json:"businessUnitId" form:"businessUnitId"`
}

type GetShipmentCommentCountRequest struct {
	ShipmentID pulid.ID `json:"shipmentId"     form:"shipmentId"`
	OrgID      pulid.ID `json:"organizationId" form:"organizationId"`
	BuID       pulid.ID `json:"businessUnitId" form:"businessUnitId"`
}

type GetCommentsByShipmentIDRequest struct {
	Filter     *pagination.QueryOptions
	ShipmentID pulid.ID `json:"shipmentId" form:"shipmentId"`
}

type HandleCommentDeletionsRequest struct {
	ExistingCommentMap map[pulid.ID]*shipment.ShipmentComment
	UpdatedCommentIDs  map[pulid.ID]struct{}
	CommentToDelete    []*shipment.ShipmentComment
}

type DeleteCommentRequest struct {
	ShipmentID pulid.ID `json:"shipmentId"     form:"shipmentId"`
	CommentID  pulid.ID `json:"commentId"      form:"commentId"`
	OrgID      pulid.ID `json:"organizationId" form:"organizationId"`
	BuID       pulid.ID `json:"businessUnitId" form:"businessUnitId"`
	UserID     pulid.ID `json:"userId"         form:"userId"`
}

type ShipmentCommentRepository interface {
	GetByID(
		ctx context.Context,
		req GetCommentByIDRequest,
	) (*shipment.ShipmentComment, error)
	ListByShipmentID(
		ctx context.Context,
		req GetCommentsByShipmentIDRequest,
	) (*pagination.ListResult[*shipment.ShipmentComment], error)
	GetCountByShipmentID(ctx context.Context, req GetShipmentCommentCountRequest) (int, error)
	Create(
		ctx context.Context,
		comment *shipment.ShipmentComment,
	) (*shipment.ShipmentComment, error)
	Update(
		ctx context.Context,
		comment *shipment.ShipmentComment,
	) (*shipment.ShipmentComment, error)
	Delete(
		ctx context.Context,
		req *DeleteCommentRequest,
	) error
}
