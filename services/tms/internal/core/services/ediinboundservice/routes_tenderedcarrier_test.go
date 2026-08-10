package ediinboundservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type tenderedCarrierFixture struct {
	service       *Service
	file          *edi.EDIInboundFile
	partner       *edi.EDIPartner
	tenderRepo    *fakeTenderRepository
	recipientRepo *mocks.MockEDITenderRecipientRepository
	commentRepo   *mocks.MockShipmentCommentRepository
	userRepo      *mocks.MockUserRepository
	eventRepo     *mocks.MockShipmentEventRepository
	invoiceRepo   *mocks.MockEDICarrierInvoiceRepository
}

func newTenderedCarrierFixture(t *testing.T) *tenderedCarrierFixture {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	partnerID := pulid.MustNew("edip_")
	fixture := &tenderedCarrierFixture{
		file: &edi.EDIInboundFile{
			ID:             pulid.MustNew("ediinf_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			EDIPartnerID:   partnerID,
		},
		partner: &edi.EDIPartner{
			ID:                partnerID,
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			Kind:              edi.PartnerKindExternal,
			Code:              "CARRIER",
			EnabledForInbound: true,
		},
		tenderRepo:    &fakeTenderRepository{},
		recipientRepo: mocks.NewMockEDITenderRecipientRepository(t),
		commentRepo:   mocks.NewMockShipmentCommentRepository(t),
		userRepo:      mocks.NewMockUserRepository(t),
		eventRepo:     mocks.NewMockShipmentEventRepository(t),
		invoiceRepo:   mocks.NewMockEDICarrierInvoiceRepository(t),
	}
	ediSvc := ediservice.New(ediservice.Params{
		Logger:              zap.NewNop(),
		ShipmentCommentRepo: fixture.commentRepo,
		UserRepo:            fixture.userRepo,
		ShipmentEventRepo:   fixture.eventRepo,
		CarrierInvoiceRepo:  fixture.invoiceRepo,
	})
	fixture.service = &Service{
		l:                   zap.NewNop(),
		metrics:             metrics.NewEDI(nil, zap.NewNop(), false),
		tenderRecipientRepo: fixture.recipientRepo,
		tenderRepo:          fixture.tenderRepo,
		ediService:          ediSvc,
	}
	return fixture
}

func (f *tenderedCarrierFixture) acceptedOffer() *tender.TenderOffer {
	partnerID := f.partner.ID
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

func (f *tenderedCarrierFixture) expectSystemUser() {
	f.userRepo.EXPECT().
		GetSystemUser(mock.Anything, mock.Anything).
		Return(&tenant.User{ID: pulid.MustNew("usr_")}, nil)
}

func rawShipmentStatus214(offerRef, statusCode string) string {
	raw := "ISA*00*          *00*          *ZZ*CARRIER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000301*0*P*>~" +
		"GS*QM*CARRIER*TRENOVA*20260107*1200*301*X*004010~" +
		"ST*214*0001~" +
		"B10*PRO-77*SHIP-1001*SCAC~"
	if offerRef != "" {
		raw += "L11*" + offerRef + "*CR~"
	}
	raw += "AT7*" + statusCode + "*NS***20260107*1200~" +
		"SE*6*0001~GE*1*301~IEA*1*000000301~"
	return raw
}

func parse214Transaction(t *testing.T, raw string) *parsedTransaction {
	t.Helper()
	interchange, err := parseInterchange(raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	require.Equal(t, edi.TransactionSet214, interchange.transactions[0].set)
	return &interchange.transactions[0]
}

func (f *tenderedCarrierFixture) expectRecipientMiss() {
	f.recipientRepo.EXPECT().
		GetActiveExternalRecipientByShipmentReference(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("EDITenderRecipient"))
}

func TestRouteShipmentStatus_RecipientHitDoesNotUseOfferFallback(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixture(t)
	shipmentID := pulid.MustNew("shp_")
	recipient := &edi.TenderRecipient{
		ID:                   pulid.MustNew("editr_"),
		SourceOrganizationID: fixture.file.OrganizationID,
		SourceBusinessUnitID: fixture.file.BusinessUnitID,
		SourceShipmentID:     shipmentID,
	}
	fixture.recipientRepo.EXPECT().
		GetActiveExternalRecipientByShipmentReference(
			mock.Anything,
			mock.MatchedBy(
				func(req repositories.GetActiveExternalEDITenderRecipientByReferenceRequest) bool {
					return req.Reference == "SHIP-1001" && req.PartnerID == fixture.partner.ID
				},
			),
		).
		Return(recipient, nil).
		Once()
	fixture.expectSystemUser()
	fixture.commentRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(comment *shipment.ShipmentComment) bool {
			return comment.ShipmentID == shipmentID
		})).
		RunAndReturn(
			func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
				return comment, nil
			},
		).
		Once()

	transaction := parse214Transaction(t, rawShipmentStatus214("", "AF"))
	warnings, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Zero(t, fixture.tenderRepo.acceptedCalls)
}

