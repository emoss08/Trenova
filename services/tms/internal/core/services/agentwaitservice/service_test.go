package agentwaitservice

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/geofence"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentwaitjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const now int64 = 1_800_000_000

type memoryWaits struct {
	repositories.AgentWaitRepository

	items   map[pulid.ID]*agentwait.Wait
	resumed []repositories.MarkAgentWaitResumedRequest
}

func newMemoryWaits(items ...*agentwait.Wait) *memoryWaits {
	waits := &memoryWaits{items: make(map[pulid.ID]*agentwait.Wait, len(items))}
	for _, item := range items {
		waits.items[item.ID] = item
	}

	return waits
}

func (m *memoryWaits) Insert(_ context.Context, entity *agentwait.Wait) (*agentwait.Wait, error) {
	entity.ID = pulid.MustNew(agentwait.IDPrefix)
	m.items[entity.ID] = entity

	return entity, nil
}

func (m *memoryWaits) Get(_ context.Context, req *repositories.GetAgentWaitRequest) (*agentwait.Wait, error) {
	wait, ok := m.items[req.ID]
	if !ok || (req.ThreadID.IsNotNil() && wait.ThreadID != req.ThreadID) {
		return nil, errortypes.NewNotFoundError("Wait not found")
	}

	return wait, nil
}

func (m *memoryWaits) ListOpenWatching(
	_ context.Context,
	req *repositories.ListOpenWaitsWatchingRequest,
) ([]*agentwait.Wait, error) {
	out := make([]*agentwait.Wait, 0, len(m.items))
	for _, wait := range m.items {
		for _, kind := range req.Kinds {
			for _, id := range req.WatchIDs {
				if wait.Status.Open() && wait.Kind == kind && wait.WatchID == id {
					out = append(out, wait)
				}
			}
		}
	}

	return out, nil
}

func (m *memoryWaits) ListOpenOfKinds(
	_ context.Context,
	req *repositories.ListOpenWaitsOfKindsRequest,
) ([]*agentwait.Wait, error) {
	out := make([]*agentwait.Wait, 0, len(m.items))
	for _, wait := range m.items {
		if wait.Status.Open() && slices.Contains(req.Kinds, wait.Kind) {
			out = append(out, wait)
		}
	}

	return out, nil
}

func (m *memoryWaits) ListOpenByThreads(
	_ context.Context,
	req *repositories.ListOpenWaitsByThreadsRequest,
) (map[pulid.ID][]*agentwait.Wait, error) {
	out := make(map[pulid.ID][]*agentwait.Wait, len(req.ThreadIDs))
	for _, wait := range m.items {
		if wait.Status.Open() && slices.Contains(req.ThreadIDs, wait.ThreadID) {
			out[wait.ThreadID] = append(out[wait.ThreadID], wait)
		}
	}

	return out, nil
}

func (m *memoryWaits) ListOverdueAcrossTenants(
	_ context.Context,
	req *repositories.ListOverdueWaitsRequest,
) ([]*agentwait.Wait, error) {
	out := make([]*agentwait.Wait, 0, len(m.items))
	for _, wait := range m.items {
		if wait.Status.Open() && wait.ExpiresAt < req.ExpiredBefore {
			out = append(out, wait)
		}
	}

	return out, nil
}

func (m *memoryWaits) UpdateCondition(
	_ context.Context,
	req *repositories.UpdateAgentWaitConditionRequest,
) error {
	m.items[req.ID].Condition = req.Condition

	return nil
}

func (m *memoryWaits) SetDue(_ context.Context, req *repositories.SetAgentWaitDueRequest) error {
	if wait, ok := m.items[req.ID]; ok {
		wait.DueAt = req.DueAt
		if req.WorkflowID != "" {
			wait.WorkflowID = req.WorkflowID
		}
	}

	return nil
}

