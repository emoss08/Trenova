package shipmentsuggestionservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentsuggestion"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fixedNow = time.Date(2026, 10, 6, 16, 30, 0, 0, time.UTC)

type console struct {
	services.DispatchConsoleService
	board      *dispatchconsoleservice.Board
	candidates map[pulid.ID][]*dispatchcandidateservice.CandidateScore
}

func (f *console) GetBoard(
	context.Context,
	*dispatchconsoleservice.GetBoardRequest,
) (*dispatchconsoleservice.Board, error) {
	return f.board, nil
}

func (f *console) GetMoveCandidates(
	_ context.Context,
	req *dispatchconsoleservice.MoveCandidatesRequest,
) ([]*dispatchcandidateservice.CandidateScore, error) {
	return f.candidates[req.MoveID], nil
}

type capabilities struct{ caps services.ShipmentBoardCapabilities }

func (f *capabilities) Capabilities(
	context.Context,
	pagination.TenantInfo,
) (*services.ShipmentBoardCapabilities, error) {
	return &f.caps, nil
}

type watchlist struct{ late []services.LateDelivery }

func (f *watchlist) Watchlist(
	context.Context,
	pagination.TenantInfo,
	string,
) (*services.ShipmentWatchlist, error) {
	return &services.ShipmentWatchlist{
		Deliveries: services.ShipmentDeliveryWatch{WorstLate: f.late},
	}, nil
}

type comments struct {
	services.ShipmentCommentService
	byShipment map[pulid.ID][]*shipment.ShipmentComment
}

func (f *comments) ListByShipmentID(
	_ context.Context,
	req *repositories.ListShipmentCommentsRequest,
) (*pagination.CursorListResult[*shipment.ShipmentComment], error) {
	return &pagination.CursorListResult[*shipment.ShipmentComment]{
		Items: f.byShipment[req.ShipmentID],
	}, nil
}

type desk struct{ entries []*detentionservice.DeskEntry }

func (f *desk) ListDesk(
	context.Context,
	pagination.TenantInfo,
) ([]*detentionservice.DeskEntry, error) {
	return f.entries, nil
}

type decisionStore struct {
	records map[string]*shipmentsuggestion.DecisionRecord
	pruned  int
}

func (f *decisionStore) ListSince(
	_ context.Context,
	req *repositories.ListSuggestionDecisionsRequest,
) ([]*shipmentsuggestion.DecisionRecord, error) {
	out := make([]*shipmentsuggestion.DecisionRecord, 0)
	for _, record := range f.records {
		if record.DecidedAt >= req.Since {
			out = append(out, record)
		}
	}

	return out, nil
}

func (f *decisionStore) Upsert(
	_ context.Context,
	record *shipmentsuggestion.DecisionRecord,
) (*shipmentsuggestion.DecisionRecord, error) {
	f.records[record.SuggestionKey] = record

	return record, nil
}

func (f *decisionStore) Delete(
	_ context.Context,
	req *repositories.DeleteSuggestionDecisionRequest,
) (*shipmentsuggestion.DecisionRecord, error) {
	record := f.records[req.Key]
	delete(f.records, req.Key)

	return record, nil
}

func (f *decisionStore) Prune(context.Context, *repositories.PruneSuggestionDecisionsRequest) error {
	f.pruned++

	return nil
}

type tenders struct {
	repositories.TenderRepository
	live *tender.Tender
}

func (f *tenders) GetLiveByMoveID(
	context.Context,
	repositories.GetLiveTenderByMoveRequest,
) (*tender.Tender, error) {
	return f.live, nil
}

type guard struct {
	services.TenderGuard
	canceled []pulid.ID
}

func (f *guard) CancelLiveTenderForMove(
	_ context.Context,
	_ pagination.TenantInfo,
	moveID pulid.ID,
	_ string,
) error {
	f.canceled = append(f.canceled, moveID)

	return nil
}

type assigner struct {
	services.AssignmentService
	unassigned []pulid.ID
}

func (f *assigner) Unassign(
	_ context.Context,
	req *repositories.UnassignShipmentMoveRequest,
) error {
	f.unassigned = append(f.unassigned, req.ShipmentMoveID)

	return nil
}

type assignmentReads struct {
	repositories.AssignmentRepository
	assignment *shipment.Assignment
}

func (f *assignmentReads) GetByMoveID(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*shipment.Assignment, error) {
	if f.assignment == nil {
		return nil, errortypes.NewNotFoundError("Assignment not found")
	}

	return f.assignment, nil
}

