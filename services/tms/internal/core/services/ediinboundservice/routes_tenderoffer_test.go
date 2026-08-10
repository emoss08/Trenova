package ediinboundservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeTenderRepository struct {
	repositories.TenderRepository

	offer   *tender.TenderOffer
	err     error
	lastReq repositories.GetTenderOfferByIDRequest
	calls   int

	acceptedOffer   *tender.TenderOffer
	acceptedErr     error
	lastAcceptedReq repositories.GetAcceptedOfferForPartnerReferenceRequest
	acceptedCalls   int
}

func (f *fakeTenderRepository) GetOfferByID(
	_ context.Context,
	req repositories.GetTenderOfferByIDRequest,
) (*tender.TenderOffer, error) {
	f.calls++
	f.lastReq = req
	if f.err != nil {
		return nil, f.err
	}
	return f.offer, nil
}

func (f *fakeTenderRepository) GetAcceptedOfferForPartnerReference(
	_ context.Context,
	req repositories.GetAcceptedOfferForPartnerReferenceRequest,
) (*tender.TenderOffer, error) {
	f.acceptedCalls++
	f.lastAcceptedReq = req
	if f.acceptedErr != nil {
		return nil, f.acceptedErr
	}
	if f.acceptedOffer == nil {
		return nil, errortypes.NewNotFoundError("Tender offer")
	}
	return f.acceptedOffer, nil
}

type fakeTenderResponseRecorder struct {
	err      error
	requests []*services.TenderResponseRequest
}

func (f *fakeTenderResponseRecorder) RecordResponse(
	_ context.Context,
	req *services.TenderResponseRequest,
) error {
	f.requests = append(f.requests, req)
	return f.err
}

type tenderOfferRouteFixture struct {
	service       *Service
	file          *edi.EDIInboundFile
	partner       *edi.EDIPartner
	tenderRepo    *fakeTenderRepository
	recorder      *fakeTenderResponseRecorder
	recipientRepo *mocks.MockEDITenderRecipientRepository
}

func newTenderOfferRouteFixture(t *testing.T) *tenderOfferRouteFixture {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	partnerID := pulid.MustNew("edip_")
	fixture := &tenderOfferRouteFixture{
		file: &edi.EDIInboundFile{
			ID:             pulid.MustNew("ediinf_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			EDIPartnerID:   partnerID,
		},
		partner: &edi.EDIPartner{
			ID:             partnerID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Kind:           edi.PartnerKindExternal,
			Code:           "CARRIER",
		},
		tenderRepo:    &fakeTenderRepository{},
		recorder:      &fakeTenderResponseRecorder{},
		recipientRepo: mocks.NewMockEDITenderRecipientRepository(t),
	}
	fixture.service = &Service{
		l:                   zap.NewNop(),
		metrics:             metrics.NewEDI(nil, zap.NewNop(), false),
		tenderRecipientRepo: fixture.recipientRepo,
		tenderRepo:          fixture.tenderRepo,
		tenderResponses:     fixture.recorder,
	}
	return fixture
}

func (f *tenderOfferRouteFixture) newOffer(offerID pulid.ID) *tender.TenderOffer {
	partnerID := f.partner.ID
	return &tender.TenderOffer{
		ID:           offerID,
		TenderID:     pulid.MustNew("tnd_"),
		Channel:      tender.ChannelEDI,
		Status:       tender.OfferStatusSent,
		EDIPartnerID: &partnerID,
		Tender: &tender.Tender{
			Status:     tender.StatusActive,
			WorkflowID: "tender-workflow",
		},
	}
}

func rawTenderResponse990(offerRef, reservationCode, remarks string) string {
	raw := "ISA*00*          *00*          *ZZ*CARRIER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000201*0*P*>~" +
		"GS*GF*CARRIER*TRENOVA*20260107*1200*201*X*004010~" +
		"ST*990*0001~" +
		"B1*SCAC*SHIP-1001*20260107*" + reservationCode + "~"
	if offerRef != "" {
		raw += "L11*" + offerRef + "*CR~"
	}
	if remarks != "" {
		raw += "K1*" + remarks + "~"
	}
	raw += "SE*5*0001~GE*1*201~IEA*1*000000201~"
	return raw
}

func parse990Transaction(t *testing.T, raw string) *parsedTransaction {
	t.Helper()
	interchange, err := parseInterchange(raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	require.Equal(t, edi.TransactionSet990, interchange.transactions[0].set)
	return &interchange.transactions[0]
}

func TestRouteTenderResponse_OfferFirstAccept(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	offerID := pulid.MustNew("tof_")
	fixture.tenderRepo.offer = fixture.newOffer(offerID)
	transaction := parse990Transaction(t, rawTenderResponse990(offerID.String(), "A", ""))

	warnings, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Equal(t, 1, fixture.tenderRepo.calls)
	require.Equal(t, offerID, fixture.tenderRepo.lastReq.OfferID)
	require.True(t, fixture.tenderRepo.lastReq.IncludeTender)
	require.Equal(t, fixture.file.OrganizationID, fixture.tenderRepo.lastReq.TenantInfo.OrgID)
	require.Equal(t, fixture.file.BusinessUnitID, fixture.tenderRepo.lastReq.TenantInfo.BuID)
	require.Len(t, fixture.recorder.requests, 1)
	recorded := fixture.recorder.requests[0]
	require.Equal(t, offerID, recorded.OfferID)
	require.Equal(t, tender.ResponseActionAccept, recorded.Action)
	require.Equal(t, tender.ResponseSourceEDI, recorded.Source)
	require.Empty(t, recorded.DeclineReason)
}

func TestRouteTenderResponse_OfferFirstDeclineCarriesReason(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	offerID := pulid.MustNew("tof_")
	fixture.tenderRepo.offer = fixture.newOffer(offerID)
	transaction := parse990Transaction(
		t,
		rawTenderResponse990(offerID.String(), "D", "No capacity"),
	)

	warnings, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Len(t, fixture.recorder.requests, 1)
	recorded := fixture.recorder.requests[0]
	require.Equal(t, tender.ResponseActionDecline, recorded.Action)
	require.Equal(t, "No capacity", recorded.DeclineReason)
}

func TestRouteTenderResponse_OfferDeclineCodesRecordDecline(t *testing.T) {
	t.Parallel()

	for _, code := range []string{"N", "R"} {
		t.Run(code, func(t *testing.T) {
			t.Parallel()

			fixture := newTenderOfferRouteFixture(t)
			offerID := pulid.MustNew("tof_")
			fixture.tenderRepo.offer = fixture.newOffer(offerID)
			transaction := parse990Transaction(t, rawTenderResponse990(offerID.String(), code, ""))

			warnings, err := fixture.service.routeTenderResponse(
				t.Context(),
				fixture.file,
				fixture.partner,
				transaction,
			)

			require.NoError(t, err)
			require.Empty(t, warnings)
			require.Len(t, fixture.recorder.requests, 1)
			recorded := fixture.recorder.requests[0]
			require.Equal(t, tender.ResponseActionDecline, recorded.Action)
			require.Equal(t, code, recorded.DeclineReason)
		})
	}
}

func TestRouteTenderResponse_OfferMissingReservationCodeIsNotApplied(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	offerID := pulid.MustNew("tof_")
	fixture.tenderRepo.offer = fixture.newOffer(offerID)
	transaction := parse990Transaction(t, rawTenderResponse990(offerID.String(), "", ""))

	warnings, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "no reservation code")
	require.Contains(t, warnings[0], offerID.String())
	require.Empty(t, fixture.recorder.requests)
}

func TestRouteTenderResponse_OfferUnknownReservationCodeIsNotApplied(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	offerID := pulid.MustNew("tof_")
	fixture.tenderRepo.offer = fixture.newOffer(offerID)
	transaction := parse990Transaction(t, rawTenderResponse990(offerID.String(), "Q", ""))

	warnings, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], `unknown reservation code "Q"`)
	require.Empty(t, fixture.recorder.requests)
}