func (m *memoryWaits) Resolve(
	_ context.Context,
	req *repositories.ResolveAgentWaitRequest,
) (*agentwait.Wait, bool, error) {
	wait := m.items[req.ID]
	if !wait.Status.Open() {
		return wait, false, nil
	}
	wait.Status = req.Status
	wait.Outcome = req.Outcome

	return wait, true, nil
}

func (m *memoryWaits) MarkResumed(_ context.Context, req *repositories.MarkAgentWaitResumedRequest) error {
	m.resumed = append(m.resumed, *req)
	wait := m.items[req.ID]
	wait.ResumedTurnID = req.TurnID
	wait.ResumedRunID = req.RunID

	return nil
}

type fakeMoves struct {
	repositories.ShipmentMoveRepository

	move *shipment.ShipmentMove
}

func (f *fakeMoves) GetByID(context.Context, *repositories.GetMoveByIDRequest) (*shipment.ShipmentMove, error) {
	if f.move == nil {
		return nil, errortypes.NewNotFoundError("Move not found")
	}

	return f.move, nil
}

type fakeHOS struct {
	repositories.TelematicsRepository

	state *telematics.WorkerHOSState
}

func (f *fakeHOS) GetWorkerHOSState(
	context.Context,
	repositories.GetWorkerHOSStateRequest,
) (*telematics.WorkerHOSState, error) {
	if f.state == nil {
		return nil, errortypes.NewNotFoundError("No clock")
	}

	return f.state, nil
}

type fakeRun struct{ id string }

func (r fakeRun) GetID() string                  { return r.id }
func (r fakeRun) GetRunID() string               { return "run" }
func (r fakeRun) GetFirstExecutionRunID() string { return "run" }
func (r fakeRun) Get(context.Context, any) error { return nil }
func (r fakeRun) GetWithOptions(context.Context, any, client.WorkflowRunGetOptions) error {
	return nil
}

type signal struct {
	workflowID string
	met        agentwaitjobs.Met
}

type fakeWorkflows struct {
	serviceports.WorkflowStarter

	started  []string
	signals  []signal
	signalFn func(workflowID string) error
}

func (f *fakeWorkflows) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	_ ...any,
) (client.WorkflowRun, error) {
	f.started = append(f.started, options.ID)

	return fakeRun{id: options.ID}, nil
}

func (f *fakeWorkflows) SignalWorkflow(_ context.Context, workflowID, _, _ string, arg any) error {
	if f.signalFn != nil {
		if err := f.signalFn(workflowID); err != nil {
			return err
		}
	}
	met, _ := arg.(agentwaitjobs.Met)
	f.signals = append(f.signals, signal{workflowID: workflowID, met: met})

	return nil
}

type fakeTurns struct {
	active  *conversation.AssistantTurn
	started []assistantturnservice.StartRequest
}

func (f *fakeTurns) Active(context.Context, repositories.ActiveAssistantTurnRequest) (*conversation.AssistantTurn, error) {
	return f.active, nil
}

func (f *fakeTurns) StartTurn(
	_ context.Context,
	req assistantturnservice.StartRequest,
	start func(turn *conversation.AssistantTurn) (string, error),
) (*conversation.AssistantTurn, error) {
	turn := &conversation.AssistantTurn{ID: pulid.MustNew("atrn_"), ThreadID: req.ThreadID}
	if _, err := start(turn); err != nil {
		return nil, err
	}
	f.started = append(f.started, req)

	return turn, nil
}

type fakeConversations struct {
	repositories.ConversationRepository
}

func (fakeConversations) GetThreadOwned(
	_ context.Context,
	req repositories.GetThreadOwnedRequest,
) (*conversation.Thread, error) {
	return &conversation.Thread{
		ID:             req.ID,
		UserID:         req.TenantInfo.UserID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}, nil
}

type fakeRuns struct {
	started []*serviceports.StartAgentRunForDefinitionRequest
	err     error
}

func (f *fakeRuns) StartForDefinition(
	_ context.Context,
	req *serviceports.StartAgentRunForDefinitionRequest,
	_ *serviceports.RequestActor,
) (*agent.AgentRun, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.started = append(f.started, req)

	return &agent.AgentRun{ID: pulid.MustNew("ar_")}, nil
}

