package edioutbound

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/edi/app/edialert"
	"github.com/emoss08/trenova/internal/modules/edi/app/edidelivery"
	"github.com/emoss08/trenova/internal/modules/edi/app/edipayloadbuild"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediprofilesecret"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderrecipient"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmenteventportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentserviceportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestResolvePayloadInvoiceIDResolves210Payload(t *testing.T) {
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	invoiceID := pulid.MustNew("inv_")
	shipmentID := pulid.MustNew("sp_")
	invoiceRepo := invoiceportsmock.NewMockInvoiceRepository(t)
	invoiceRepo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{
			ID:         invoiceID,
			TenantInfo: tenantInfo,
		}).
		Return(&invoice.Invoice{
			ID:           invoiceID,
			ShipmentID:   shipmentID,
			Number:       "INV-210",
			CurrencyCode: "USD",
			TotalAmount:  decimal.NewFromInt(100),
			Lines: []*invoice.InvoiceLine{
				{
					LineNumber: 1,
					Type:       invoice.InvoiceLineTypeFreight,
					Amount:     decimal.NewFromInt(100),
				},
			},
		}, nil).
		Once()

	svc := &Service{
		invoiceRepo: invoiceRepo,
		alert:       &edialert.Service{},
		delivery: edidelivery.New(edidelivery.Params{
			Logger:      zap.NewNop(),
			InvoiceRepo: invoiceRepo,
			Alert:       &edialert.Service{},
			Recipient:   &editenderrecipient.Service{},
			Secret:      &ediprofilesecret.Service{},
		}),
		payload:   &edipayloadbuild.Service{},
		recipient: editenderrecipient.New(editenderrecipient.Params{}),
	}
	payload, err := svc.resolvePayload(t.Context(), &PreviewEDIDocumentRequest{
		TenantInfo: tenantInfo,
		InvoiceID:  invoiceID,
	}, edidocument.TransactionSet210)

	require.NoError(t, err)
	require.NotNil(t, payload.FreightInvoice)
	require.Equal(t, edidocument.TransactionSet210, payload.TransactionSet)
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
	eventRepo := shipmenteventportsmock.NewMockShipmentEventRepository(t)
	shipmentSvc := shipmentserviceportsmock.NewMockShipmentReader(t)
	eventRepo.EXPECT().
		GetByID(mock.Anything, shipmenteventports.GetShipmentEventByIDRequest{
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
		Get(mock.Anything, mock.MatchedBy(func(req *shipmentports.GetShipmentByIDRequest) bool {
			return req.ID == shipmentID && req.ExpandShipmentDetails
		})).
		Return(&shipment.Shipment{ID: shipmentID, BOL: "BOL-214"}, nil).
		Once()

	svc := &Service{
		shipmentEventRepo: eventRepo,
		shipmentSvc:       shipmentSvc,
		alert:             &edialert.Service{},
		delivery: edidelivery.New(edidelivery.Params{
			Logger:    zap.NewNop(),
			Alert:     &edialert.Service{},
			Recipient: editenderrecipient.New(editenderrecipient.Params{}),
			Secret:    &ediprofilesecret.Service{},
		}),
		payload: edipayloadbuild.New(edipayloadbuild.Params{
			ShipmentSvc: shipmentSvc,
		}),
		recipient: editenderrecipient.New(editenderrecipient.Params{}),
	}
	payload, err := svc.resolvePayload(t.Context(), &PreviewEDIDocumentRequest{
		TenantInfo:      tenantInfo,
		ShipmentEventID: eventID,
	}, edidocument.TransactionSet214)

	require.NoError(t, err)
	require.NotNil(t, payload.ShipmentStatus)
	require.Equal(t, edidocument.TransactionSet214, payload.TransactionSet)
	require.Equal(t, shipmentID, payload.ShipmentStatus.ShipmentID)
	require.Equal(t, "AF", payload.ShipmentStatus.StatusCode)
}

func TestResolvePayloadRejectsSourceTransactionMismatch(t *testing.T) {
	svc := &Service{
		alert: edialert.New(edialert.Params{
			Logger: zap.NewNop(),
		}),
		delivery: edidelivery.New(edidelivery.Params{
			Logger:    zap.NewNop(),
			Alert:     &edialert.Service{},
			Recipient: editenderrecipient.New(editenderrecipient.Params{}),
			Secret:    &ediprofilesecret.Service{},
		}),
		payload:   &edipayloadbuild.Service{},
		recipient: editenderrecipient.New(editenderrecipient.Params{}),
	}
	_, err := svc.resolvePayload(t.Context(), &PreviewEDIDocumentRequest{
		InvoiceID: pulid.MustNew("inv_"),
	}, edidocument.TransactionSet204)

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
	eventRepo := shipmenteventportsmock.NewMockShipmentEventRepository(t)
	eventRepo.EXPECT().
		GetByID(mock.Anything, shipmenteventports.GetShipmentEventByIDRequest{
			ID:         eventID,
			TenantInfo: tenantInfo,
		}).
		Return(&shipmentevent.Event{
			ID:         eventID,
			ShipmentID: eventShipmentID,
			Type:       shipmentevent.TypeMoveDeparted,
		}, nil).
		Once()

	svc := &Service{
		shipmentEventRepo: eventRepo,
		alert:             &edialert.Service{},
		delivery: edidelivery.New(edidelivery.Params{
			Logger:    zap.NewNop(),
			Alert:     &edialert.Service{},
			Recipient: editenderrecipient.New(editenderrecipient.Params{}),
			Secret:    &ediprofilesecret.Service{},
		}),
		payload:   &edipayloadbuild.Service{},
		recipient: editenderrecipient.New(editenderrecipient.Params{}),
	}
	_, err := svc.resolvePayload(t.Context(), &PreviewEDIDocumentRequest{
		TenantInfo:      tenantInfo,
		ShipmentID:      requestShipmentID,
		ShipmentEventID: eventID,
	}, edidocument.TransactionSet214)

	require.Error(t, err)
	require.Contains(t, err.Error(), "Shipment ID must match")
}
