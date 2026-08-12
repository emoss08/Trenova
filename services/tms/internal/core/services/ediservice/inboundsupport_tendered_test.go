package ediservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func acceptedTenderOfferForTest() *tender.TenderOffer {
	partnerID := pulid.MustNew("edip_")
	return &tender.TenderOffer{
		ID:           pulid.MustNew("tof_"),
		TenderID:     pulid.MustNew("tnd_"),
		CarrierID:    pulid.MustNew("car_"),
		Channel:      tender.ChannelEDI,
		Status:       tender.OfferStatusAccepted,
		RateMethod:   shipment.CarrierRateMethodFlat,
		Rate:         decimal.NewFromInt(1500),
		EDIPartnerID: &partnerID,
		Tender: &tender.Tender{
			ID:         pulid.MustNew("tnd_"),
			ShipmentID: pulid.MustNew("shp_"),
		},
		Carrier: &carrier.Carrier{Name: "Acme Trucking"},
	}
}

func TestRecordTenderedShipmentStatus_WritesCommentAndEvent(t *testing.T) {
	t.Parallel()

	commentRepo := mocks.NewMockShipmentCommentRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	eventRepo := mocks.NewMockShipmentEventRepository(t)
	service := &Service{
		l:                   zap.NewNop(),
		shipmentCommentRepo: commentRepo,
		userRepo:            userRepo,
		shipmentEventRepo:   eventRepo,
	}
	offer := acceptedTenderOfferForTest()
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	userRepo.EXPECT().
		GetSystemUser(mock.Anything, mock.Anything).
		Return(&tenant.User{ID: pulid.MustNew("usr_")}, nil).
		Once()
	var recordedComment *shipment.ShipmentComment
	commentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
				recordedComment = comment
				return comment, nil
			},
		).
		Once()
	var recordedEvent *shipmentevent.Event
	eventRepo.EXPECT().
		Insert(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, event *shipmentevent.Event) error {
			recordedEvent = event
			return nil
		}).
		Once()

	warnings, err := service.RecordTenderedShipmentStatus(
		t.Context(),
		&RecordTenderedShipmentStatusRequest{
			TenantInfo:  tenantInfo,
			Offer:       offer,
			StatusCode:  "A7",
			ReasonCode:  "AM",
			ReferenceID: "PRO-77",
			EventAt:     1767787200,
		},
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.NotNil(t, recordedComment)
	require.Equal(t, offer.Tender.ShipmentID, recordedComment.ShipmentID)
	require.Contains(t, recordedComment.Comment, "A7")
	require.Contains(t, recordedComment.Comment, "shipment canceled")
	require.Contains(t, recordedComment.Comment, "reason AM")
	require.Equal(t, offer.ID, recordedComment.Metadata["tenderOfferId"])

	require.NotNil(t, recordedEvent)
	require.Equal(t, tenantInfo.OrgID, recordedEvent.OrganizationID)
	require.Equal(t, tenantInfo.BuID, recordedEvent.BusinessUnitID)
	require.Equal(t, offer.Tender.ShipmentID, recordedEvent.ShipmentID)
	require.Equal(t, shipmentevent.SeverityDanger, recordedEvent.Severity)
	require.Equal(t, "Acme Trucking", recordedEvent.ActorLabel)
	require.Equal(t, "Carrier reported EDI 214 status A7", recordedEvent.Summary)
	require.Equal(t, offer.TenderID.String(), recordedEvent.CorrelationID)
	require.EqualValues(t, 1767787200, recordedEvent.OccurredAt)
}

func TestRecordTenderedShipmentStatus_RequiresOfferWithTender(t *testing.T) {
	t.Parallel()

	service := &Service{l: zap.NewNop()}
	offer := acceptedTenderOfferForTest()
	offer.Tender = nil

	_, err := service.RecordTenderedShipmentStatus(t.Context(), &RecordTenderedShipmentStatusRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		Offer:      offer,
		StatusCode: "AF",
	})

	require.Error(t, err)
}