type harness struct {
	svc       *Service
	waits     *memoryWaits
	moves     *fakeMoves
	hos       *fakeHOS
	workflows *fakeWorkflows
	turns     *fakeTurns
	runs      *fakeRuns
	actor     *serviceports.RequestActor
}

func newHarness(items ...*agentwait.Wait) *harness {
	h := &harness{
		waits:     newMemoryWaits(items...),
		moves:     &fakeMoves{},
		hos:       &fakeHOS{},
		workflows: &fakeWorkflows{},
		turns:     &fakeTurns{},
		runs:      &fakeRuns{},
		actor: &serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeUser,
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
	}
	h.svc = &Service{
		l:             zap.NewNop(),
		waits:         h.waits,
		facts:         &facts{moves: h.moves, telematics: h.hos},
		conversations: fakeConversations{},
		turns:         h.turns,
		runService:    h.runs,
		workflows:     h.workflows,
		now:           func() int64 { return now },
	}

	return h
}

func (h *harness) register(kind agentwait.Kind, condition agentwait.Condition) (*agentwait.Wait, error) {
	return h.svc.Register(context.Background(), &serviceports.RegisterWaitRequest{
		Actor:        h.actor,
		DefinitionID: pulid.MustNew("agdef_"),
		ThreadID:     pulid.MustNew("athr_"),
		Kind:         kind,
		Condition:    condition,
		Description:  "Truck 2214 to reach Kroger DC",
	})
}

func moveWithStop(arrived *int64) (*shipment.ShipmentMove, *shipment.Stop) {
	stop := &shipment.Stop{ID: pulid.MustNew("stp_"), ScheduledWindowStart: now + 4*3600, ActualArrival: arrived}
	move := &shipment.ShipmentMove{
		ID:     pulid.MustNew("smv_"),
		Status: shipment.MoveStatusInTransit,
		Stops:  []*shipment.Stop{stop},
	}
	stop.ShipmentMoveID = move.ID

	return move, stop
}

func TestRegister_WaitsOnAStopAndStartsItsWorkflow(t *testing.T) {
	t.Parallel()

	h := newHarness()
	move, stop := moveWithStop(nil)
	h.moves.move = move

	wait, err := h.register(agentwait.KindStopArrival, agentwait.Condition{
		ShipmentMoveID: move.ID,
		StopID:         stop.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, move.ID, wait.WatchID)
	assert.Equal(t, now+agentwait.DefaultLifetimeSecs, wait.ExpiresAt)
	assert.Equal(t, []string{agentwaitjobs.WorkflowIDFor(wait.ID)}, h.workflows.started)
	assert.Equal(t, agentwaitjobs.WorkflowIDFor(wait.ID), wait.WorkflowID)
}

func TestRegister_RefusesToWaitForWhatHasHappened(t *testing.T) {
	t.Parallel()

	arrived := now - 600
	h := newHarness()
	move, stop := moveWithStop(&arrived)
	h.moves.move = move

	_, err := h.register(agentwait.KindStopArrival, agentwait.Condition{
		ShipmentMoveID: move.ID,
		StopID:         stop.ID,
	})

	require.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "nothing to wait for")
	assert.Empty(t, h.waits.items, "nothing is recorded")
	assert.Empty(t, h.workflows.started)
}

func TestRegister_AnAppointmentComesDueBeforeItsWindow(t *testing.T) {
	t.Parallel()

	h := newHarness()
	move, stop := moveWithStop(nil)
	h.moves.move = move

	wait, err := h.register(agentwait.KindAppointmentNear, agentwait.Condition{
		ShipmentMoveID: move.ID,
		StopID:         stop.ID,
		MinutesBefore:  90,
	})

	require.NoError(t, err)
	require.NotNil(t, wait.DueAt)
	assert.Equal(t, stop.ScheduledWindowStart-90*60, *wait.DueAt)
}

