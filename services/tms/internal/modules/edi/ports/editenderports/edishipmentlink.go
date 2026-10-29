package editenderports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
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
	) (*pagination.ListResult[*edisync.ShipmentLink], error)
	GetShipmentLinkByID(
		ctx context.Context,
		req GetEDIShipmentLinkByIDRequest,
	) (*edisync.ShipmentLink, error)
	GetShipmentLinksByShipmentID(
		ctx context.Context,
		req GetEDIShipmentLinksByShipmentIDRequest,
	) ([]*edisync.ShipmentLink, error)
	CreateShipmentLink(
		ctx context.Context,
		entity *edisync.ShipmentLink,
	) (*edisync.ShipmentLink, error)
}
