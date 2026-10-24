package checkcallservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/dbtest"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testNow = int64(1_800_000_000)

type memoryCheckCalls struct {
	checkcallports.CheckCallRepository
	rows      []*checkcall.CheckCall
	monitored []*checkcallports.MonitoredMove
}

func (r *memoryCheckCalls) Insert(
	_ context.Context,
	entity *checkcall.CheckCall,
) (*checkcall.CheckCall, bool, error) {
	for _, row := range r.rows {
		if row.Source == entity.Source && row.SourceKey == entity.SourceKey {
			return row, false, nil
		}
	}
	r.rows = append(r.rows, entity)
	return entity, true, nil
}

func (r *memoryCheckCalls) LatestForMove(
	_ context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
) (*checkcall.CheckCall, error) {
	var latest *checkcall.CheckCall
	for _, row := range r.rows {
		if row.ShipmentMoveID == moveID && (latest == nil || row.ReceivedAt > latest.ReceivedAt) {
			latest = row
		}
	}
	return latest, nil
}

func (r *memoryCheckCalls) LatestForMoves(
	_ context.Context,
	_ pagination.TenantInfo,
	moveIDs []pulid.ID,
) (map[pulid.ID]*checkcall.CheckCall, error) {
	wanted := make(map[pulid.ID]struct{}, len(moveIDs))
	for _, id := range moveIDs {
		wanted[id] = struct{}{}
	}
	out := make(map[pulid.ID]*checkcall.CheckCall, len(moveIDs))
	for _, row := range r.rows {
		if _, ok := wanted[row.ShipmentMoveID]; !ok {
			continue
		}
		if latest := out[row.ShipmentMoveID]; latest == nil || row.ReportedAt > latest.ReportedAt {
			out[row.ShipmentMoveID] = row
		}
	}
	return out, nil
}

func (r *memoryCheckCalls) ListMonitoredMoves(
	_ context.Context,
	_ pagination.TenantInfo,
	_ int,
) ([]*checkcallports.MonitoredMove, error) {
	return r.monitored, nil
}

type memoryLinks struct {
	checkcallports.TrackingLinkRepository
	links   []*checkcall.TrackingLink
	touched int
}

func (r *memoryLinks) Create(_ context.Context, entity *checkcall.TrackingLink) error {
	r.links = append(r.links, entity)
	return nil
}

func (r *memoryLinks) GetByHash(_ context.Context, hash string) (*checkcall.TrackingLink, error) {
	for _, link := range r.links {
		if link.TokenHash == hash {
			return link, nil
		}
	}
	return nil, nil
}

func (r *memoryLinks) GetActiveForMove(
	_ context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
	now int64,
) (*checkcall.TrackingLink, error) {
	for _, link := range r.links {
		if link.ShipmentMoveID == moveID && link.IsUsable(now) {
			return link, nil
		}
	}
	return nil, nil
}

func (r *memoryLinks) RevokeForMove(
	_ context.Context,
	req *checkcallports.RevokeTrackingLinksRequest,
) (int, error) {
	count := 0
	for _, link := range r.links {
		if link.ShipmentMoveID == req.MoveID && link.RevokedAt == nil {
			at := req.At
			link.RevokedAt = &at
			count++
		}
	}
	return count, nil
}

func (r *memoryLinks) Touch(context.Context, pagination.TenantInfo, pulid.ID, int64) error {
	r.touched++
	return nil
}

type fakeMoves struct {
	shipmentports.ShipmentMoveRepository
	move *shipment.ShipmentMove
}

func (f *fakeMoves) GetByID(
	_ context.Context,
	req *shipmentports.GetMoveByIDRequest,
) (*shipment.ShipmentMove, error) {
	if f.move == nil || req.MoveID != f.move.ID {
		return nil, errortypes.NewNotFoundError("Shipment move")
	}
	return f.move, nil
}

type fakeShipments struct {
	shipmentports.ShipmentRepository
}

func (fakeShipments) GetByID(
	_ context.Context,
	req *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	return &shipment.Shipment{ID: req.ID, ProNumber: "PRO-77"}, nil
}

type fakeAssignments struct {
	shipmentports.CarrierAssignmentRepository
	assignment *shipment.CarrierAssignment
}

func (f *fakeAssignments) GetActiveByMoveID(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*shipment.CarrierAssignment, error) {
	if f.assignment == nil {
		return nil, errortypes.NewNotFoundError("Carrier assignment")
	}
	return f.assignment, nil
}

