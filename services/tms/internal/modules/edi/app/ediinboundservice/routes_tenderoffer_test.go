package ediinboundservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/documentmetrics"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/testutil/mocks/editenderportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeTenderRepository struct {
	tenderports.TenderRepository

	offer   *tender.TenderOffer
	err     error
	lastReq tenderports.GetOfferForPartnerKeyRequest
	calls   int

	acceptedOffer   *tender.TenderOffer
	acceptedErr     error
	lastAcceptedReq tenderports.GetOfferForPartnerKeyRequest
	acceptedCalls   int
}

func (f *fakeTenderRepository) GetOfferForPartnerKey(
	_ context.Context,
	req tenderports.GetOfferForPartnerKeyRequest,
) (*tender.TenderOffer, error) {
	if len(req.Statuses) == 0 {
		f.calls++
		f.lastReq = req
		return found(f.offer, f.err)
	}
	f.acceptedCalls++
	f.lastAcceptedReq = req
	return found(f.acceptedOffer, f.acceptedErr)
}

func found(offer *tender.TenderOffer, err error) (*tender.TenderOffer, error) {
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return nil, errortypes.NewNotFoundError("Tender offer")
	}
	return offer, nil
}

type fakeTenderResponseRecorder struct {
	err      error
	requests []*tenderports.TenderResponseRequest
}

func (f *fakeTenderResponseRecorder) RecordResponse(
	_ context.Context,
	req *tenderports.TenderResponseRequest,
) error {
	f.requests = append(f.requests, req)
	return f.err
}

type tenderOfferRouteFixture struct {
	service       *Service
	file          *ediinboundfile.EDIInboundFile
	partner       *edi.EDIPartner
	tenderRepo    *fakeTenderRepository
	recorder      *fakeTenderResponseRecorder
	recipientRepo *editenderportsmock.MockEDITenderRecipientRepository
}

func newTenderOfferRouteFixture(t *testing.T) *tenderOfferRouteFixture {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	partnerID := pulid.MustNew("edip_")
	fixture := &tenderOfferRouteFixture{
		file: &ediinboundfile.EDIInboundFile{
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
		recipientRepo: editenderportsmock.NewMockEDITenderRecipientRepository(t),
	}
	fixture.service = &Service{
		l:                   zap.NewNop(),
		metrics:             documentmetrics.NewEDI(nil, zap.NewNop(), false),
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

const offeredProNumber = "S2603001042"

func rawTenderResponse990(key, reservationCode, remarks string) string {
	raw := "ISA*00*          *00*          *ZZ*CARRIER        *ZZ*TRENOVA        " +
		"*260107*1200*^*00401*000000201*0*P*>~" +
		"GS*GF*CARRIER*TRENOVA*20260107*1200*201*X*004010~" +
		"ST*990*0001~" +
		"B1*SCAC*" + key + "*20260107*" + reservationCode + "~"
	if remarks != "" {
		raw += "K1*" + remarks + "~"
	}
	raw += "SE*4*0001~GE*1*201~IEA*1*000000201~"
	return raw
}

// parse990Transaction reads a test 990 and what it answers, as a partner's response map would
// read B1, L11 and K1.
func parse990Transaction(t *testing.T, raw string) (*parsedTransaction, *tenderResponseDetails) {
	t.Helper()
	interchange, err := parseTestInterchange(t, raw)
	require.NoError(t, err)
	require.Len(t, interchange.transactions, 1)
	transaction := &interchange.transactions[0]
	require.Equal(t, edidocument.TransactionSet990, transaction.set)
	details := &tenderResponseDetails{}
	if b1 := findSegment(transaction.segments, "B1"); b1 != nil {
		details.scac = elementValue(b1, 1)
		details.shipmentRef = elementValue(b1, 2)
		details.reservationCode = elementValue(b1, 4)
	}
	for _, l11 := range findSegmentsByID(transaction.segments, "L11") {
		details.references = append(details.references, elementValue(&l11, 1))
	}
	if k1 := findSegment(transaction.segments, "K1"); k1 != nil {
		details.remarks = elementValue(k1, 1)
	}
	return transaction, details
}

func TestRouteTenderResponse_MatchesTheOfferByTheNumberItQuotes(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	offerID := pulid.MustNew("tof_")
	fixture.tenderRepo.offer = fixture.newOffer(offerID)
	transaction, details := parse990Transaction(t, rawTenderResponse990(offeredProNumber, "A", ""))

	warnings, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
	)

	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Equal(t, 1, fixture.tenderRepo.calls)
	request := fixture.tenderRepo.lastReq
	require.Equal(t, fixture.partner.ID, request.PartnerID)
	require.Equal(t, fixture.file.OrganizationID, request.TenantInfo.OrgID)
	require.Equal(t, fixture.file.BusinessUnitID, request.TenantInfo.BuID)
	require.Empty(t, request.Statuses)
	require.Equal(t, []tender.PartnerKey{{ProNumber: offeredProNumber}}, request.Keys)
	require.Len(t, fixture.recorder.requests, 1)
	recorded := fixture.recorder.requests[0]
	require.Equal(t, offerID, recorded.OfferID)
	require.Equal(t, tender.ResponseActionAccept, recorded.Action)
	require.Equal(t, tender.ResponseSourceEDI, recorded.Source)
	require.Empty(t, recorded.DeclineReason)
}

func TestRouteTenderResponse_AMoveKeyNamesTheShipmentAndTheMove(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	fixture.tenderRepo.offer = fixture.newOffer(pulid.MustNew("tof_"))
	transaction, details := parse990Transaction(
		t,
		rawTenderResponse990(offeredProNumber+"-2", "A", ""),
	)

	_, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
	)

	require.NoError(t, err)
	require.Equal(t, []tender.PartnerKey{
		{ProNumber: offeredProNumber + "-2"},
		{ProNumber: offeredProNumber, Move: 2},
	}, fixture.tenderRepo.lastReq.Keys)
}

func TestRouteTenderResponse_OfferDeclineCarriesReason(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	fixture.tenderRepo.offer = fixture.newOffer(pulid.MustNew("tof_"))
	transaction, details := parse990Transaction(
		t,
		rawTenderResponse990(offeredProNumber, "D", "No capacity"),
	)

	warnings, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
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
			fixture.tenderRepo.offer = fixture.newOffer(pulid.MustNew("tof_"))
			transaction, details := parse990Transaction(
				t,
				rawTenderResponse990(offeredProNumber, code, ""),
			)

			warnings, err := fixture.service.routeTenderResponse(
				t.Context(),
				fixture.file,
				fixture.partner,
				transaction,
				details,
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
	fixture.tenderRepo.offer = fixture.newOffer(pulid.MustNew("tof_"))
	transaction, details := parse990Transaction(t, rawTenderResponse990(offeredProNumber, "", ""))

	warnings, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "no reservation code")
	require.Contains(t, warnings[0], offeredProNumber)
	require.NotContains(t, warnings[0], "tof_")
	require.Empty(t, fixture.recorder.requests)
}

func TestRouteTenderResponse_OfferUnknownReservationCodeIsNotApplied(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	fixture.tenderRepo.offer = fixture.newOffer(pulid.MustNew("tof_"))
	transaction, details := parse990Transaction(t, rawTenderResponse990(offeredProNumber, "Q", ""))

	warnings, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], `unknown reservation code "Q"`)
	require.Empty(t, fixture.recorder.requests)
}