func TestRecordTenderedShipmentStatus_EventInsertFailureIsNonFatal(t *testing.T) {
	t.Parallel()

	commentRepo := mocks.NewMockShipmentCommentRepository(t)
	userRepo := mocks.NewMockUserRepository(t)
	eventRepo := mocks.NewMockShipmentEventRepository(t)
	service := &Service{
		l:                   zap.NewNop(),
		shipmentCommentRepo: commentRepo,
		userRepo:            userRepo,
		shipmentEventRepo:   eventRepo,
	}

	userRepo.EXPECT().
		GetSystemUser(mock.Anything, mock.Anything).
		Return(&tenant.User{ID: pulid.MustNew("usr_")}, nil).
		Once()
	commentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
				return comment, nil
			},
		).
		Once()
	eventRepo.EXPECT().
		Insert(mock.Anything, mock.Anything).
		Return(errors.New("event store unavailable")).
		Once()

	warnings, err := service.RecordTenderedShipmentStatus(
		t.Context(),
		&RecordTenderedShipmentStatusRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
			Offer:      acceptedTenderOfferForTest(),
			StatusCode: "AF",
		},
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
}

func TestRecordInboundFreightInvoice_AcceptedOfferStampsCarrierAndShipment(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name           string
		rateMethod     shipment.CarrierRateMethod
		expectedAmount bool
	}{
		{name: "flat rate stamps expected amount", rateMethod: shipment.CarrierRateMethodFlat, expectedAmount: true},
		{name: "per mile leaves expected amount null", rateMethod: shipment.CarrierRateMethodPerMile, expectedAmount: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			invoiceRepo := mocks.NewMockEDICarrierInvoiceRepository(t)
			service := &Service{l: zap.NewNop(), carrierInvoiceRepo: invoiceRepo}
			offer := acceptedTenderOfferForTest()
			offer.RateMethod = testCase.rateMethod
			orgID := pulid.MustNew("org_")
			buID := pulid.MustNew("bu_")
			partner := &edi.EDIPartner{
				ID:             pulid.MustNew("edip_"),
				OrganizationID: orgID,
				BusinessUnitID: buID,
			}
			message := &edi.EDIMessage{
				ID:             pulid.MustNew("edim_"),
				OrganizationID: orgID,
				BusinessUnitID: buID,
			}

			var recordedInvoice *edi.CarrierInvoice
			invoiceRepo.EXPECT().
				CreateCarrierInvoice(mock.Anything, mock.Anything).
				RunAndReturn(
					func(_ context.Context, entity *edi.CarrierInvoice) (*edi.CarrierInvoice, error) {
						recordedInvoice = entity
						return entity, nil
					},
				).
				Once()

			result, err := service.RecordInboundFreightInvoice(
				t.Context(),
				&RecordInboundFreightInvoiceRequest{
					Partner:       partner,
					Message:       message,
					AcceptedOffer: offer,
					Payload: &edi.FreightInvoicePayload{
						InvoiceNumber: "INV-1",
						BOL:           "BOL-42",
						TotalAmount: decimal.NullDecimal{
							Decimal: decimal.NewFromInt(1500),
							Valid:   true,
						},
						ReferenceNumbers: map[string]string{},
					},
				},
			)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, recordedInvoice)
			require.Equal(t, offer.Tender.ShipmentID, recordedInvoice.ShipmentID)
			require.Equal(t, offer.CarrierID, recordedInvoice.CarrierID)
			require.True(t, recordedInvoice.TenderRecipientID.IsNil())
			require.Equal(t, testCase.expectedAmount, recordedInvoice.ExpectedAmount.Valid)
			if testCase.expectedAmount {
				require.True(t, recordedInvoice.ExpectedAmount.Decimal.Equal(offer.Rate))
				require.Equal(
					t,
					edi.CarrierInvoiceReconciliationStatusMatched,
					recordedInvoice.ReconciliationStatus,
				)
			}
		})
	}
}