type fakeCarriers struct {
	carrierports.CarrierRepository
}

func (fakeCarriers) GetByID(
	_ context.Context,
	req carrierports.GetCarrierByIDRequest,
) (*carrier.Carrier, error) {
	return &carrier.Carrier{ID: req.ID, Name: "Redline Freight"}, nil
}

type fakeOrgs struct {
	tenantports.OrganizationRepository
}

func (fakeOrgs) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Name: "Acme Logistics"}, nil
}

type fakeControls struct {
	dispatchcontrolports.DispatchControlRepository
	control *dispatchcontrol.DispatchControl
	err     error
}

func (f *fakeControls) GetByOrgID(
	context.Context,
	dispatchcontrolports.GetDispatchControlRequest,
) (*dispatchcontrol.DispatchControl, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.control == nil {
		return nil, errortypes.NewNotFoundError(
			"DispatchControl not found within your organization",
		)
	}
	return f.control, nil
}

type fakeTracking struct {
	trackingeventports.TrackingEventService
	observed []*trackingeventports.RecordObservedStopEventParams
	reported []*trackingeventports.RecordReportedStopEventParams
	outcome  trackingevent.Outcome
	err      error
}

func (f *fakeTracking) RecordObserved(
	_ context.Context,
	params *trackingeventports.RecordObservedStopEventParams,
) (*trackingeventports.StopEventResult, error) {
	f.observed = append(f.observed, params)
	if f.err != nil {
		return nil, f.err
	}
	return &trackingeventports.StopEventResult{
		Event:   &trackingevent.TrackingEvent{ID: pulid.MustNew("tkev_")},
		Outcome: f.outcome,
	}, nil
}

func (f *fakeTracking) RecordReported(
	_ context.Context,
	params *trackingeventports.RecordReportedStopEventParams,
) (*trackingeventports.StopEventResult, error) {
	f.reported = append(f.reported, params)
	if f.err != nil {
		return nil, f.err
	}
	return &trackingeventports.StopEventResult{
		Event:   &trackingevent.TrackingEvent{ID: pulid.MustNew("tkev_")},
		Outcome: trackingevent.OutcomeApplied,
	}, nil
}

type fakeTemplates struct {
	documenttemplateports.DocumentTemplateResolver
	data any
}

func (f *fakeTemplates) RenderMessage(
	_ context.Context,
	req *documenttemplateports.RenderMessageRequest,
) (*documenttemplateports.RenderedMessage, error) {
	f.data = req.Data
	return &documenttemplateports.RenderedMessage{Subject: "s", HTML: "h", Text: "t"}, nil
}

type fakeEmail struct {
	emailports.EmailService
	sent []*emailports.SendEmailRequest
	err  error
}

func (f *fakeEmail) Send(
	_ context.Context,
	req *emailports.SendEmailRequest,
) (*email.Message, error) {
	f.sent = append(f.sent, req)
	if f.err != nil {
		return nil, f.err
	}
	return &email.Message{}, nil
}

type observedCalls struct {
	moves []pulid.ID
}

func (o *observedCalls) CheckCallRecorded(
	_ context.Context,
	_ pagination.TenantInfo,
	_, moveID pulid.ID,
) {
	o.moves = append(o.moves, moveID)
}

