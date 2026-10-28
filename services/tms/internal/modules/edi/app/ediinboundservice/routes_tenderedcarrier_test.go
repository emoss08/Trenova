package ediinboundservice

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/ladingdoc"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/testutil/mocks/carrierportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/ediportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/editenderportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentcommentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmenteventportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/shipmentmoveportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/trackingeventportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
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
	recipientRepo *editenderportsmock.MockEDITenderRecipientRepository
	commentRepo   *shipmentcommentportsmock.MockShipmentCommentRepository
	userRepo      *tenantportsmock.MockUserRepository
	eventRepo     *shipmenteventportsmock.MockShipmentEventRepository
	invoiceRepo   *ediportsmock.MockEDICarrierInvoiceRepository
	moveRepo      *shipmentmoveportsmock.MockShipmentMoveRepository
	tracking      *trackingeventportsmock.MockTrackingEventService
	matcher       *carrierportsmock.MockCarrierInvoiceAutoMatcher
}

func (f *tenderedCarrierFixture) withInvoiceMatcher(
	t *testing.T,
) *carrierportsmock.MockCarrierInvoiceAutoMatcher {
	t.Helper()
	f.matcher = carrierportsmock.NewMockCarrierInvoiceAutoMatcher(t)
	f.service.invoiceMatcher = f.matcher
	return f.matcher
}

func newTenderedCarrierFixture(t *testing.T) *tenderedCarrierFixture {
	t.Helper()
	return buildTenderedCarrierFixture(t, false)
}

func newTenderedCarrierFixtureWithStopActuals(t *testing.T) *tenderedCarrierFixture {
	t.Helper()
	return buildTenderedCarrierFixture(t, true)
}

func buildTenderedCarrierFixture(t *testing.T, withStopActuals bool) *tenderedCarrierFixture {
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
		recipientRepo: editenderportsmock.NewMockEDITenderRecipientRepository(t),
		commentRepo:   shipmentcommentportsmock.NewMockShipmentCommentRepository(t),
		userRepo:      tenantportsmock.NewMockUserRepository(t),
		eventRepo:     shipmenteventportsmock.NewMockShipmentEventRepository(t),
		invoiceRepo:   ediportsmock.NewMockEDICarrierInvoiceRepository(t),
	}
	fixture.invoiceRepo.EXPECT().
		GetCarrierInvoiceByNumber(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("CarrierInvoice not found")).
		Maybe()
	params := ediservice.Params{
		Logger:              zap.NewNop(),
		ShipmentCommentRepo: fixture.commentRepo,
		UserRepo:            fixture.userRepo,
		ShipmentEventRepo:   fixture.eventRepo,
		CarrierInvoiceRepo:  fixture.invoiceRepo,
	}
	if withStopActuals {
		fixture.moveRepo = shipmentmoveportsmock.NewMockShipmentMoveRepository(t)
		fixture.tracking = trackingeventportsmock.NewMockTrackingEventService(t)
		params.ShipmentMoveRepo = fixture.moveRepo
		params.TrackingEvents = fixture.tracking
	}
	ediSvc := ediservice.New(params)
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
	segments := []string{"ST*214*0001", "B10*PRO-77*SHIP-1001*SCAC"}
	if offerRef != "" {
		segments = append(segments, "L11*"+offerRef+"*CR")
	}
	segments = append(segments, "LX*1", "AT7*"+statusCode+"*NS***20260107*1200")
	segments = append(segments, "SE*"+strconv.Itoa(len(segments)+1)+"*0001")
	return "ISA*00*          *00*          *ZZ*CARRIER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000301*0*P*>~" +
		"GS*QM*CARRIER*TRENOVA*20260107*1200*301*X*004010~" +
		strings.Join(segments, "~") + "~" +
		"GE*1*301~IEA*1*000000301~"
}

