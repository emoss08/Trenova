package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ListShipmentLinks(
	ctx context.Context,
	req *repositories.ListEDIShipmentLinksRequest,
) (*pagination.ListResult[*edi.ShipmentLink], error) {
	return s.shipmentLinkRepo.ListShipmentLinks(ctx, req)
}

func (s *Service) GetShipmentLink(
	ctx context.Context,
	req repositories.GetEDIShipmentLinkByIDRequest,
) (*edi.ShipmentLink, error) {
	return s.shipmentLinkRepo.GetShipmentLinkByID(ctx, req)
}