func TestRouteShipmentStatus_RecipientMissMatchesAcceptedOffer(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixture(t)
	offer := fixture.acceptedOffer()
	fixture.tenderRepo.acceptedOffer = offer
	fixture.expectRecipientMiss()
	fixture.expectSystemUser()

	var recordedComment *shipment.ShipmentComment
	fixture.commentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
				recordedComment = comment
				return comment, nil
			},
		).
		Once()
	var recordedEvent *shipmentevent.Event
	fixture.eventRepo.EXPECT().
		Insert(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, event *shipmentevent.Event) error {
			recordedEvent = event
			return nil
		}).
		Once()

	transaction := parse214Transaction(t, rawShipmentStatus214(offer.ID.String(), "AF"))
	warnings, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Equal(t, 1, fixture.tenderRepo.acceptedCalls)
	request := fixture.tenderRepo.lastAcceptedReq
	require.Equal(t, offer.ID, request.OfferID)
	require.Equal(t, fixture.partner.ID, request.PartnerID)
	require.Equal(t, fixture.file.OrganizationID, request.TenantInfo.OrgID)
	require.Equal(t, fixture.file.BusinessUnitID, request.TenantInfo.BuID)
	require.Contains(t, request.References, "SHIP-1001")
	require.Contains(t, request.References, "PRO-77")
	require.NotContains(t, request.References, offer.ID.String())

	require.NotNil(t, recordedComment)
	require.Equal(t, offer.Tender.ShipmentID, recordedComment.ShipmentID)
	require.Contains(t, recordedComment.Comment, "EDI 214 shipment status received")
	require.Contains(t, recordedComment.Comment, "AF")

	require.NotNil(t, recordedEvent)
	require.Equal(t, offer.Tender.ShipmentID, recordedEvent.ShipmentID)
	require.Equal(t, shipmentevent.TypeStatusChanged, recordedEvent.Type)
	require.Equal(t, shipmentevent.SeverityInfo, recordedEvent.Severity)
	require.Equal(t, shipmentevent.ActorEDI, recordedEvent.ActorType)
	require.Equal(t, "Acme Trucking", recordedEvent.ActorLabel)
	require.Equal(t, "Carrier reported EDI 214 status AF", recordedEvent.Summary)
	require.Equal(t, offer.TenderID.String(), recordedEvent.CorrelationID)
}

func TestRouteShipmentStatus_RecipientMissAndOfferMissWarns(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixture(t)
	fixture.expectRecipientMiss()

	transaction := parse214Transaction(t, rawShipmentStatus214("", "AF"))
	warnings, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(
		t,
		warnings[0],
		"could not be matched to an outbound tender or an accepted tender offer",
	)
	require.Equal(t, 1, fixture.tenderRepo.acceptedCalls)
}

func TestRouteShipmentStatus_OfferBusinessErrorBecomesWarning(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixture(t)
	offer := fixture.acceptedOffer()
	fixture.tenderRepo.acceptedOffer = offer
	fixture.expectRecipientMiss()
	fixture.userRepo.EXPECT().
		GetSystemUser(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewBusinessError("system user is unavailable")).
		Once()

	transaction := parse214Transaction(t, rawShipmentStatus214(offer.ID.String(), "AF"))
	warnings, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], offer.ID.String())
	require.Contains(t, warnings[0], "was not applied")
}

func TestRouteShipmentStatus_OfferInfrastructureErrorPropagates(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixture(t)
	fixture.tenderRepo.acceptedErr = errors.New("database unavailable")
	fixture.expectRecipientMiss()

	transaction := parse214Transaction(t, rawShipmentStatus214("", "AF"))
	_, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "database unavailable")
}

func (f *tenderedCarrierFixture) freightInvoiceMessage(
	payload *edi.FreightInvoicePayload,
) *edi.EDIMessage {
	return &edi.EDIMessage{
		ID:             pulid.MustNew("edim_"),
		OrganizationID: f.file.OrganizationID,
		BusinessUnitID: f.file.BusinessUnitID,
		PayloadSnapshot: edi.DocumentPayload{
			TransactionSet: edi.TransactionSet210,
			FreightInvoice: payload,
		},
	}
}

