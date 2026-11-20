package edipayloadbuild

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) PurchaseOrder(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) (string, error) {
	if orderID.IsNil() {
		return "", nil
	}
	entity, err := s.orderRepo.GetByID(ctx, orderports.GetOrderByIDRequest{
		ID:         orderID,
		TenantInfo: tenantInfo,
	})
	if errortypes.IsNotFoundError(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read the load's order: %w", err)
	}
	return strings.TrimSpace(entity.PONumber), nil
}

func (s *Service) ShipmentPurchaseOrder(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (string, error) {
	if shipmentID.IsNil() {
		return "", nil
	}
	source, err := s.shipmentSvc.Get(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
	})
	if errortypes.IsNotFoundError(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read the load's shipment: %w", err)
	}
	return s.PurchaseOrder(ctx, tenantInfo, source.OrderID)
}

func (s *Service) TenderPayload(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	source *shipment.Shipment,
) (editransfer.LoadTenderPayload, error) {
	payload := BuildTenderPayload(source)
	poNumber, err := s.PurchaseOrder(ctx, tenantInfo, source.OrderID)
	if err != nil {
		return editransfer.LoadTenderPayload{}, err
	}
	payload.PONumber = poNumber
	return payload, nil
}

func (s *Service) FreightInvoicePayload(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	source *invoice.Invoice,
) (edipayload.DocumentPayload, error) {
	payload := BuildFreightInvoicePayload(source)
	shipments := payload.FreightInvoice.Shipments
	if len(shipments) == 0 {
		return payload, nil
	}
	shipmentIDs := make([]pulid.ID, 0, len(shipments))
	for index := range shipments {
		shipmentIDs = append(shipmentIDs, shipments[index].ShipmentID)
	}
	summaries, err := s.shipmentRepo.ListSummariesByIDs(
		ctx,
		&shipmentports.ListShipmentSummariesRequest{
			TenantInfo:  tenantInfo,
			ShipmentIDs: shipmentIDs,
		},
	)
	if err != nil {
		return edipayload.DocumentPayload{}, fmt.Errorf("read the invoiced shipments: %w", err)
	}
	applyShipmentSummaries(shipments, summaries)
	return payload, nil
}

func applyShipmentSummaries(
	shipments []edipayload.FreightInvoiceShipment,
	summaries []*shipmentports.ShipmentSummary,
) {
	byID := make(map[pulid.ID]*shipmentports.ShipmentSummary, len(summaries))
	for _, summary := range summaries {
		if summary != nil {
			byID[summary.ShipmentID] = summary
		}
	}
	for index := range shipments {
		summary, ok := byID[shipments[index].ShipmentID]
		if !ok {
			continue
		}
		shipments[index].PONumber = strings.TrimSpace(summary.PONumber)
		if summary.ServiceDate != nil {
			shipments[index].DeliveryDate = *summary.ServiceDate
		}
	}
}