func TestRegister_DriveTimeAlreadyBelowIsRefused(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.hos.state = &telematics.WorkerHOSState{DriveRemainingMs: 90 * 60 * 1000}

	_, err := h.register(agentwait.KindHOSDriveBelow, agentwait.Condition{
		WorkerID:          pulid.MustNew("wrk_"),
		DriveMinutesBelow: 120,
	})

	require.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "1h 30m")
}

func TestRegister_ADelegatedTaskCannotPark(t *testing.T) {
	t.Parallel()

	h := newHarness()
	_, err := h.svc.Register(context.Background(), &serviceports.RegisterWaitRequest{
		Actor:     h.actor,
		Delegated: true,
		Kind:      agentwait.KindTime,
	})

	assert.True(t, errortypes.IsBusinessError(err))
}

func openWait(kind agentwait.Kind, condition agentwait.Condition) *agentwait.Wait {
	wait := &agentwait.Wait{
		ID:                pulid.MustNew(agentwait.IDPrefix),
		OrganizationID:    pulid.MustNew("org_"),
		BusinessUnitID:    pulid.MustNew("bu_"),
		AgentDefinitionID: pulid.MustNew("agdef_"),
		ThreadID:          pulid.MustNew("athr_"),
		UserID:            pulid.MustNew("usr_"),
		Kind:              kind,
		Condition:         &condition,
		Description:       "Truck to reach the dock",
		Status:            agentwait.StatusWaiting,
	}
	wait.Normalize()
	wait.WorkflowID = agentwaitjobs.WorkflowIDFor(wait.ID)

	return wait
}

func TestNotifyEvent_EndsOnlyTheWaitOnThatStop(t *testing.T) {
	t.Parallel()

	move := pulid.MustNew("smv_")
	here, there := pulid.MustNew("stp_"), pulid.MustNew("stp_")
	atHere := openWait(agentwait.KindStopArrival, agentwait.Condition{ShipmentMoveID: move, StopID: here})
	atThere := openWait(agentwait.KindStopArrival, agentwait.Condition{ShipmentMoveID: move, StopID: there})
	anyStop := openWait(agentwait.KindStopArrival, agentwait.Condition{ShipmentMoveID: move})
	leaving := openWait(agentwait.KindStopDeparture, agentwait.Condition{ShipmentMoveID: move})
	h := newHarness(atHere, atThere, anyStop, leaving)

	h.svc.NotifyEvent(context.Background(), &serviceports.AgentEvent{
		Kind:       agent.EventShipmentMoveArrived,
		SubjectID:  move,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Related:    []pulid.ID{here},
		Detail:     "Arrived at stop.",
	})

	signalled := make([]string, 0, len(h.workflows.signals))
	for _, s := range h.workflows.signals {
		signalled = append(signalled, s.workflowID)
		assert.Equal(t, "Arrived at stop.", s.met.Detail)
	}
	assert.ElementsMatch(t, []string{atHere.WorkflowID, anyStop.WorkflowID}, signalled)
}