func TestRouteTenderResponse_OfferBusinessErrorIsHandled(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	offerID := pulid.MustNew("tof_")
	fixture.tenderRepo.offer = fixture.newOffer(offerID)
	fixture.recorder.err = errortypes.NewBusinessError("This offer is no longer available")
	transaction := parse990Transaction(t, rawTenderResponse990(offerID.String(), "A", ""))

	warnings, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], offerID.String())
	require.Contains(t, warnings[0], "no longer available")
}

func TestRouteTenderResponse_OfferInfrastructureErrorFailsTransaction(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	offerID := pulid.MustNew("tof_")
	fixture.tenderRepo.offer = fixture.newOffer(offerID)
	fixture.recorder.err = errors.New("temporal unavailable")
	transaction := parse990Transaction(t, rawTenderResponse990(offerID.String(), "A", ""))

	_, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "temporal unavailable")
}

func TestRouteTenderResponse_PartnerMismatchDoesNotRecord(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	offerID := pulid.MustNew("tof_")
	offer := fixture.newOffer(offerID)
	otherPartnerID := pulid.MustNew("edip_")
	offer.EDIPartnerID = &otherPartnerID
	fixture.tenderRepo.offer = offer
	transaction := parse990Transaction(t, rawTenderResponse990(offerID.String(), "A", ""))

	warnings, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "was not tendered to partner")
	require.Empty(t, fixture.recorder.requests)
}

func TestRouteTenderResponse_NoOfferReferenceFallsThroughToLegacy(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	transaction := parse990Transaction(t, rawTenderResponse990("", "A", ""))

	fixture.recipientRepo.EXPECT().
		GetActiveExternalRecipientByShipmentReference(
			mock.Anything,
			mock.MatchedBy(
				func(req repositories.GetActiveExternalEDITenderRecipientByReferenceRequest) bool {
					return req.Reference == "SHIP-1001" && req.PartnerID == fixture.partner.ID
				},
			),
		).
		Return(nil, errortypes.NewNotFoundError("EDITenderRecipient")).
		Once()

	warnings, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "could not be matched to an outbound tender")
	require.Equal(t, 0, fixture.tenderRepo.calls)
	require.Empty(t, fixture.recorder.requests)
}

func TestRouteTenderResponse_UnknownOfferFallsThroughToLegacy(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	offerID := pulid.MustNew("tof_")
	fixture.tenderRepo.err = errortypes.NewNotFoundError("TenderOffer")
	transaction := parse990Transaction(t, rawTenderResponse990(offerID.String(), "A", ""))

	fixture.recipientRepo.EXPECT().
		GetActiveExternalRecipientByShipmentReference(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("EDITenderRecipient")).
		Once()

	warnings, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 2)
	require.Contains(t, warnings[0], "references unknown tender offer")
	require.Empty(t, fixture.recorder.requests)
}

func TestTenderOfferReference_IgnoresNonOfferReferences(t *testing.T) {
	t.Parallel()

	details := &tenderResponseDetails{
		shipmentRef: "SHIP-1001",
		references:  []string{"BOL-42", "tof_short", "PRO-77"},
	}
	require.True(t, tenderOfferReference(details).IsNil())

	offerID := pulid.MustNew("tof_")
	details.references = append(details.references, offerID.String())
	require.Equal(t, offerID, tenderOfferReference(details))
}