// parse214Transaction reads a test 214 through the broker starter's status map, as a carrier
// partner's 214 is read, every lookup answered as unmapped.
func parse214Transaction(t *testing.T, raw string) (*parsedTransaction, *shipmentStatusDetails) {
	t.Helper()
	interchange, err := parseTestInterchange(t, raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	transaction := &interchange.transactions[0]
	require.Equal(t, edi.TransactionSet214, transaction.set)

	client := lading.Default()
	starter, err := client.Starter(t.Context(), &lading.StarterRequest{
		Name:  "carrier",
		Label: "Carrier",
		Role:  lading.RoleBroker,
		Them:  &[2]string{"ZZ", "CARRIER"},
		Us:    &[2]string{"ZZ", "TRENOVA"},
		Usage: "production",
	})
	require.NoError(t, err)
	encoded, err := lading.Encode(&lading.Project{Files: starter.Files}, &lading.Context{
		Org: lading.Org{Name: "Trenova", SCAC: "TRNV", Zone: "America/New_York"},
	})
	require.NoError(t, err)
	check, err := client.Check(t.Context(), encoded)
	require.NoError(t, err)
	_, clause := check.Receiving("214")
	require.NotNil(t, clause)

	request := &lading.RunRequest{Map: clause.Map, Input: lading.X12Input(raw), Rules: true}
	result, err := client.Run(t.Context(), encoded, request)
	require.NoError(t, err)
	for _, need := range result.Needs {
		request.Lookups = append(
			request.Lookups,
			lading.LookupAnswer{Kind: need.Kind, Code: need.Code},
		)
	}
	if len(request.Lookups) > 0 {
		result, err = client.Run(t.Context(), encoded, request)
		require.NoError(t, err)
	}
	require.Nil(t, result.Error)
	require.Len(t, result.Transactions, 1)
	status, err := ladingdoc.ToShipmentStatus(result.Transactions[0].Document)
	require.NoError(t, err)
	return transaction, statusDetails(transaction, status)
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
		BusinessUnitID:       fixture.file.BusinessUnitID,
		SourceShipmentID:     shipmentID,
	}
	fixture.recipientRepo.EXPECT().
		GetActiveExternalRecipientByShipmentReference(
			mock.Anything,
			mock.MatchedBy(
				func(req editenderports.GetActiveExternalEDITenderRecipientByReferenceRequest) bool {
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

	transaction, details := parse214Transaction(t, rawShipmentStatus214("", "AF"))
	warnings, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
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

	transaction, details := parse214Transaction(t, rawShipmentStatus214(offer.ID.String(), "AF"))
	warnings, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
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

	transaction, details := parse214Transaction(t, rawShipmentStatus214("", "AF"))
	warnings, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
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

	transaction, details := parse214Transaction(t, rawShipmentStatus214(offer.ID.String(), "AF"))
	warnings, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], offer.ID.String())
	require.Contains(t, warnings[0], "was not applied")
}

func (f *tenderedCarrierFixture) tenderedMoveWithStops(
	moveID pulid.ID,
) (*shipment.ShipmentMove, *shipment.Stop, *shipment.Stop) {
	origin := &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipment.StopTypePickup,
		Status:   shipment.StopStatusNew,
		Sequence: 0,
	}
	destination := &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipment.StopTypeDelivery,
		Status:   shipment.StopStatusNew,
		Sequence: 1,
	}
	move := &shipment.ShipmentMove{
		ID:     moveID,
		Status: shipment.MoveStatusAssigned,
		Stops:  []*shipment.Stop{origin, destination},
	}
	return move, origin, destination
}

func (f *tenderedCarrierFixture) expectTrackingEvents(
	move *shipment.ShipmentMove,
	times int,
) *[]*trackingeventports.RecordObservedStopEventParams {
	recorded := make([]*trackingeventports.RecordObservedStopEventParams, 0, times)
	f.tracking.EXPECT().
		RecordObserved(mock.Anything, mock.AnythingOfType("*trackingeventports.RecordObservedStopEventParams")).
		RunAndReturn(
			func(
				_ context.Context,
				params *trackingeventports.RecordObservedStopEventParams,
			) (*trackingeventports.StopEventResult, error) {
				recorded = append(recorded, params)
				return &trackingeventports.StopEventResult{
					Move:    move,
					Outcome: trackingevent.OutcomeApplied,
				}, nil
			},
		).
		Times(times)
	return &recorded
}

