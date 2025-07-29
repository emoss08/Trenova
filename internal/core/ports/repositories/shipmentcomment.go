/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetCommentByIDRequest struct {
	CommentID pulid.ID `json:"commentId"      query:"commentId"`
	OrgID     pulid.ID `json:"organizationId" query:"organizationId"`
	BuID      pulid.ID `json:"businessUnitId" query:"businessUnitId"`
}

type GetCommentsByShipmentIDRequest struct {
	Filter     *ports.QueryOptions
	ShipmentID pulid.ID `json:"shipmentId" query:"shipmentId"`
}

type HandleCommentDeletionsRequest struct {
	ExistingCommentMap map[pulid.ID]*shipment.ShipmentComment
	UpdatedCommentIDs  map[pulid.ID]struct{}
	CommentToDelete    []*shipment.ShipmentComment
}

type ShipmentCommentRepository interface {
	GetByID(ctx context.Context, req GetCommentByIDRequest) (*shipment.ShipmentComment, error)
	ListByShipmentID(
		ctx context.Context,
		req GetCommentsByShipmentIDRequest,
	) (*ports.ListResult[*shipment.ShipmentComment], error)
	Create(
		ctx context.Context,
		comment *shipment.ShipmentComment,
	) (*shipment.ShipmentComment, error)
}