func TestNotifyEvent_AReplyMatchedToTheCarrierEndsItsWait(t *testing.T) {
	t.Parallel()

	carrier := pulid.MustNew("car_")
	waiting := openWait(agentwait.KindReply, agentwait.Condition{CarrierID: carrier})
	h := newHarness(waiting)

	h.svc.NotifyEvent(context.Background(), &serviceports.AgentEvent{
		Kind:       agent.EventInboundMessageClassified,
		SubjectID:  pulid.MustNew("inm_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Related:    []pulid.ID{pulid.MustNew("shp_"), carrier},
	})

	require.Len(t, h.workflows.signals, 1)
	assert.Equal(t, waiting.WorkflowID, h.workflows.signals[0].workflowID)
}

func TestNotifyHOS_EndsTheWaitOnceDriveTimeDropsBelow(t *testing.T) {
	t.Parallel()

	worker := pulid.MustNew("wrk_")
	waiting := openWait(agentwait.KindHOSDriveBelow, agentwait.Condition{WorkerID: worker, DriveMinutesBelow: 120})
	h := newHarness(waiting)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	h.svc.NotifyHOS(context.Background(), tenant, []*telematics.WorkerHOSState{
		{WorkerID: worker, DriveRemainingMs: 3 * 60 * 60 * 1000, RecordedAt: now},
	})
	assert.Empty(t, h.workflows.signals, "three hours left is not below two")

	h.svc.NotifyHOS(context.Background(), tenant, []*telematics.WorkerHOSState{
		{WorkerID: worker, DriveRemainingMs: 105 * 60 * 1000, RecordedAt: now},
	})
	require.Len(t, h.workflows.signals, 1)
	assert.Contains(t, h.workflows.signals[0].met.Detail, "1h 45m")
}

func finishInput(wait *agentwait.Wait, status agentwait.Status) *agentwaitjobs.FinishInput {
	payload := &agentwaitjobs.Payload{WaitID: wait.ID}
	payload.OrganizationID = wait.OrganizationID
	payload.BusinessUnitID = wait.BusinessUnitID

	return &agentwaitjobs.FinishInput{Payload: payload, Status: status, Outcome: "Arrived."}
}

func TestFinish_PicksTheConversationUpAsItsOwnerOnce(t *testing.T) {
	t.Parallel()

	wait := openWait(agentwait.KindTime, agentwait.Condition{At: now})
	h := newHarness(wait)

	require.NoError(t, h.svc.Finish(context.Background(), finishInput(wait, agentwait.StatusMet)))
	require.NoError(t, h.svc.Finish(context.Background(), finishInput(wait, agentwait.StatusMet)))

	require.Len(t, h.turns.started, 1, "a retried finish starts no second turn")
	assert.Equal(t, conversation.AssistantTurnOriginWaitResolved, h.turns.started[0].Origin)
	assert.Equal(t, wait.UserID, h.turns.started[0].UserID)
	assert.Equal(t, agentwait.StatusMet, wait.Status)
	assert.Equal(t, "Arrived.", wait.Outcome)
}

func TestFinish_ABusyConversationIsTriedAgain(t *testing.T) {
	t.Parallel()

	wait := openWait(agentwait.KindTime, agentwait.Condition{At: now})
	h := newHarness(wait)
	h.turns.active = &conversation.AssistantTurn{ID: pulid.MustNew("atrn_")}

	err := h.svc.Finish(context.Background(), finishInput(wait, agentwait.StatusMet))
	require.ErrorIs(t, err, agentwaitjobs.ErrConversationBusy)
	assert.Equal(t, agentwait.StatusMet, wait.Status, "the wait is closed; only the pick-up waits")

	h.turns.active = nil
	require.NoError(t, h.svc.Finish(context.Background(), finishInput(wait, agentwait.StatusMet)))
	assert.Len(t, h.turns.started, 1)
}

func TestFinish_ABackgroundRunPicksUpOnTheSameRecord(t *testing.T) {
	t.Parallel()

	wait := openWait(agentwait.KindTime, agentwait.Condition{At: now})
	wait.ThreadID, wait.UserID = pulid.Nil, pulid.Nil
	wait.RunID = pulid.MustNew("ar_")
	wait.SubjectType = agent.SubjectShipmentMove
	wait.SubjectID = pulid.MustNew("smv_")
	h := newHarness(wait)

	require.NoError(t, h.svc.Finish(context.Background(), finishInput(wait, agentwait.StatusTimedOut)))

	require.Len(t, h.runs.started, 1)
	started := h.runs.started[0]
	assert.Equal(t, agent.RunTriggerWait, started.Trigger)
	assert.Equal(t, wait.ID, started.WaitID)
	assert.Equal(t, wait.SubjectID, started.SubjectID)
	assert.Equal(t, wait.AgentDefinitionID, started.DefinitionID)
}

func TestFinish_ACancelledWaitPicksNothingUp(t *testing.T) {
	t.Parallel()

	wait := openWait(agentwait.KindTime, agentwait.Condition{At: now})
	wait.Status = agentwait.StatusCancelled
	h := newHarness(wait)

	require.NoError(t, h.svc.Finish(context.Background(), finishInput(wait, agentwait.StatusMet)))
	assert.Empty(t, h.turns.started)
}

func TestCancel_ABackgroundRunReachesOnlyItsOwnAgentsWaits(t *testing.T) {
	t.Parallel()

	wait := openWait(agentwait.KindTime, agentwait.Condition{At: now})
	h := newHarness(wait)
	tenant := pagination.TenantInfo{OrgID: wait.OrganizationID, BuID: wait.BusinessUnitID}

	_, err := h.svc.Cancel(context.Background(), &serviceports.CancelWaitRequest{
		ID:           wait.ID,
		TenantInfo:   tenant,
		DefinitionID: pulid.MustNew("agdef_"),
	})
	require.True(t, errortypes.IsNotFoundError(err))
	assert.True(t, wait.Status.Open())
}

func TestRegister_ADrivingDriverComesDueWhenTheClockCrosses(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.hos.state = &telematics.WorkerHOSState{
		DutyStatus:       telematics.DutyStatusDriving,
		DriveRemainingMs: 5 * 60 * 60 * 1000,
		RecordedAt:       now - 600,
	}

	wait, err := h.register(agentwait.KindHOSDriveBelow, agentwait.Condition{
		WorkerID:          pulid.MustNew("wrk_"),
		DriveMinutesBelow: 120,
	})

	require.NoError(t, err)
	require.NotNil(t, wait.DueAt)
	assert.Equal(t, now-600+3*60*60, *wait.DueAt,
		"three hours of driving from the clock's own reading takes it to two left")
}

func TestRegister_AClockThatRanDownSinceItWasReadIsAlreadyBelow(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.hos.state = &telematics.WorkerHOSState{
		DutyStatus:       telematics.DutyStatusDriving,
		DriveRemainingMs: 125 * 60 * 1000,
		RecordedAt:       now - 10*60,
	}

	_, err := h.register(agentwait.KindHOSDriveBelow, agentwait.Condition{
		WorkerID:          pulid.MustNew("wrk_"),
		DriveMinutesBelow: 120,
	})

	require.True(t, errortypes.IsBusinessError(err), "125 minutes read ten minutes ago is 115 now")
}

func TestNotifyHOS_MovesTheDueTimeAndTellsTheWorkflow(t *testing.T) {
	t.Parallel()

	worker := pulid.MustNew("wrk_")
	waiting := openWait(agentwait.KindHOSDriveBelow, agentwait.Condition{WorkerID: worker, DriveMinutesBelow: 60})
	h := newHarness(waiting)
	tenant := pagination.TenantInfo{OrgID: waiting.OrganizationID, BuID: waiting.BusinessUnitID}

	h.svc.NotifyHOS(context.Background(), tenant, []*telematics.WorkerHOSState{{
		WorkerID: worker, DutyStatus: telematics.DutyStatusDriving,
		DriveRemainingMs: 2 * 60 * 60 * 1000, RecordedAt: now,
	}})
	require.NotNil(t, waiting.DueAt)
	assert.Equal(t, now+60*60, *waiting.DueAt)
	require.Len(t, h.workflows.signals, 1, "the workflow is told to read its new due time")

	h.svc.NotifyHOS(context.Background(), tenant, []*telematics.WorkerHOSState{{
		WorkerID: worker, DutyStatus: telematics.DutyStatusDriving,
		DriveRemainingMs: 2 * 60 * 60 * 1000, RecordedAt: now,
	}})
	assert.Len(t, h.workflows.signals, 1, "an unchanged due time is not signalled again")

	h.svc.NotifyHOS(context.Background(), tenant, []*telematics.WorkerHOSState{{
		WorkerID: worker, DutyStatus: telematics.DutyStatus("on_duty"),
		DriveRemainingMs: 2 * 60 * 60 * 1000, RecordedAt: now + 60,
	}})
	assert.Nil(t, waiting.DueAt, "a driver not driving has no crossing to time")
}

func TestCheck_ProjectsDriveTimeFromTheLastClockRead(t *testing.T) {
	t.Parallel()

	worker := pulid.MustNew("wrk_")
	waiting := openWait(agentwait.KindHOSDriveBelow, agentwait.Condition{WorkerID: worker, DriveMinutesBelow: 60})
	h := newHarness(waiting)
	h.hos.state = &telematics.WorkerHOSState{
		DutyStatus: telematics.DutyStatusDriving, DriveRemainingMs: 90 * 60 * 1000, RecordedAt: now - 31*60,
	}
	tenant := pagination.TenantInfo{OrgID: waiting.OrganizationID, BuID: waiting.BusinessUnitID}

	check, err := h.svc.Check(context.Background(), tenant, waiting.ID, now)
	require.NoError(t, err)
	assert.True(t, check.Met, "90 minutes read 31 minutes ago while driving is 59 now")
	assert.Contains(t, check.Detail, "projected")
}

func movingTo(stop *shipment.Stop, location *location.Location) *shipment.ShipmentMove {
	stop.Location = location
	return &shipment.ShipmentMove{
		ID:     stop.ShipmentMoveID,
		Status: shipment.MoveStatusInTransit,
		Stops:  []*shipment.Stop{stop},
	}
}

func krogerDC() *location.Location {
	lat, lon := 36.3729, -94.2088
	return &location.Location{
		Name:         "Kroger DC",
		Latitude:     &lat,
		Longitude:    &lon,
		GeofenceType: geofence.TypeAuto,
	}
}

type fakeAssignments struct {
	repositories.AssignmentRepository

	tractor pulid.ID
}

func (f *fakeAssignments) GetByMoveID(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*shipment.Assignment, error) {
	tractor := f.tractor
	return &shipment.Assignment{TractorID: &tractor}, nil
}

func TestNotifyPositions_ATruckInsideTheStopsGeofenceArrives(t *testing.T) {
	t.Parallel()

	stop := &shipment.Stop{ID: pulid.MustNew("stp_"), ShipmentMoveID: pulid.MustNew("smv_")}
	move := movingTo(stop, krogerDC())
	waiting := openWait(agentwait.KindStopArrival, agentwait.Condition{ShipmentMoveID: move.ID})
	h := newHarness(waiting)
	h.moves.move = move
	tractor := pulid.MustNew("trc_")
	h.svc.facts.assignments = &fakeAssignments{tractor: tractor}
	tenant := pagination.TenantInfo{OrgID: waiting.OrganizationID, BuID: waiting.BusinessUnitID}

	h.svc.NotifyPositions(context.Background(), tenant, []*telematics.VehiclePosition{
		{TractorID: tractor, Latitude: 36.40, Longitude: -94.2088, RecordedAt: now},
	})
	assert.Empty(t, h.workflows.signals, "three kilometres out is not there yet")

	h.svc.NotifyPositions(context.Background(), tenant, []*telematics.VehiclePosition{
		{TractorID: tractor, Latitude: 36.3731, Longitude: -94.2088, RecordedAt: now},
	})
	require.Len(t, h.workflows.signals, 1)
	assert.Contains(t, h.workflows.signals[0].met.Detail, "Kroger DC")
}

func TestNotifyPositions_LeavingIsBeingOutsideAfterBeingInside(t *testing.T) {
	t.Parallel()

	stop := &shipment.Stop{ID: pulid.MustNew("stp_"), ShipmentMoveID: pulid.MustNew("smv_")}
	move := movingTo(stop, krogerDC())
	waiting := openWait(agentwait.KindStopDeparture, agentwait.Condition{
		ShipmentMoveID: move.ID, StopID: stop.ID,
	})
	h := newHarness(waiting)
	h.moves.move = move
	tractor := pulid.MustNew("trc_")
	h.svc.facts.assignments = &fakeAssignments{tractor: tractor}
	tenant := pagination.TenantInfo{OrgID: waiting.OrganizationID, BuID: waiting.BusinessUnitID}
	outside := []*telematics.VehiclePosition{{TractorID: tractor, Latitude: 36.40, Longitude: -94.2088, RecordedAt: now}}

	h.svc.NotifyPositions(context.Background(), tenant, outside)
	assert.Empty(t, h.workflows.signals, "outside before ever being inside is not leaving")

	h.svc.NotifyPositions(context.Background(), tenant, []*telematics.VehiclePosition{
		{TractorID: tractor, Latitude: 36.3731, Longitude: -94.2088, RecordedAt: now},
	})
	assert.Equal(t, now, waiting.Condition.SeenInsideAt)
	assert.Empty(t, h.workflows.signals)

	h.svc.NotifyPositions(context.Background(), tenant, outside)
	require.Len(t, h.workflows.signals, 1)
	assert.Contains(t, h.workflows.signals[0].met.Detail, "has left")
}

func TestNotifyPositions_AStalePositionSaysNothing(t *testing.T) {
	t.Parallel()

	stop := &shipment.Stop{ID: pulid.MustNew("stp_"), ShipmentMoveID: pulid.MustNew("smv_")}
	move := movingTo(stop, krogerDC())
	waiting := openWait(agentwait.KindStopArrival, agentwait.Condition{ShipmentMoveID: move.ID})
	h := newHarness(waiting)
	h.moves.move = move
	tractor := pulid.MustNew("trc_")
	h.svc.facts.assignments = &fakeAssignments{tractor: tractor}

	h.svc.NotifyPositions(context.Background(),
		pagination.TenantInfo{OrgID: waiting.OrganizationID, BuID: waiting.BusinessUnitID},
		[]*telematics.VehiclePosition{
			{TractorID: tractor, Latitude: 36.3731, Longitude: -94.2088, RecordedAt: now - positionFreshFor - 1},
		})

	assert.Empty(t, h.workflows.signals)
}

func TestMeet_ASecondNoticeOfAWaitAlreadyEndedPicksNothingUp(t *testing.T) {
	t.Parallel()

	waiting := openWait(agentwait.KindStopArrival, agentwait.Condition{ShipmentMoveID: pulid.MustNew("smv_")})
	waiting.Status = agentwait.StatusMet
	h := newHarness(waiting)
	h.workflows.signalFn = func(string) error { return serviceerror.NewNotFound("workflow completed") }

	h.svc.meet(context.Background(), waiting, "Arrived again.")

	assert.Empty(t, h.turns.started, "the workflow that ended it already picked it up")
}

func TestMeet_AnOpenWaitWhoseWorkflowIsGoneIsPickedUpHere(t *testing.T) {
	t.Parallel()

	waiting := openWait(agentwait.KindStopArrival, agentwait.Condition{ShipmentMoveID: pulid.MustNew("smv_")})
	h := newHarness(waiting)
	h.workflows.signalFn = func(string) error { return serviceerror.NewNotFound("no such workflow") }

	h.svc.meet(context.Background(), waiting, "Arrived.")

	assert.Equal(t, agentwait.StatusMet, waiting.Status)
	assert.Len(t, h.turns.started, 1)
}

func TestReconcileOverdue_ClosesAndPicksUpOnlyWaitsPastTheirExpiry(t *testing.T) {
	t.Parallel()

	lost := openWait(agentwait.KindTime, agentwait.Condition{At: now - 7200})
	lost.ExpiresAt = now - overdueGrace - 60
	recent := openWait(agentwait.KindTime, agentwait.Condition{At: now - 60})
	recent.ExpiresAt = now - 60
	h := newHarness(lost, recent)

	closed, err := h.svc.ReconcileOverdue(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, closed)
	assert.Equal(t, agentwait.StatusTimedOut, lost.Status)
	assert.True(t, recent.Status.Open(), "a wait just past its expiry is still its workflow's")
	assert.Len(t, h.turns.started, 1)
}