type harness struct {
	service   *Service
	console   *console
	caps      *capabilities
	watchlist *watchlist
	comments  *comments
	desk      *desk
	decisions *decisionStore
	tenders   *tenders
	guard     *guard
	assigner  *assigner
	reads     *assignmentReads
}

func newHarness(operation tenant.OperationType) *harness {
	h := &harness{
		console:   &console{board: &dispatchconsoleservice.Board{}, candidates: map[pulid.ID][]*dispatchcandidateservice.CandidateScore{}},
		caps:      &capabilities{caps: services.ShipmentBoardCapabilities{OperationType: operation, HOS: true}},
		watchlist: &watchlist{},
		comments:  &comments{byShipment: map[pulid.ID][]*shipment.ShipmentComment{}},
		desk:      &desk{},
		decisions: &decisionStore{records: map[string]*shipmentsuggestion.DecisionRecord{}},
		tenders:   &tenders{},
		guard:     &guard{},
		assigner:  &assigner{},
		reads:     &assignmentReads{},
	}
	h.service = NewWithDependencies(&Dependencies{
		Console:         h.console,
		Capabilities:    h.caps,
		Watchlist:       h.watchlist,
		Comments:        h.comments,
		Detention:       h.desk,
		Decisions:       h.decisions,
		Tenders:         h.tenders,
		TenderGuard:     h.guard,
		Assignments:     h.assigner,
		AssignmentReads: h.reads,
		Now:             func() time.Time { return fixedNow },
	})

	return h
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_"), UserID: pulid.MustNew("usr_")}
}

func uncovered(pro string, pickupIn time.Duration) *dispatchconsoleservice.BoardMove {
	revenue := 3590.0
	return &dispatchconsoleservice.BoardMove{BoardMove: &repositories.BoardMove{
		MoveID:            pulid.MustNew("smv_"),
		ShipmentID:        pulid.MustNew("shp_"),
		ProNumber:         pro,
		OriginCity:        "San Antonio",
		DestinationCity:   "Dallas",
		OriginWindowStart: fixedNow.Add(pickupIn).Unix(),
		Revenue:           &revenue,
	}}
}

func (h *harness) queue(t *testing.T) *services.ShipmentSuggestionQueue {
	t.Helper()
	out, err := h.service.Suggestions(t.Context(), &services.SuggestionRequest{
		TenantInfo: testTenant(),
		UserID:     pulid.MustNew("usr_"),
		Timezone:   "America/Chicago",
	})
	require.NoError(t, err)

	return out
}

func kinds(items []*services.ShipmentSuggestion) []services.SuggestionKind {
	out := make([]services.SuggestionKind, 0, len(items))
	for _, item := range items {
		out = append(out, item.Kind)
	}

	return out
}

func TestCoverage_OffersTheBestUnblockedDriverToAnAssetCarrier(t *testing.T) {
	t.Parallel()

	h := newHarness(tenant.OperationTypeAsset)
	move := uncovered("S2610-1214", 2*time.Hour)
	later := uncovered("S2610-1300", 30*time.Hour)
	covered := uncovered("S2610-1301", time.Hour)
	covered.IsCovered = true
	h.console.board.Moves = []*dispatchconsoleservice.BoardMove{later, covered, move}
	deadhead := 5.0
	blocked := &dispatchcandidateservice.CandidateScore{
		WorkerID: pulid.MustNew("wrk_"), WorkerName: "Blocked Driver",
		Findings: []dispatcheligibility.Finding{{Code: "HOS", Severity: dispatcheligibility.SeverityBlock}},
	}
	best := &dispatchcandidateservice.CandidateScore{
		WorkerID: pulid.MustNew("wrk_"), TractorID: pulid.MustNew("trc_"),
		WorkerName: "Gwen Grant", Score: 96, DeadheadMiles: &deadhead,
		DriveRemainingMs: int64(6 * time.Hour / time.Millisecond),
	}
	h.console.candidates[move.MoveID] = []*dispatchcandidateservice.CandidateScore{blocked, best}

	items := h.queue(t).Items

	require.Len(t, items, 1, "only loads picking up in the next day that nothing covers")
	item := items[0]
	assert.Equal(t, services.SuggestionCoverage, item.Kind)
	assert.Equal(t, "coverage:"+move.MoveID.String(), item.Key)
	assert.Equal(t, "Assign Gwen Grant to San Antonio → Dallas", item.Title)
	assert.Equal(t, "5 mi out with 6h of drive time left. Pickup closes at 13:30.", item.Reason)
	assert.Equal(t, []string{"$3590", "5 mi out", "96% fit"}, item.Impact)
	assert.Equal(t, services.SuggestionActionAssignDriver, item.Primary.Type)
	assert.Equal(t, best.WorkerID, item.Primary.WorkerID)
	assert.Equal(t, best.TractorID, item.Primary.TractorID)
	assert.Equal(t, move.MoveID, item.Primary.MoveID)
}