func TestRouteShipmentStatus_X3RecordsOriginArrivalWithEventTime(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixtureWithStopActuals(t)
	offer := fixture.acceptedOffer()
	moveID := pulid.MustNew("sm_")
	offer.Tender.ShipmentMoveID = moveID
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
	fixture.eventRepo.EXPECT().
		Insert(mock.Anything, mock.AnythingOfType("*shipmentevent.Event")).
		Return(nil).
		Once()

	move, origin, _ := fixture.tenderedMoveWithStops(moveID)
	fixture.moveRepo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *shipmentmoveports.GetMoveByIDRequest) bool {
			return req.MoveID == moveID && req.ExpandMoveDetails
		})).
		Return(move, nil).
		Once()
	recordedEvents := fixture.expectTrackingEvents(move, 1)

	transaction, details := parse214Transaction(t, rawShipmentStatus214(offer.ID.String(), "X3"))
	warnings, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Len(t, *recordedEvents, 1)
	recorded := (*recordedEvents)[0]
	require.Equal(t, shipment.VisitArrival, recorded.Kind)
	require.Equal(t, origin.ID, recorded.StopID)
	require.Equal(t, moveID, recorded.MoveID)
	require.Equal(t, trackingevent.SourceEDI, recorded.Source)
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 1, 7, 12, 0, 0, 0, chicago).Unix(), recorded.EventAt)
	require.Equal(t, "edi_inbound_file:"+fixture.file.ID.String()+":0001:0", recorded.RawReference)
}

func TestRouteShipmentStatus_D1RecordsArriveAndDepartAtDestination(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixtureWithStopActuals(t)
	offer := fixture.acceptedOffer()
	moveID := pulid.MustNew("sm_")
	offer.Tender.ShipmentMoveID = moveID
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
	fixture.eventRepo.EXPECT().
		Insert(mock.Anything, mock.AnythingOfType("*shipmentevent.Event")).
		Return(nil).
		Once()

	move, _, destination := fixture.tenderedMoveWithStops(moveID)
	fixture.moveRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentmoveports.GetMoveByIDRequest")).
		Return(move, nil).
		Once()
	recordedEvents := fixture.expectTrackingEvents(move, 2)

	transaction, details := parse214Transaction(t, rawShipmentStatus214(offer.ID.String(), "D1"))
	warnings, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	recorded := *recordedEvents
	require.Len(t, recorded, 2)
	require.Equal(t, shipment.VisitArrival, recorded[0].Kind)
	require.Equal(t, shipment.VisitDeparture, recorded[1].Kind)
	require.Equal(t, destination.ID, recorded[0].StopID)
	require.Equal(t, destination.ID, recorded[1].StopID)
}

func TestRouteShipmentStatus_StopActualBusinessErrorBecomesWarningKeepsComment(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixtureWithStopActuals(t)
	offer := fixture.acceptedOffer()
	moveID := pulid.MustNew("sm_")
	offer.Tender.ShipmentMoveID = moveID
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
	fixture.eventRepo.EXPECT().
		Insert(mock.Anything, mock.AnythingOfType("*shipmentevent.Event")).
		Return(nil).
		Once()

	move, _, _ := fixture.tenderedMoveWithStops(moveID)
	fixture.moveRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentmoveports.GetMoveByIDRequest")).
		Return(move, nil).
		Once()
	fixture.tracking.EXPECT().
		RecordObserved(mock.Anything, mock.Anything).
		Return(&trackingeventports.StopEventResult{
			Outcome: trackingevent.OutcomeRefused,
			Reason:  "Complete the earlier stops on this load first",
		}, nil).
		Once()

	transaction, details := parse214Transaction(t, rawShipmentStatus214(offer.ID.String(), "X1"))
	warnings, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "Complete the earlier stops on this load first")
	require.NotNil(t, recordedComment)
	require.Contains(t, recordedComment.Comment, "X1")
}

func TestRouteShipmentStatus_OfferInfrastructureErrorPropagates(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixture(t)
	fixture.tenderRepo.acceptedErr = errors.New("database unavailable")
	fixture.expectRecipientMiss()

	transaction, details := parse214Transaction(t, rawShipmentStatus214("", "AF"))
	_, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
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
		"B3**INV-1*BOL-42*PP**20260107*150000****SCAC~" +
		"SE*3*0001~GE*1*401~IEA*1*000000401~"
	interchange, err := parseTestInterchange(t, raw)
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

func TestRouteFreightInvoice_TenderedOfferAutoMatches(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixture(t)
	offer := fixture.acceptedOffer()
	fixture.tenderRepo.acceptedOffer = offer
	fixture.expectRecipientMiss()
	fixture.expectSystemUser()
	matcher := fixture.withInvoiceMatcher(t)
	fixture.commentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
				return comment, nil
			},
		).
		Once()

	invoiceID := pulid.MustNew("edici_")
	fixture.invoiceRepo.EXPECT().
		CreateCarrierInvoice(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, entity *edi.CarrierInvoice) (*edi.CarrierInvoice, error) {
				entity.ID = invoiceID
				return entity, nil
			},
		).
		Once()
	var matchRequest *carrierports.AutoMatchInboundInvoiceRequest
	matcher.EXPECT().
		AutoMatchInboundInvoice(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req *carrierports.AutoMatchInboundInvoiceRequest) {
			matchRequest = req
		}).
		Return(&carrierports.AutoMatchInboundInvoiceResult{
			Matched:      true,
			AutoAccepted: true,
			Status:       "Resolved",
		}, nil).
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
	require.NotNil(t, matchRequest)
	require.Equal(t, invoiceID, matchRequest.EDICarrierInvoiceID)
	require.Equal(t, fixture.file.OrganizationID, matchRequest.TenantInfo.OrgID)
	require.Equal(t, fixture.file.BusinessUnitID, matchRequest.TenantInfo.BuID)
}