type harness struct {
	svc         *Service
	checkCalls  *memoryCheckCalls
	links       *memoryLinks
	moves       *fakeMoves
	assignments *fakeAssignments
	tracking    *fakeTracking
	templates   *fakeTemplates
	email       *fakeEmail
	controls    *fakeControls
	observer    *observedCalls
	tenant      pagination.TenantInfo
	move        *shipment.ShipmentMove
	pickup      *shipment.Stop
	delivery    *shipment.Stop
	carrierID   pulid.ID
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	pickup := &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipment.StopTypePickup,
		Status:   shipment.StopStatusNew,
		Sequence: 0,
		Location: &location.Location{
			Name:  "Shipper DC",
			City:  "Dallas",
			State: &usstate.UsState{Abbreviation: "TX"},
		},
	}
	delivery := &shipment.Stop{
		ID:       pulid.MustNew("stp_"),
		Type:     shipment.StopTypeDelivery,
		Status:   shipment.StopStatusNew,
		Sequence: 1,
	}
	move := &shipment.ShipmentMove{
		ID:           pulid.MustNew("smv_"),
		ShipmentID:   pulid.MustNew("shp_"),
		Status:       shipment.MoveStatusInTransit,
		CoverageType: shipment.MoveCoverageTypeCarrier,
		Stops:        []*shipment.Stop{pickup, delivery},
	}
	carrierID := pulid.MustNew("car_")

	h := &harness{
		checkCalls: &memoryCheckCalls{},
		links:      &memoryLinks{},
		moves:      &fakeMoves{move: move},
		assignments: &fakeAssignments{assignment: &shipment.CarrierAssignment{
			CarrierID: carrierID,
			Status:    shipment.CarrierAssignmentStatusConfirmed,
		}},
		tracking:  &fakeTracking{outcome: trackingevent.OutcomeApplied},
		templates: &fakeTemplates{},
		email:     &fakeEmail{},
		controls: &fakeControls{control: &dispatchcontrol.DispatchControl{
			EnableCheckCallMonitoring: true,
			CheckCallIntervalHours:    4,
		}},
		observer:  &observedCalls{},
		tenant:    tenantInfo,
		move:      move,
		pickup:    pickup,
		delivery:  delivery,
		carrierID: carrierID,
	}
	h.svc = New(Params{
		DB:     dbtest.NopConnection{},
		Logger: zap.NewNop(),
		Config: &config.Config{Tendering: config.TenderingConfig{
			PublicBaseURL: "https://app.example.com/",
		}},
		Repo:                h.checkCalls,
		Links:               h.links,
		MoveRepo:            h.moves,
		ShipmentRepo:        fakeShipments{},
		CarrierAssignments:  h.assignments,
		CarrierRepo:         fakeCarriers{},
		OrgRepo:             fakeOrgs{},
		DispatchControlRepo: h.controls,
		TrackingEvents:      h.tracking,
		Templates:           h.templates,
		EmailService:        h.email,
		Observers:           []checkcallports.CheckCallObserver{h.observer},
	})
	h.svc.now = func() int64 { return testNow }
	return h
}

func (h *harness) issue(t *testing.T, email string) *checkcallports.IssuedTrackingLink {
	t.Helper()
	issued, err := h.svc.IssueLink(t.Context(), &checkcallports.IssueTrackingLinkRequest{
		TenantInfo:     h.tenant,
		MoveID:         h.move.ID,
		RecipientEmail: email,
		IssuedByID:     h.tenant.UserID,
	})
	require.NoError(t, err)
	return issued
}

func tokenOf(t *testing.T, issued *checkcallports.IssuedTrackingLink) string {
	t.Helper()
	index := strings.LastIndex(issued.URL, "/")
	require.Positive(t, index)
	return issued.URL[index+1:]
}

func int64Ptr(value int64) *int64 { return &value }

func TestLog_AnUpdateIsStoredWithoutATrackingEvent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	result, err := h.svc.Log(t.Context(), &checkcallports.LogCheckCallRequest{
		TenantInfo:   h.tenant,
		MoveID:       h.move.ID,
		Source:       checkcall.SourceDispatcher,
		SourceKey:    "idem-1",
		ReportedByID: h.tenant.UserID,
		Details: checkcallports.CheckCallDetails{
			Kind:        checkcall.KindUpdate,
			ReportedAt:  int64Ptr(testNow - 600),
			City:        " Tulsa ",
			StateCode:   "ok",
			Delayed:     true,
			DelayReason: "Weather on I-44",
			ContactName: "Jo",
		},
	})
	require.NoError(t, err)

	stored := result.CheckCall
	assert.Equal(t, checkcall.KindUpdate, stored.Kind)
	assert.Equal(t, "Tulsa", stored.City)
	assert.Equal(t, "OK", stored.StateCode)
	assert.Equal(t, testNow-600, stored.ReportedAt)
	assert.Equal(t, testNow, stored.ReceivedAt)
	assert.True(t, stored.StopID.IsNil())
	assert.True(t, stored.TrackingEventID.IsNil())
	assert.Empty(t, h.tracking.reported)
	assert.Equal(t, []pulid.ID{h.move.ID}, h.observer.moves)
}