func TestCoverage_TendersWhenNoDriverCanTakeIt(t *testing.T) {
	t.Parallel()

	both := newHarness(tenant.OperationTypeBoth)
	move := uncovered("S2610-1214", 2*time.Hour)
	both.console.board.Moves = []*dispatchconsoleservice.BoardMove{move}

	items := both.queue(t).Items
	require.Len(t, items, 1)
	assert.Equal(t, services.SuggestionTender, items[0].Kind)
	assert.Equal(t, services.SuggestionActionTenderCarrier, items[0].Primary.Type)
	assert.Equal(t, "tender:"+move.MoveID.String(), items[0].Key)

	asset := newHarness(tenant.OperationTypeAsset)
	asset.console.board.Moves = []*dispatchconsoleservice.BoardMove{move}
	items = asset.queue(t).Items
	require.Len(t, items, 1)
	assert.Equal(t, services.SuggestionActionReview, items[0].Primary.Type,
		"a fleet with no broker side and no free driver gets a review, not a tender")
}

func TestCoverage_LatePickupIsDangerAndSortsFirst(t *testing.T) {
	t.Parallel()

	h := newHarness(tenant.OperationTypeBrokerage)
	soon := uncovered("SOON", 3*time.Hour)
	late := uncovered("LATE", -30*time.Minute)
	h.console.board.Moves = []*dispatchconsoleservice.BoardMove{soon, late}

	items := h.queue(t).Items
	require.Len(t, items, 2)
	assert.Equal(t, "LATE", items[0].ProNumber)
	assert.Equal(t, services.SuggestionToneDanger, items[0].Tone)
	assert.Equal(t, services.SuggestionToneWarning, items[1].Tone)
}

func TestDelayNotice_SkipsCustomersAlreadyToldThisHour(t *testing.T) {
	t.Parallel()

	h := newHarness(tenant.OperationTypeAsset)
	told := services.LateDelivery{ShipmentID: pulid.MustNew("shp_"), ProNumber: "S1", DeltaMinutes: 202, City: "New Orleans", CustomerName: "Cargill Protein"}
	untold := services.LateDelivery{ShipmentID: pulid.MustNew("shp_"), ProNumber: "S2", DeltaMinutes: 188, City: "Houston", CustomerName: "PepsiCo"}
	h.watchlist.late = []services.LateDelivery{told, untold}
	h.comments.byShipment[told.ShipmentID] = []*shipment.ShipmentComment{{
		CreatedAt: fixedNow.Add(-10 * time.Minute).Unix(),
		Type:      shipment.CommentTypeCustomerUpdate,
		Metadata:  map[string]any{"source": "board", "tool": "notify_shipment_delay"},
	}}

	items := h.queue(t).Items
	require.Len(t, items, 1)
	item := items[0]
	assert.Equal(t, services.SuggestionDelayNotice, item.Kind)
	assert.Equal(t, untold.ShipmentID, item.ShipmentID)
	assert.Equal(t, "Let PepsiCo know about the delay", item.Title)
	assert.Equal(t, []string{"+3h 8m"}, item.Impact)
	assert.Contains(t, item.Primary.Message, "running about 3h 8m behind")
}

func TestHoursOfService_WarnsAboutWorkingDriversRunningOut(t *testing.T) {
	t.Parallel()

	h := newHarness(tenant.OperationTypeAsset)
	worker := pulid.MustNew("wrk_")
	move := uncovered("S2610-0601", -2*time.Hour)
	move.IsCovered = true
	move.AssignedWorkerID = worker
	move.DestinationCity = "Nashville"
	h.console.board.Moves = []*dispatchconsoleservice.BoardMove{move}
	h.console.board.Drivers = []*dispatchconsoleservice.BoardDriver{{
		BoardDriver:      &repositories.BoardDriver{WorkerID: worker, FirstName: "Tom", LastName: "Avery"},
		Availability:     dispatchconsoleservice.AvailabilityWorking,
		DriveRemainingMs: int64(95 * time.Minute / time.Millisecond),
		HOSRecordedAt:    fixedNow.Unix(),
	}}

	items := h.queue(t).Items
	require.Equal(t, []services.SuggestionKind{services.SuggestionHoursOfService}, kinds(items))
	assert.Equal(t, "Tom Avery runs out of hours before Nashville", items[0].Title)
	assert.Equal(t, "1:35 of drive time left on S2610-0601.", items[0].Reason)

	h.caps.caps.HOS = false
	assert.Empty(t, h.queue(t).Items, "nothing to say without an ELD")
}