func TestRouteFreightInvoice_AutoMatchWarningsSurfaceOnFile(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixture(t)
	offer := fixture.acceptedOffer()
	fixture.tenderRepo.acceptedOffer = offer
	fixture.expectRecipientMiss()
	fixture.expectSystemUser()
	matcher := fixture.withInvoiceMatcher(t)
	fixture.commentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
				return comment, nil
			},
		).
		Once()
	fixture.invoiceRepo.EXPECT().
		CreateCarrierInvoice(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, entity *edi.CarrierInvoice) (*edi.CarrierInvoice, error) {
				entity.ID = pulid.MustNew("edici_")
				return entity, nil
			},
		).
		Once()
	matcher.EXPECT().
		AutoMatchInboundInvoice(mock.Anything, mock.Anything).
		Return(&carrierports.AutoMatchInboundInvoiceResult{
			Warnings: []string{
				"carrier invoice INV-1 was not auto-matched: " +
					"no carrier assignment matches its pro number or shipment reference",
			},
		}, nil).
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
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "was not auto-matched")
}

func TestRouteFreightInvoice_AutoMatchErrorDegradesToWarning(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixture(t)
	offer := fixture.acceptedOffer()
	fixture.tenderRepo.acceptedOffer = offer
	fixture.expectRecipientMiss()
	fixture.expectSystemUser()
	matcher := fixture.withInvoiceMatcher(t)
	fixture.commentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
				return comment, nil
			},
		).
		Once()
	fixture.invoiceRepo.EXPECT().
		CreateCarrierInvoice(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, entity *edi.CarrierInvoice) (*edi.CarrierInvoice, error) {
				entity.ID = pulid.MustNew("edici_")
				return entity, nil
			},
		).
		Once()
	matcher.EXPECT().
		AutoMatchInboundInvoice(mock.Anything, mock.Anything).
		Return(nil, errors.New("settlement database unavailable")).
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
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "carrier invoice INV-1 could not be auto-matched")
	require.Contains(t, warnings[0], "settlement database unavailable")
}