func freightInvoiceTransaction(t *testing.T) *parsedTransaction {
	t.Helper()
	raw := "ISA*00*          *00*          *ZZ*CARRIER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000401*0*P*>~" +
		"GS*IM*CARRIER*TRENOVA*20260107*1200*401*X*004010~" +
		"ST*210*0001~" +
		"B3**INV-1*BOL-42*PP**150000*20260107***SCAC~" +
		"SE*3*0001~GE*1*401~IEA*1*000000401~"
	interchange, err := parseInterchange(raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	return &interchange.transactions[0]
}

func TestRouteFreightInvoice_RecipientMissMatchesAcceptedOfferByBOL(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixture(t)
	offer := fixture.acceptedOffer()
	fixture.tenderRepo.acceptedOffer = offer
	fixture.expectRecipientMiss()
	fixture.expectSystemUser()
	fixture.commentRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(comment *shipment.ShipmentComment) bool {
			return comment.ShipmentID == offer.Tender.ShipmentID
		})).
		RunAndReturn(
			func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
				return comment, nil
			},
		).
		Once()

	var recordedInvoice *edi.CarrierInvoice
	fixture.invoiceRepo.EXPECT().
		CreateCarrierInvoice(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, entity *edi.CarrierInvoice) (*edi.CarrierInvoice, error) {
				recordedInvoice = entity
				return entity, nil
			},
		).
		Once()

	payload := &edi.FreightInvoicePayload{
		InvoiceNumber:    "INV-1",
		BOL:              "BOL-42",
		TotalAmount:      decimal.NullDecimal{Decimal: decimal.NewFromInt(1500), Valid: true},
		ReferenceNumbers: map[string]string{"BM": "BOL-42"},
	}
	message := fixture.freightInvoiceMessage(payload)
	warnings, err := fixture.service.routeFreightInvoice(
		t.Context(),
		fixture.file,
		fixture.partner,
		message,
		freightInvoiceTransaction(t),
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Equal(t, 1, fixture.tenderRepo.acceptedCalls)
	require.True(t, fixture.tenderRepo.lastAcceptedReq.OfferID.IsNil())
	require.Contains(t, fixture.tenderRepo.lastAcceptedReq.References, "BOL-42")

	require.NotNil(t, recordedInvoice)
	require.Equal(t, offer.Tender.ShipmentID, recordedInvoice.ShipmentID)
	require.Equal(t, offer.CarrierID, recordedInvoice.CarrierID)
	require.True(t, recordedInvoice.TenderRecipientID.IsNil())
	require.True(t, recordedInvoice.ExpectedAmount.Valid)
	require.True(t, recordedInvoice.ExpectedAmount.Decimal.Equal(offer.Rate))
	require.Equal(
		t,
		edi.CarrierInvoiceReconciliationStatusMatched,
		recordedInvoice.ReconciliationStatus,
	)
}

func TestRouteFreightInvoice_PerMileOfferLeavesExpectedAmountNull(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixture(t)
	offer := fixture.acceptedOffer()
	offer.RateMethod = shipment.CarrierRateMethodPerMile
	fixture.tenderRepo.acceptedOffer = offer
	fixture.expectRecipientMiss()
	fixture.expectSystemUser()
	fixture.commentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
				return comment, nil
			},
		).
		Once()

	var recordedInvoice *edi.CarrierInvoice
	fixture.invoiceRepo.EXPECT().
		CreateCarrierInvoice(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, entity *edi.CarrierInvoice) (*edi.CarrierInvoice, error) {
				recordedInvoice = entity
				return entity, nil
			},
		).
		Once()

	payload := &edi.FreightInvoicePayload{
		InvoiceNumber:    "INV-1",
		BOL:              "BOL-42",
		TotalAmount:      decimal.NullDecimal{Decimal: decimal.NewFromInt(1500), Valid: true},
		ReferenceNumbers: map[string]string{},
	}
	message := fixture.freightInvoiceMessage(payload)
	warnings, err := fixture.service.routeFreightInvoice(
		t.Context(),
		fixture.file,
		fixture.partner,
		message,
		freightInvoiceTransaction(t),
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "cannot be compared")
	require.NotNil(t, recordedInvoice)
	require.Equal(t, offer.CarrierID, recordedInvoice.CarrierID)
	require.False(t, recordedInvoice.ExpectedAmount.Valid)
}

func TestRouteFreightInvoice_RecipientMissAndOfferMissStillRecordsInvoice(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixture(t)
	fixture.expectRecipientMiss()

	var recordedInvoice *edi.CarrierInvoice
	fixture.invoiceRepo.EXPECT().
		CreateCarrierInvoice(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, entity *edi.CarrierInvoice) (*edi.CarrierInvoice, error) {
				recordedInvoice = entity
				return entity, nil
			},
		).
		Once()

	payload := &edi.FreightInvoicePayload{
		InvoiceNumber:    "INV-1",
		BOL:              "BOL-42",
		TotalAmount:      decimal.NullDecimal{Decimal: decimal.NewFromInt(1500), Valid: true},
		ReferenceNumbers: map[string]string{},
	}
	message := fixture.freightInvoiceMessage(payload)
	warnings, err := fixture.service.routeFreightInvoice(
		t.Context(),
		fixture.file,
		fixture.partner,
		message,
		freightInvoiceTransaction(t),
	)

	require.NoError(t, err)
	require.NotNil(t, recordedInvoice)
	require.True(t, recordedInvoice.ShipmentID.IsNil())
	require.True(t, recordedInvoice.CarrierID.IsNil())
	foundMissWarning := false
	for _, warning := range warnings {
		if strings.Contains(
			warning,
			"could not be matched to an outbound tender or an accepted tender offer",
		) {
			foundMissWarning = true
		}
	}
	require.True(t, foundMissWarning)
}