func TestDetention_OffersToApprovePendingOccurrencesOnly(t *testing.T) {
	t.Parallel()

	h := newHarness(tenant.OperationTypeAsset)
	pending := &detention.DetentionOccurrence{
		ID: pulid.MustNew("dto_"), ShipmentID: pulid.MustNew("shp_"), Status: detention.OccurrenceStatusPending,
		BillableMinutes: 95, BillableAmount: decimal.RequireFromString("142.50"),
		LocationName: "Cargill Protein Plant", ShipmentProNumber: "S2610-0398",
	}
	accruing := &detention.DetentionOccurrence{ID: pulid.MustNew("dto_"), Status: detention.OccurrenceStatusAccruing}
	h.desk.entries = []*detentionservice.DeskEntry{{Occurrence: accruing}, {Occurrence: pending}}

	items := h.queue(t).Items
	require.Len(t, items, 1)
	assert.Equal(t, "Approve detention at Cargill Protein Plant", items[0].Title)
	assert.Equal(t, "1h 35m of billable time on S2610-0398 is waiting for approval.", items[0].Reason)
	assert.Equal(t, "Approve $143", items[0].Primary.Label)
	assert.Equal(t, pending.ID, items[0].Primary.DetentionOccurrenceID)
}

func TestRetender_FlagsTendersThatNeedAPerson(t *testing.T) {
	t.Parallel()

	h := newHarness(tenant.OperationTypeBrokerage)
	move := uncovered("S2610-0415", 5*time.Hour)
	move.LiveTender = &dispatchconsoleservice.MoveTenderSummary{Status: tender.StatusNeedsReview}
	h.console.board.Moves = []*dispatchconsoleservice.BoardMove{move}

	items := h.queue(t).Items
	require.Equal(t, []services.SuggestionKind{services.SuggestionRetender}, kinds(items),
		"a load already out to tender is not offered for tendering again")
}

func TestDecisions_HideDoneMoveLaterBackAndCountTheShift(t *testing.T) {
	t.Parallel()

	h := newHarness(tenant.OperationTypeBrokerage)
	first := uncovered("FIRST", time.Hour)
	second := uncovered("SECOND", 2*time.Hour)
	third := uncovered("THIRD", 3*time.Hour)
	h.console.board.Moves = []*dispatchconsoleservice.BoardMove{first, second, third}

	decide := func(move *dispatchconsoleservice.BoardMove, decision shipmentsuggestion.Decision) {
		require.NoError(t, h.service.Decide(t.Context(), &services.DecideSuggestionRequest{
			TenantInfo: testTenant(),
			Key:        "tender:" + move.MoveID.String(),
			Decision:   decision,
		}))
	}
	decide(first, shipmentsuggestion.DecisionDone)
	decide(second, shipmentsuggestion.DecisionLater)
	h.decisions.records["coverage:"+pulid.MustNew("smv_").String()] = &shipmentsuggestion.DecisionRecord{
		SuggestionKey: "stale", Decision: shipmentsuggestion.DecisionDone,
		DecidedAt: fixedNow.Add(-13 * time.Hour).Unix(),
	}

	got := h.queue(t)
	require.Len(t, got.Items, 2)
	assert.Equal(t, "THIRD", got.Items[0].ProNumber)
	assert.False(t, got.Items[0].Deferred)
	assert.Equal(t, "SECOND", got.Items[1].ProNumber)
	assert.True(t, got.Items[1].Deferred)
	assert.Equal(t, 1, got.HandledThisShift, "only this shift's done items count")
	assert.Equal(t, 2, h.decisions.pruned)
}

func TestDecide_RefusesKeysTheBoardNeverIssued(t *testing.T) {
	t.Parallel()

	h := newHarness(tenant.OperationTypeAsset)
	for _, key := range []string{"", "coverage", "made-up:smv_01JSNAPSHOT000000000000000", "coverage:short"} {
		err := h.service.Decide(t.Context(), &services.DecideSuggestionRequest{
			TenantInfo: testTenant(), Key: key, Decision: shipmentsuggestion.DecisionDone,
		})
		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation, key)
	}
}

func doneDecision(h *harness, key string, at time.Time) {
	h.decisions.records[key] = &shipmentsuggestion.DecisionRecord{
		SuggestionKey: key, Decision: shipmentsuggestion.DecisionDone, DecidedAt: at.Unix(),
	}
}