func TestLog_AnArrivalBecomesADispatcherTrackingEvent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	result, err := h.svc.Log(t.Context(), &checkcallports.LogCheckCallRequest{
		TenantInfo:   h.tenant,
		MoveID:       h.move.ID,
		Source:       checkcall.SourceAgent,
		SourceKey:    "run_1:call_2",
		ReportedByID: h.tenant.UserID,
		Details: checkcallports.CheckCallDetails{
			Kind:       checkcall.KindArrival,
			StopID:     h.delivery.ID,
			ReportedAt: int64Ptr(testNow - 300),
		},
	})
	require.NoError(t, err)

	require.Len(t, h.tracking.reported, 1)
	reported := h.tracking.reported[0]
	assert.Equal(t, trackingevent.SourceAgent, reported.Source)
	assert.Equal(t, shipmentports.StopActualActionArrive, reported.Action)
	assert.Equal(t, "run_1:call_2", reported.SourceKey)
	assert.Equal(t, testNow-300, *reported.OccurredAt)
	assert.Equal(t, h.delivery.ID, result.CheckCall.StopID)
	assert.False(t, result.CheckCall.TrackingEventID.IsNil())
	assert.Equal(t, trackingevent.OutcomeApplied, result.Outcome)

	again, err := h.svc.Log(t.Context(), &checkcallports.LogCheckCallRequest{
		TenantInfo: h.tenant,
		MoveID:     h.move.ID,
		Source:     checkcall.SourceAgent,
		SourceKey:  "run_1:call_2",
		Details: checkcallports.CheckCallDetails{
			Kind:       checkcall.KindArrival,
			StopID:     h.delivery.ID,
			ReportedAt: int64Ptr(testNow - 300),
		},
	})
	require.NoError(t, err)
	assert.True(t, again.Replayed)
	assert.Len(t, h.checkCalls.rows, 1)
}