func TestRouteTenderResponse_OfferBusinessErrorIsHandled(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	fixture.tenderRepo.offer = fixture.newOffer(pulid.MustNew("tof_"))
	fixture.recorder.err = errortypes.NewBusinessError("This offer is no longer available")
	transaction, details := parse990Transaction(t, rawTenderResponse990(offeredProNumber, "A", ""))

	warnings, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], offeredProNumber)
	require.Contains(t, warnings[0], "no longer available")
}

func TestRouteTenderResponse_OfferInfrastructureErrorFailsTransaction(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	fixture.tenderRepo.offer = fixture.newOffer(pulid.MustNew("tof_"))
	fixture.recorder.err = errors.New("temporal unavailable")
	transaction, details := parse990Transaction(t, rawTenderResponse990(offeredProNumber, "A", ""))

	_, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "temporal unavailable")
}

func TestRouteTenderResponse_OfferLookupFailureFailsTransaction(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	fixture.tenderRepo.err = errors.New("database unavailable")
	transaction, details := parse990Transaction(t, rawTenderResponse990(offeredProNumber, "A", ""))

	_, err := fixture.service.routeTenderResponse(
		t.Context(),
		fixture.file,
		fixture.partner,
		transaction,
		details,
	)

	require.ErrorContains(t, err, "database unavailable")
	require.Empty(t, fixture.recorder.requests)
}

func TestRouteTenderResponse_NoOfferFallsThroughToTheRecipients(t *testing.T) {
	t.Parallel()

	fixture := newTenderOfferRouteFixture(t)
	transaction, details := parse990Transaction(t, rawTenderResponse990(offeredProNumber, "A", ""))

	fixture.recipientRepo.EXPECT().
		GetActiveExternalRecipientByShipmentReference(
			mock.Anything,
			mock.MatchedBy(
				func(req editenderports.GetActiveExternalEDITenderRecipientByReferenceRequest) bool {
					return req.Reference == offeredProNumber && req.PartnerID == fixture.partner.ID
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
		details,
	)

	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "could not be matched to an outbound tender")
	require.Equal(t, 1, fixture.tenderRepo.calls)
	require.Empty(t, fixture.recorder.requests)
}

func findSegmentsByID(
	segments []edix12inspect.X12Segment,
	segmentID string,
) []edix12inspect.X12Segment {
	result := make([]edix12inspect.X12Segment, 0, 4)
	for index := range segments {
		if segments[index].SegmentID == segmentID {
			result = append(result, segments[index])
		}
	}
	return result
}