func TestUndo_CancelsTheTenderTheSuggestionStarted(t *testing.T) {
	t.Parallel()

	h := newHarness(tenant.OperationTypeBrokerage)
	moveID := pulid.MustNew("smv_")
	key := "tender:" + moveID.String()
	doneDecision(h, key, fixedNow.Add(-time.Minute))
	h.tenders.live = &tender.Tender{CreatedAt: fixedNow.Add(-70 * time.Second).Unix()}

	require.NoError(t, h.service.Undo(t.Context(), &services.UndoSuggestionRequest{TenantInfo: testTenant(), Key: key}))

	assert.Equal(t, []pulid.ID{moveID}, h.guard.canceled)
	assert.Empty(t, h.assigner.unassigned)
	assert.NotContains(t, h.decisions.records, key)
}

func TestUndo_TakesOffTheDriverTheSuggestionAssigned(t *testing.T) {
	t.Parallel()

	h := newHarness(tenant.OperationTypeAsset)
	moveID := pulid.MustNew("smv_")
	key := "coverage:" + moveID.String()
	doneDecision(h, key, fixedNow.Add(-time.Minute))
	h.reads.assignment = &shipment.Assignment{CreatedAt: fixedNow.Add(-65 * time.Second).Unix()}

	require.NoError(t, h.service.Undo(t.Context(), &services.UndoSuggestionRequest{TenantInfo: testTenant(), Key: key}))

	assert.Equal(t, []pulid.ID{moveID}, h.assigner.unassigned)
	assert.Empty(t, h.guard.canceled)
}

func TestUndo_LeavesAloneWorkItDidNotDo(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_")
	key := "coverage:" + moveID.String()

	olderAssignment := newHarness(tenant.OperationTypeAsset)
	doneDecision(olderAssignment, key, fixedNow.Add(-time.Minute))
	olderAssignment.reads.assignment = &shipment.Assignment{CreatedAt: fixedNow.Add(-2 * time.Hour).Unix()}
	require.NoError(t, olderAssignment.service.Undo(t.Context(), &services.UndoSuggestionRequest{TenantInfo: testTenant(), Key: key}))
	assert.Empty(t, olderAssignment.assigner.unassigned, "an assignment someone made earlier is not ours to undo")

	tooLate := newHarness(tenant.OperationTypeAsset)
	doneDecision(tooLate, key, fixedNow.Add(-time.Hour))
	tooLate.reads.assignment = &shipment.Assignment{CreatedAt: fixedNow.Add(-time.Hour).Unix()}
	require.NoError(t, tooLate.service.Undo(t.Context(), &services.UndoSuggestionRequest{TenantInfo: testTenant(), Key: key}))
	assert.Empty(t, tooLate.assigner.unassigned, "past the undo window only the mark is taken back")

	later := newHarness(tenant.OperationTypeAsset)
	later.decisions.records[key] = &shipmentsuggestion.DecisionRecord{
		SuggestionKey: key, Decision: shipmentsuggestion.DecisionLater, DecidedAt: fixedNow.Unix(),
	}
	later.reads.assignment = &shipment.Assignment{CreatedAt: fixedNow.Unix()}
	require.NoError(t, later.service.Undo(t.Context(), &services.UndoSuggestionRequest{TenantInfo: testTenant(), Key: key}))
	assert.Empty(t, later.assigner.unassigned, "putting something off did nothing to undo")
	assert.NotContains(t, later.decisions.records, key)
}

func TestNarration_KeepsOnlyRewordingsThatInventNothing(t *testing.T) {
	t.Parallel()

	honest := &services.ShipmentSuggestion{
		Key: "a", Title: "Assign Gwen Grant to San Antonio → Dallas",
		Reason: "5 mi out. Pickup closes at 13:30.", Impact: []string{"$3590"},
	}
	invented := &services.ShipmentSuggestion{
		Key: "b", Title: "Tender San Antonio → Dallas", Reason: "Pickup closes at 13:30.",
		Impact: []string{"$3590"},
	}

	applied := applyNarration([]*services.ShipmentSuggestion{honest, invented}, []narratedItem{
		{Key: "a", Title: "Put Gwen on the San Antonio run", Reason: "She is 5 miles out and the $3590 load closes at 13:30."},
		{Key: "b", Title: "Tender this one", Reason: "It pays $4200 and closes at 13:30."},
	})

	assert.True(t, applied)
	assert.Equal(t, "Put Gwen on the San Antonio run", honest.Title)
	assert.Equal(t, "Tender San Antonio → Dallas", invented.Title, "a made-up figure keeps the computed wording")
}
