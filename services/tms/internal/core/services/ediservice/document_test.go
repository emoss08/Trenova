package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResolvePayloadInvoiceIDResolves210Payload(t *testing.T) {
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	invoiceID := pulid.MustNew("inv_")
	shipmentID := pulid.MustNew("sp_")
	invoiceRepo := mocks.NewMockInvoiceRepository(t)
	invoiceRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetInvoiceByIDRequest{
			ID:         invoiceID,
			TenantInfo: tenantInfo,
		}).
		Return(&invoice.Invoice{
			ID:           invoiceID,
			ShipmentID:   shipmentID,
			Number:       "INV-210",
			CurrencyCode: "USD",
			TotalAmount:  decimal.NewFromInt(100),
			Lines: []*invoice.InoviceLine{
				{LineNumber: 1, Type: invoice.InvoiceLineTypeFreight, Amount: decimal.NewFromInt(100)},
			},
		}, nil).
		Once()

	svc := &Service{invoiceRepo: invoiceRepo}
	payload, err := svc.resolvePayload(t.Context(), &PreviewEDIDocumentRequest{
		TenantInfo: tenantInfo,
		InvoiceID:  invoiceID,
	}, &edi.EDIPartnerDocumentProfile{TransactionSet: edi.TransactionSet210})

	require.NoError(t, err)
	require.NotNil(t, payload.FreightInvoice)
	require.Equal(t, edi.TransactionSet210, payload.TransactionSet)
	require.Equal(t, invoiceID, payload.FreightInvoice.InvoiceID)
	require.Equal(t, shipmentID, payload.FreightInvoice.ShipmentID)
}

func TestResolvePayloadShipmentEventIDResolves214Payload(t *testing.T) {
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	eventID := pulid.MustNew("se_")
	shipmentID := pulid.MustNew("sp_")
	eventRepo := mocks.NewMockShipmentEventRepository(t)
	shipmentSvc := mocks.NewMockShipmentService(t)
	eventRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetShipmentEventByIDRequest{
			ID:         eventID,
			TenantInfo: tenantInfo,
		}).
		Return(&shipmentevent.Event{
			ID:         eventID,
			ShipmentID: shipmentID,
			Type:       shipmentevent.TypeMoveDeparted,
			OccurredAt: 1715817600,
		}, nil).
		Once()
	shipmentSvc.EXPECT().
		Get(mock.Anything, mock.MatchedBy(func(req *repositories.GetShipmentByIDRequest) bool {
			return req.ID == shipmentID && req.ExpandShipmentDetails
		})).
		Return(&shipment.Shipment{ID: shipmentID, BOL: "BOL-214"}, nil).
		Once()

	svc := &Service{shipmentEventRepo: eventRepo, shipmentSvc: shipmentSvc}
	payload, err := svc.resolvePayload(t.Context(), &PreviewEDIDocumentRequest{
		TenantInfo:      tenantInfo,
		ShipmentEventID: eventID,
	}, &edi.EDIPartnerDocumentProfile{TransactionSet: edi.TransactionSet214})

	require.NoError(t, err)
	require.NotNil(t, payload.ShipmentStatus)
	require.Equal(t, edi.TransactionSet214, payload.TransactionSet)
	require.Equal(t, shipmentID, payload.ShipmentStatus.ShipmentID)
	require.Equal(t, "AF", payload.ShipmentStatus.StatusCode)
}

func TestResolvePayloadRejectsSourceTransactionMismatch(t *testing.T) {
	svc := &Service{}
	_, err := svc.resolvePayload(t.Context(), &PreviewEDIDocumentRequest{
		InvoiceID: pulid.MustNew("inv_"),
	}, &edi.EDIPartnerDocumentProfile{TransactionSet: edi.TransactionSet204})

	require.Error(t, err)
	require.Contains(t, err.Error(), "invoice")
	require.Contains(t, err.Error(), "210")
}

func TestResolvePayloadRejectsShipmentEventMismatch(t *testing.T) {
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	eventID := pulid.MustNew("se_")
	eventShipmentID := pulid.MustNew("sp_")
	requestShipmentID := pulid.MustNew("sp_")
	eventRepo := mocks.NewMockShipmentEventRepository(t)
	eventRepo.EXPECT().
		GetByID(mock.Anything, repositories.GetShipmentEventByIDRequest{
			ID:         eventID,
			TenantInfo: tenantInfo,
		}).
		Return(&shipmentevent.Event{
			ID:         eventID,
			ShipmentID: eventShipmentID,
			Type:       shipmentevent.TypeMoveDeparted,
		}, nil).
		Once()

	svc := &Service{shipmentEventRepo: eventRepo}
	_, err := svc.resolvePayload(t.Context(), &PreviewEDIDocumentRequest{
		TenantInfo:      tenantInfo,
		ShipmentID:      requestShipmentID,
		ShipmentEventID: eventID,
	}, &edi.EDIPartnerDocumentProfile{TransactionSet: edi.TransactionSet214})

	require.Error(t, err)
	require.Contains(t, err.Error(), "Shipment ID must match")
}
