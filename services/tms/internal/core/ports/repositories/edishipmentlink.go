package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListEDIShipmentLinksRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetEDIShipmentLinkByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetEDIShipmentLinksByShipmentIDRequest struct {
	ShipmentID pulid.ID              `json:"shipmentId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type EDIShipmentLinkRepository interface {
	ListShipmentLinks(
		ctx context.Context,
		req *ListEDIShipmentLinksRequest,
	) (*pagination.ListResult[*edi.ShipmentLink], error)
	GetShipmentLinkByID(
		ctx context.Context,
		req GetEDIShipmentLinkByIDRequest,
	) (*edi.ShipmentLink, error)
	GetShipmentLinksByShipmentID(
		ctx context.Context,
		req GetEDIShipmentLinksByShipmentIDRequest,
	) ([]*edi.ShipmentLink, error)
	CreateShipmentLink(ctx context.Context, entity *edi.ShipmentLink) (*edi.ShipmentLink, error)
}