func TestRouteFreightInvoice_NoOfferSkipsAutoMatch(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixture(t)
	fixture.expectRecipientMiss()
	fixture.withInvoiceMatcher(t)

	fixture.invoiceRepo.EXPECT().
		CreateCarrierInvoice(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, entity *edi.CarrierInvoice) (*edi.CarrierInvoice, error) {
				entity.ID = pulid.MustNew("edici_")
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
	_, err := fixture.service.routeFreightInvoice(
		t.Context(),
		fixture.file,
		fixture.partner,
		message,
		freightInvoiceTransaction(t),
	)

	require.NoError(t, err)
	fixture.matcher.AssertNotCalled(t, "AutoMatchInboundInvoice")
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

func TestRouteShipmentStatus_EveryAT7BecomesATimedTrackingEvent(t *testing.T) {
	t.Parallel()

	fixture := newTenderedCarrierFixtureWithStopActuals(t)
	offer := fixture.acceptedOffer()
	moveID := pulid.MustNew("sm_")
	offer.Tender.ShipmentMoveID = moveID
	fixture.tenderRepo.acceptedOffer = offer
	fixture.expectRecipientMiss()
	fixture.expectSystemUser()

	comments := make([]*shipment.ShipmentComment, 0, 3)
	fixture.commentRepo.EXPECT().
		Create(mock.Anything, mock.Anything).
		RunAndReturn(
			func(_ context.Context, comment *shipment.ShipmentComment) (*shipment.ShipmentComment, error) {
				comments = append(comments, comment)
				return comment, nil
			},
		).
		Times(3)
	events := make([]*shipmentevent.Event, 0, 3)
	fixture.eventRepo.EXPECT().
		Insert(mock.Anything, mock.AnythingOfType("*shipmentevent.Event")).
		RunAndReturn(func(_ context.Context, event *shipmentevent.Event) error {
			events = append(events, event)
			return nil
		}).
		Times(3)

	origin := &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipment.StopTypePickup,
		Status:   shipment.StopStatusNew,
		Sequence: 0,
	}
	intermediate := &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipment.StopTypeDelivery,
		Status:   shipment.StopStatusNew,
		Sequence: 1,
	}
	final := &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipment.StopTypeDelivery,
		Status:   shipment.StopStatusNew,
		Sequence: 2,
		Location: &location.Location{
			City:     "Denver",
			Timezone: "America/Denver",
			State:    &usstate.UsState{Abbreviation: "CO"},
		},
	}
	move := &shipment.ShipmentMove{
		ID:     moveID,
		Status: shipment.MoveStatusAssigned,
		Stops:  []*shipment.Stop{origin, intermediate, final},
	}
	fixture.moveRepo.EXPECT().
		GetByID(mock.Anything, mock.AnythingOfType("*shipmentmoveports.GetMoveByIDRequest")).
		Return(move, nil).
		Times(3)
	recordedEvents := fixture.expectTrackingEvents(move, 3)

	raw := "ISA*00*          *00*          *ZZ*CARRIER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000302*0*P*>~" +
		"GS*QM*CARRIER*TRENOVA*20260107*1200*302*X*004010~" +
		"ST*214*0007~" +
		"B10*PRO-77*SHIP-1001*SCAC~" +
		"L11*" + offer.ID.String() + "*CR~" +
		"LX*1~" +
		"AT7*AF*NS***20260107*0945*ET~" +
		"L11*0*QN~" +
		"LX*2~" +
		"AT7*X1*NS***20260107*1415*CS~" +
		"L11*1*QN~" +
		"LX*3~" +
		"AT7*X1*NS***20260108*0700*LT~" +
		"MS1*DENVER*CO*USA~" +
		"SE*13*0007~GE*1*302~IEA*1*000000302~"

	transaction, details := parse214Transaction(t, raw)
	warnings, err := fixture.service.routeShipmentStatus(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
	)

	require.NoError(t, err)
	require.Empty(t, warnings)

	newYork, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	denver, err := time.LoadLocation("America/Denver")
	require.NoError(t, err)

	recorded := *recordedEvents
	require.Len(t, recorded, 3)

	require.Equal(t, origin.ID, recorded[0].StopID)
	require.Equal(t, shipment.VisitDeparture, recorded[0].Kind)
	require.Equal(t, trackingevent.MatchStopReference, recorded[0].MatchMethod)
	require.Equal(t, time.Date(2026, 1, 7, 9, 45, 0, 0, newYork).Unix(), recorded[0].EventAt)

	require.Equal(t, intermediate.ID, recorded[1].StopID)
	require.Equal(t, shipment.VisitArrival, recorded[1].Kind)
	require.Equal(t, trackingevent.MatchStopReference, recorded[1].MatchMethod)
	require.Equal(t, time.Date(2026, 1, 7, 20, 15, 0, 0, time.UTC).Unix(), recorded[1].EventAt)

	require.Equal(t, final.ID, recorded[2].StopID)
	require.Equal(t, shipment.VisitArrival, recorded[2].Kind)
	require.Equal(t, trackingevent.MatchLocation, recorded[2].MatchMethod)
	require.Equal(t, time.Date(2026, 1, 8, 7, 0, 0, 0, denver).Unix(), recorded[2].EventAt)

	keys := map[string]struct{}{}
	for index, params := range recorded {
		keys[params.SourceKey] = struct{}{}
		require.Equal(
			t,
			"edi_inbound_file:"+fixture.file.ID.String()+":0007:"+strconv.Itoa(index),
			params.RawReference,
		)
	}
	require.Len(t, keys, 3)

	require.Len(t, comments, 3)
	require.Contains(t, comments[0].Comment, "AF")
	require.Contains(t, comments[2].Comment, "X1")
	require.Len(t, events, 3)
	require.Equal(t, recorded[2].EventAt, events[2].OccurredAt)
}
