package editransferservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ListShipmentLinks(
	ctx context.Context,
	req *editenderports.ListEDIShipmentLinksRequest,
) (*pagination.ListResult[*edisync.ShipmentLink], error) {
	return s.shipmentLinkRepo.ListShipmentLinks(ctx, req)
}

func (s *Service) GetShipmentLink(
	ctx context.Context,
	req editenderports.GetEDIShipmentLinkByIDRequest,
) (*edisync.ShipmentLink, error) {
	return s.shipmentLinkRepo.GetShipmentLinkByID(ctx, req)
}