func TestLog_RefusesBadInput(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	base := checkcallports.LogCheckCallRequest{
		TenantInfo: h.tenant,
		MoveID:     h.move.ID,
		Source:     checkcall.SourceDispatcher,
	}
	cases := map[string]func(*checkcallports.LogCheckCallRequest){
		"carrier source":       func(r *checkcallports.LogCheckCallRequest) { r.Source = checkcall.SourceCarrier },
		"unknown kind":         func(r *checkcallports.LogCheckCallRequest) { r.Details.Kind = "Dwell" },
		"arrival without stop": func(r *checkcallports.LogCheckCallRequest) { r.Details.Kind = checkcall.KindArrival },
		"future time": func(r *checkcallports.LogCheckCallRequest) {
			r.Details.ReportedAt = int64Ptr(testNow + 3600)
		},
		"eight days ago": func(r *checkcallports.LogCheckCallRequest) {
			r.Details.ReportedAt = int64Ptr(testNow - 8*24*3600)
		},
		"delay without reason": func(r *checkcallports.LogCheckCallRequest) { r.Details.Delayed = true },
		"eta too far": func(r *checkcallports.LogCheckCallRequest) {
			r.Details.EstimatedArrival = int64Ptr(testNow + 40*24*3600)
		},
		"source key too long": func(r *checkcallports.LogCheckCallRequest) {
			r.SourceKey = strings.Repeat("k", 256)
		},
	}
	for name, mutate := range cases {
		req := base
		req.Details = checkcallports.CheckCallDetails{Kind: checkcall.KindUpdate}
		mutate(&req)
		_, err := h.svc.Log(t.Context(), &req)
		require.Error(t, err, name)
		assert.True(t, errortypes.IsRefusal(err), name)
	}

	_, err := h.svc.Log(t.Context(), &checkcallports.LogCheckCallRequest{
		TenantInfo: h.tenant,
		MoveID:     h.move.ID,
		Source:     checkcall.SourceDispatcher,
		Details: checkcallports.CheckCallDetails{
			Kind:   checkcall.KindDeparture,
			StopID: pulid.MustNew("stp_"),
		},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, h.checkCalls.rows)
}

func TestIssueLink_ReplacesTheLiveLinkAndStoresOnlyTheHash(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	first := h.issue(t, "")
	second := h.issue(t, "")

	require.Len(t, h.links.links, 2)
	assert.NotNil(t, h.links.links[0].RevokedAt)
	assert.Nil(t, h.links.links[1].RevokedAt)
	assert.True(t, strings.HasPrefix(second.URL, "https://app.example.com/carrier-tracking/"))
	token := tokenOf(t, second)
	assert.Equal(t, tokenutils.Hash(token), second.Link.TokenHash)
	assert.NotContains(t, second.Link.TokenHash, token)
	assert.Equal(t, h.carrierID, second.Link.CarrierID)
	assert.Equal(t, testNow+checkcall.DefaultLinkTTLSeconds, second.Link.ExpiresAt)
	assert.False(t, second.Emailed)
	assert.NotEqual(t, first.URL, second.URL)
}

func TestIssueLink_EmailsTheCarrier(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	issued := h.issue(t, " dispatch@redline.example ")

	assert.True(t, issued.Emailed)
	require.Len(t, h.email.sent, 1)
	assert.Equal(t, []string{"dispatch@redline.example"}, h.email.sent[0].To)
	assert.Equal(
		t,
		"carrier-tracking-link-"+issued.Link.ID.String(),
		h.email.sent[0].IdempotencyKey,
	)
	assert.Equal(t, "dispatch@redline.example", issued.Link.RecipientEmail)
}

func TestIssueLink_AFailedEmailStillLeavesTheLink(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.email.err = errors.New("smtp unavailable")
	issued := h.issue(t, "dispatch@redline.example")

	assert.False(t, issued.Emailed)
	assert.Contains(t, issued.EmailError, "smtp unavailable")
	require.Len(t, h.links.links, 1)
	assert.Nil(t, h.links.links[0].RevokedAt)
}

func TestIssueLink_RefusesLoadsItCannotTrack(t *testing.T) {
	t.Parallel()

	for name, setup := range map[string]func(*harness){
		"own fleet":     func(h *harness) { h.move.CoverageType = shipment.MoveCoverageTypeDriver },
		"completed":     func(h *harness) { h.move.Status = shipment.MoveStatusCompleted },
		"no carrier":    func(h *harness) { h.assignments.assignment = nil },
		"no public url": func(h *harness) { h.svc.cfg = &config.Config{} },
	} {
		h := newHarness(t)
		setup(h)
		_, err := h.svc.IssueLink(t.Context(), &checkcallports.IssueTrackingLinkRequest{
			TenantInfo: h.tenant,
			MoveID:     h.move.ID,
		})
		require.Error(t, err, name)
		assert.True(t, errortypes.IsBusinessError(err), name)
		assert.Empty(t, h.links.links, name)
	}

	h := newHarness(t)
	_, err := h.svc.IssueLink(t.Context(), &checkcallports.IssueTrackingLinkRequest{
		TenantInfo:     h.tenant,
		MoveID:         h.move.ID,
		RecipientEmail: "not an email",
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsRefusal(err))
}

func TestRevokeLink(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	issued := h.issue(t, "")
	require.NoError(t, h.svc.RevokeLink(t.Context(), &checkcallports.RevokeTrackingLinkRequest{
		TenantInfo: h.tenant,
		MoveID:     h.move.ID,
	}))
	assert.NotNil(t, issued.Link.RevokedAt)

	active, err := h.svc.ActiveLink(t.Context(), h.tenant, h.move.ID)
	require.NoError(t, err)
	assert.Nil(t, active)

	err = h.svc.RevokeLink(t.Context(), &checkcallports.RevokeTrackingLinkRequest{
		TenantInfo: h.tenant,
		MoveID:     h.move.ID,
	})
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestPreviewByToken_ShowsTheStopsAndNoCustomerData(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	arrived := testNow - 7200
	h.pickup.ActualArrival = &arrived
	token := tokenOf(t, h.issue(t, ""))

	view, err := h.svc.PreviewByToken(t.Context(), token)
	require.NoError(t, err)

	assert.Equal(t, "Acme Logistics", view.CompanyName)
	assert.Equal(t, "Redline Freight", view.CarrierName)
	assert.Equal(t, "PRO-77", view.ShipmentProNumber)
	assert.False(t, view.Closed)
	require.Len(t, view.Stops, 2)
	assert.Equal(t, "Shipper DC", view.Stops[0].LocationName)
	assert.Equal(t, "TX", view.Stops[0].StateCode)
	assert.Equal(t, &arrived, view.Stops[0].ArrivedAt)
	assert.Nil(t, view.LastCheckCallAt)
}

func TestPostByToken_RecordsACarrierArrival(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	issued := h.issue(t, "")
	token := tokenOf(t, issued)
	latitude, longitude := 32.78, -96.8

	result, err := h.svc.PostByToken(t.Context(), token, &checkcallports.PublicCheckCallInput{
		Details: checkcallports.CheckCallDetails{
			Kind:        checkcall.KindArrival,
			StopID:      h.pickup.ID,
			ReportedAt:  int64Ptr(testNow + 3*3600),
			Latitude:    &latitude,
			Longitude:   &longitude,
			ContactName: "Sam",
		},
		ClientEventID: "tap-1",
	})
	require.NoError(t, err)
	assert.Equal(t, trackingevent.OutcomeApplied, result.Outcome)

	require.Len(t, h.tracking.observed, 1)
	observed := h.tracking.observed[0]
	assert.Equal(t, trackingevent.SourceCarrier, observed.Source)
	assert.Equal(t, shipment.VisitArrival, observed.Kind)
	assert.Equal(t, trackingevent.MatchDirect, observed.MatchMethod)
	assert.Equal(t, testNow, observed.EventAt, "a time ahead of the server is read as now")
	assert.Equal(t, "ctl:"+issued.Link.ID.String()+":tap-1", observed.SourceKey)
	assert.Equal(t, "carrier_tracking_link:"+issued.Link.ID.String(), observed.RawReference)

	require.Len(t, h.checkCalls.rows, 1)
	stored := h.checkCalls.rows[0]
	assert.Equal(t, checkcall.SourceCarrier, stored.Source)
	assert.Equal(t, issued.Link.ID, stored.TrackingLinkID)
	assert.Equal(t, "Sam", stored.ContactName)
	assert.Equal(t, 1, h.links.touched)

	again, err := h.svc.PostByToken(t.Context(), token, &checkcallports.PublicCheckCallInput{
		Details: checkcallports.CheckCallDetails{
			Kind:   checkcall.KindArrival,
			StopID: h.pickup.ID,
		},
		ClientEventID: "tap-1",
	})
	require.NoError(t, err)
	assert.True(t, again.Replayed)
	assert.Len(t, h.checkCalls.rows, 1)
}

func TestPostByToken_EveryDeadLinkGetsTheSameAnswer(t *testing.T) {
	t.Parallel()

	for name, setup := range map[string]func(*harness, *checkcallports.IssuedTrackingLink) string{
		"unknown": func(*harness, *checkcallports.IssuedTrackingLink) string { return "nope" },
		"empty":   func(*harness, *checkcallports.IssuedTrackingLink) string { return " " },
		"expired": func(_ *harness, issued *checkcallports.IssuedTrackingLink) string {
			issued.Link.ExpiresAt = testNow - 1
			return ""
		},
		"revoked": func(_ *harness, issued *checkcallports.IssuedTrackingLink) string {
			at := testNow - 1
			issued.Link.RevokedAt = &at
			return ""
		},
		"carrier replaced": func(h *harness, _ *checkcallports.IssuedTrackingLink) string {
			h.assignments.assignment = &shipment.CarrierAssignment{
				CarrierID: pulid.MustNew("car_"),
				Status:    shipment.CarrierAssignmentStatusConfirmed,
			}
			return ""
		},
		"move gone": func(h *harness, _ *checkcallports.IssuedTrackingLink) string {
			h.moves.move = nil
			return ""
		},
	} {
		h := newHarness(t)
		issued := h.issue(t, "")
		token := setup(h, issued)
		if token == "" {
			token = tokenOf(t, issued)
		}
		_, err := h.svc.PostByToken(t.Context(), token, &checkcallports.PublicCheckCallInput{
			Details: checkcallports.CheckCallDetails{Kind: checkcall.KindUpdate},
		})
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), "This tracking link is no longer valid", name)
		_, err = h.svc.PreviewByToken(t.Context(), token)
		require.Error(t, err, name)
		assert.Empty(t, h.checkCalls.rows, name)
	}
}

func TestPostByToken_AFinishedLoadIsReadOnly(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	token := tokenOf(t, h.issue(t, ""))
	h.move.Status = shipment.MoveStatusCompleted

	view, err := h.svc.PreviewByToken(t.Context(), token)
	require.NoError(t, err)
	assert.True(t, view.Closed)

	_, err = h.svc.PostByToken(t.Context(), token, &checkcallports.PublicCheckCallInput{
		Details: checkcallports.CheckCallDetails{Kind: checkcall.KindUpdate},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestPostByToken_RefusesAStopOffTheLoadAndABadEventID(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	token := tokenOf(t, h.issue(t, ""))

	_, err := h.svc.PostByToken(t.Context(), token, &checkcallports.PublicCheckCallInput{
		Details: checkcallports.CheckCallDetails{
			Kind:   checkcall.KindDeparture,
			StopID: pulid.MustNew("stp_"),
		},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsRefusal(err))

	_, err = h.svc.PostByToken(t.Context(), token, &checkcallports.PublicCheckCallInput{
		Details:       checkcallports.CheckCallDetails{Kind: checkcall.KindUpdate},
		ClientEventID: "has space",
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsRefusal(err))
	assert.Empty(t, h.checkCalls.rows)
	assert.Empty(t, h.tracking.observed)
}

func TestListOverdue(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	two := int16(2)
	h.checkCalls.monitored = []*checkcallports.MonitoredMove{
		{MoveID: pulid.MustNew("smv_"), LastContactAt: testNow - 3*3600},
		{
			MoveID:                pulid.MustNew("smv_"),
			LastContactAt:         testNow - 3*3600,
			CustomerIntervalHours: &two,
		},
		{MoveID: pulid.MustNew("smv_"), LastContactAt: testNow - 10*3600},
		{MoveID: pulid.MustNew("smv_"), LastContactAt: 0},
	}

	overdue, err := h.svc.ListOverdue(t.Context(), h.tenant, testNow)
	require.NoError(t, err)
	require.Len(t, overdue, 2)
	assert.Equal(t, h.checkCalls.monitored[2].MoveID, overdue[0].MoveID)
	assert.Equal(t, int64(6*3600), overdue[0].OverdueBy)
	assert.Equal(t, int16(4), overdue[0].IntervalHours)
	assert.Equal(t, h.checkCalls.monitored[1].MoveID, overdue[1].MoveID)
	assert.Equal(t, int16(2), overdue[1].IntervalHours)

	h.controls.control.EnableCheckCallMonitoring = false
	overdue, err = h.svc.ListOverdue(t.Context(), h.tenant, testNow)
	require.NoError(t, err)
	assert.Empty(t, overdue)
}

func TestPreviewLog_BuildsTheRecordWithoutWriting(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	preview, err := h.svc.PreviewLog(t.Context(), &checkcallports.LogCheckCallRequest{
		TenantInfo: h.tenant,
		MoveID:     h.move.ID,
		Source:     checkcall.SourceAgent,
		SourceKey:  "run_1:call_3",
		Details: checkcallports.CheckCallDetails{
			Kind:       checkcall.KindArrival,
			StopID:     h.delivery.ID,
			ReportedAt: int64Ptr(testNow - 120),
			StateCode:  "tx",
		},
	})
	require.NoError(t, err)

	assert.Equal(t, h.move.ID, preview.Move.ID)
	assert.Equal(t, h.delivery.ID, preview.CheckCall.StopID)
	assert.Equal(t, "TX", preview.CheckCall.StateCode)
	assert.Equal(t, testNow-120, preview.CheckCall.ReportedAt)
	assert.Empty(t, h.tracking.reported)
	assert.Empty(t, h.checkCalls.rows)
	assert.Empty(t, h.observer.moves)

	_, err = h.svc.PreviewLog(t.Context(), &checkcallports.LogCheckCallRequest{
		TenantInfo: h.tenant,
		MoveID:     h.move.ID,
		Source:     checkcall.SourceAgent,
		Details: checkcallports.CheckCallDetails{
			Kind:       checkcall.KindUpdate,
			ReportedAt: int64Ptr(testNow + 3600),
		},
	})
	require.Error(t, err)
}

func TestLog_RefusesACanceledLoad(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.move.Status = shipment.MoveStatusCanceled

	_, err := h.svc.Log(t.Context(), &checkcallports.LogCheckCallRequest{
		TenantInfo: h.tenant,
		MoveID:     h.move.ID,
		Source:     checkcall.SourceDispatcher,
		Details:    checkcallports.CheckCallDetails{Kind: checkcall.KindUpdate},
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, h.checkCalls.rows)
}

func TestListOverdue_AnOrganizationWithoutDispatchControlsIsNotMonitored(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.checkCalls.monitored = []*checkcallports.MonitoredMove{
		{MoveID: pulid.MustNew("smv_"), LastContactAt: testNow - 30*3600},
	}
	h.controls.control = nil

	overdue, err := h.svc.ListOverdue(t.Context(), h.tenant, testNow)
	require.NoError(t, err)
	assert.Empty(t, overdue)

	h.controls.err = errors.New("connection reset")
	_, err = h.svc.ListOverdue(t.Context(), h.tenant, testNow)
	require.Error(t, err)
}
