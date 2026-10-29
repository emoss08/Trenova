package inboundqueue

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func waitingState(id pulid.ID) *inboundmessageports.QueueState {
	return &inboundmessageports.QueueState{
		ID:             id,
		OrganizationID: inboundqueuetest.QueueTenant.OrgID,
		BusinessUnitID: inboundqueuetest.QueueTenant.BuID,
		Status:         inboundmessage.StatusInReview,
		Classification: inboundmessage.ClassificationTender,
		ReceivedAt:     inboundqueuetest.TestNow - 3600,
		DueAt:          inboundqueuetest.TestNow + 3600,
		FromAddress:    "ops@northwind.example",
		SenderDomain:   "northwind.example",
		ThreadKey:      "thread-" + id.String(),
	}
}

func newID(prefix string) pulid.ID { return pulid.MustNew(prefix) }

func TestValidateSnooze(t *testing.T) {
	t.Parallel()

	docType := newID("dt_")
	cases := []struct {
		name string
		req  SnoozeRequest
		ok   bool
	}{
		{
			name: "until a time",
			req:  SnoozeRequest{Until: inboundqueuetest.TestNow + 3600},
			ok:   true,
		},
		{name: "nothing", req: SnoozeRequest{}},
		{name: "both", req: SnoozeRequest{
			Until: inboundqueuetest.TestNow + 60, Event: inboundmessage.SnoozeEventInvoicePaid,
		}},
		{name: "in the past", req: SnoozeRequest{Until: inboundqueuetest.TestNow - 1}},
		{name: "beyond ninety days", req: SnoozeRequest{
			Until: inboundqueuetest.TestNow + inboundmessage.MaxSnoozeSeconds + 1,
		}},
		{name: "exactly ninety days", req: SnoozeRequest{
			Until: inboundqueuetest.TestNow + inboundmessage.MaxSnoozeSeconds,
		}, ok: true},
		{name: "unknown event", req: SnoozeRequest{Event: "Lunch"}},
		{name: "document without a type", req: SnoozeRequest{
			Event: inboundmessage.SnoozeEventDocumentAttached,
		}},
		{name: "document with a type", req: SnoozeRequest{
			Event: inboundmessage.SnoozeEventDocumentAttached, DocumentTypeID: docType,
		}, ok: true},
		{name: "a type without a document", req: SnoozeRequest{
			Event: inboundmessage.SnoozeEventShipmentDelivered, DocumentTypeID: docType,
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSnooze(&tc.req, inboundqueuetest.TestNow)
			if tc.ok {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
		})
	}
}

func TestValidateAwaitUntil(t *testing.T) {
	t.Parallel()

	assert.NoError(
		t,
		ValidateAwaitUntil(
			inboundqueuetest.TestNow+inboundmessage.MaxAwaitSeconds,
			inboundqueuetest.TestNow,
		),
	)
	assert.Error(
		t,
		ValidateAwaitUntil(
			inboundqueuetest.TestNow+inboundmessage.MaxAwaitSeconds+1,
			inboundqueuetest.TestNow,
		),
	)
	assert.Error(t, ValidateAwaitUntil(inboundqueuetest.TestNow, inboundqueuetest.TestNow))
}

func TestSnooze_ParksAndSchedulesTheWakeUp(t *testing.T) {
	t.Parallel()

	id := newID("imsg_")
	actor := newID("usr_")
	svc, fakes := withQueue(&testService{}, waitingState(id))

	result, err := svc.queue().Snooze(t.Context(), &SnoozeRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		ActorID:    actor,
		Selection:  Selection{IDs: []pulid.ID{id}},
		Until:      inboundqueuetest.TestNow + 7200,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Affected)

	state := fakes.Queue.State(id)
	assert.Equal(t, inboundmessage.ParkingSnoozed, state.Parked)
	assert.Equal(t, inboundqueuetest.TestNow+7200, state.SnoozedUntil)
	assert.Equal(t, []pulid.ID{id}, fakes.Wakes.Scheduled)
	require.Len(t, fakes.Activities.Recorded, 1)
	activity := fakes.Activities.Recorded[0]
	assert.Equal(t, triage.ActivitySnoozed, activity.Kind)
	assert.Equal(t, triage.ActorPerson, activity.ActorKind)
	assert.Equal(t, actor, activity.ActorID)
	assert.Equal(t, inboundqueuetest.TestNow+7200, activity.Until)
}

func TestSnooze_RefusesASettledMessageAndAnEventWithNoShipment(t *testing.T) {
	t.Parallel()

	settled := waitingState(newID("imsg_"))
	settled.Status = inboundmessage.StatusActioned
	svc, fakes := withQueue(&testService{}, settled)

	_, err := svc.queue().Snooze(t.Context(), &SnoozeRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		Selection:  Selection{IDs: []pulid.ID{settled.ID}},
		Until:      inboundqueuetest.TestNow + 60,
	})
	require.Error(t, err)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)

	unlinked := waitingState(newID("imsg_"))
	fakes.Queue.States[unlinked.ID] = unlinked
	_, err = svc.queue().Snooze(t.Context(), &SnoozeRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		Selection:  Selection{IDs: []pulid.ID{unlinked.ID}},
		Event:      inboundmessage.SnoozeEventShipmentDelivered,
	})
	require.Error(t, err)
	assert.Empty(t, fakes.Queue.Updates, "nothing was written")
}

func TestSnooze_WatchesTheMatchedShipmentForAnEvent(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.MatchedShipmentID = newID("shp_")
	svc, fakes := withQueue(&testService{}, state)

	_, err := svc.queue().Snooze(t.Context(), &SnoozeRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
		Event:      inboundmessage.SnoozeEventShipmentDelivered,
	})
	require.NoError(t, err)

	parked := fakes.Queue.State(state.ID)
	assert.Equal(t, state.MatchedShipmentID, parked.SnoozeWatchID)
	assert.Empty(t, fakes.Wakes.Scheduled, "an event snooze has no timer")
}

func TestWakeIfDue_IsIdempotent(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.Parked = inboundmessage.ParkingSnoozed
	state.SnoozedUntil = inboundqueuetest.TestNow + 600
	svc, fakes := withQueue(&testService{}, state)

	early, err := svc.queue().
		WakeIfDue(t.Context(), inboundqueuetest.QueueTenant, state.ID, inboundqueuetest.TestNow)
	require.NoError(t, err)
	assert.False(t, early.Woken)
	assert.True(t, early.Schedule.Open)
	assert.Equal(t, inboundqueuetest.TestNow+600, early.Schedule.WakeAt)

	woken, err := svc.queue().
		WakeIfDue(t.Context(), inboundqueuetest.QueueTenant, state.ID, inboundqueuetest.TestNow+600)
	require.NoError(t, err)
	assert.True(t, woken.Woken)
	assert.Empty(t, fakes.Queue.State(state.ID).Parked)

	again, err := svc.queue().
		WakeIfDue(t.Context(), inboundqueuetest.QueueTenant, state.ID, inboundqueuetest.TestNow+601)
	require.NoError(t, err)
	assert.False(t, again.Woken)
	assert.False(t, again.Schedule.Open, "a woken message has nothing left to wait for")
	assert.Equal(t, []triage.ActivityKind{triage.ActivityWoken}, fakes.Activities.Kinds())
}

func TestWakeIfDue_StopsWaitingOnAReplyThatNeverCame(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.Parked = inboundmessage.ParkingAwaitingReply
	state.AwaitingUntil = inboundqueuetest.TestNow
	svc, fakes := withQueue(&testService{}, state)

	result, err := svc.queue().
		WakeIfDue(t.Context(), inboundqueuetest.QueueTenant, state.ID, inboundqueuetest.TestNow)
	require.NoError(t, err)
	assert.True(t, result.Woken)
	assert.Equal(t, []triage.ActivityKind{triage.ActivityStoppedWaiting}, fakes.Activities.Kinds())
	assert.Equal(t, triage.ActorSystem, fakes.Activities.Recorded[0].ActorKind)
}

func TestReconcileOverdueWakes_WakesWhatALostTimerMissed(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.Parked = inboundmessage.ParkingSnoozed
	state.SnoozedUntil = inboundqueuetest.TestNow - 7200
	svc, fakes := withQueue(&testService{}, state)
	fakes.Queue.Overdue = []inboundmessageports.ParkedRef{{
		ID: state.ID, OrganizationID: inboundqueuetest.QueueTenant.OrgID, BusinessUnitID: inboundqueuetest.QueueTenant.BuID,
	}}

	woken, err := svc.queue().ReconcileOverdueWakes(t.Context(), inboundqueuetest.TestNow)
	require.NoError(t, err)
	assert.Equal(t, 1, woken)
	assert.Empty(t, fakes.Queue.State(state.ID).Parked)
}

func TestResolveSelection_IsBounded(t *testing.T) {
	t.Parallel()

	svc, fakes := withQueue(&testService{})
	ids := make([]pulid.ID, inboundmessageports.MaxBulkSelection+1)
	for idx := range ids {
		ids[idx] = newID("imsg_")
	}

	_, err := svc.queue().
		resolveSelection(t.Context(), inboundqueuetest.QueueTenant, Selection{IDs: ids})
	require.Error(t, err)

	_, err = svc.queue().resolveSelection(t.Context(), inboundqueuetest.QueueTenant, Selection{})
	require.Error(t, err)

	fakes.Queue.Selected = ids[:3]
	got, err := svc.queue().resolveSelection(t.Context(), inboundqueuetest.QueueTenant, Selection{
		Folder: &inboundmessageports.ListInboundMessagesRequest{
			Lane: inboundmessageports.LaneWaiting,
		},
	})
	require.NoError(t, err)
	assert.Len(t, got, 3)
	require.NotNil(t, fakes.Queue.SelectReq)
	assert.Equal(t, inboundmessageports.MaxBulkSelection, fakes.Queue.SelectReq.Limit)
	assert.Equal(t, inboundqueuetest.QueueTenant, fakes.Queue.SelectReq.Folder.Filter.TenantInfo,
		"the folder is always read inside the caller's tenant")

	deduped, err := svc.queue().
		resolveSelection(t.Context(), inboundqueuetest.QueueTenant, Selection{
			IDs: []pulid.ID{ids[0], ids[0], ids[1]},
		})
	require.NoError(t, err)
	assert.Len(t, deduped, 2)
}

func TestReviewMany_SettlesRecordsAndCancelsTimers(t *testing.T) {
	t.Parallel()

	waiting := waitingState(newID("imsg_"))
	snoozed := waitingState(newID("imsg_"))
	snoozed.Parked = inboundmessage.ParkingSnoozed
	snoozed.SnoozedUntil = inboundqueuetest.TestNow + 600
	done := waitingState(newID("imsg_"))
	done.Status = inboundmessage.StatusActioned
	actor := newID("usr_")
	svc, fakes := withQueue(&testService{}, waiting, snoozed, done)

	result, err := svc.queue().ReviewMany(t.Context(), &BulkReviewRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		ActorID:    actor,
		Selection:  Selection{IDs: []pulid.ID{waiting.ID, snoozed.ID, done.ID}},
		Status:     inboundmessage.StatusActioned,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Affected, "a message already handled is not handled again")
	assert.Nil(t, result.RuleID)
	assert.Equal(t, 1, fakes.DB.Txs, "one transaction for the whole selection")
	require.Len(t, fakes.Queue.Updates, 1)
	assert.Len(t, fakes.Queue.Updates[0].IDs, 2, "one statement for the whole selection")
	assert.Equal(t, []pulid.ID{snoozed.ID}, fakes.Wakes.Cancelled)
	assert.Equal(t,
		[]triage.ActivityKind{triage.ActivityHandled, triage.ActivityHandled},
		fakes.Activities.Kinds(),
	)
	assert.Zero(t, fakes.Queue.State(waiting.ID).DueAt, "a settled message is due nowhere")

	_, err = svc.queue().ReviewMany(t.Context(), &BulkReviewRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		Selection:  Selection{IDs: []pulid.ID{waiting.ID}},
		Status:     inboundmessage.StatusInReview,
	})
	require.Error(t, err, "only handled and ignored are a person's to choose")
}

type stubTeacher struct {
	taught []inboundmessageports.TeachRequest
}

func (s *stubTeacher) Teach(
	_ context.Context,
	req inboundmessageports.TeachRequest,
) (*triage.Rule, error) {
	s.taught = append(s.taught, req)

	return &triage.Rule{ID: "trule_01J0000000000000000000000T"}, nil
}

func TestReviewMany_TeachesARuleFromTheMessage(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.MailboxID = newID("imbx_")
	teacher := &stubTeacher{}
	svc, fakes := withQueue(&testService{ruleTeacher: teacher}, state)

	result, err := svc.queue().ReviewMany(t.Context(), &BulkReviewRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
		Status:     inboundmessage.StatusIgnored,
		Teach:      inboundmessage.NovaActionIgnore,
	})
	require.NoError(t, err)
	require.NotNil(t, result.RuleID)
	require.Len(t, teacher.taught, 1)
	assert.Equal(t, inboundmessageports.TeachRequest{
		TenantInfo:     inboundqueuetest.QueueTenant,
		MessageID:      state.ID,
		Classification: inboundmessage.ClassificationTender,
		SenderDomain:   "northwind.example",
		MailboxID:      state.MailboxID,
		Action:         inboundmessage.NovaActionIgnore,
	}, teacher.taught[0])
	assert.Contains(t, fakes.Activities.Kinds(), triage.ActivityRuleCreated)
}

func TestReopen_PutsSettledMessagesBackWithAFreshTarget(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.Status = inboundmessage.StatusIgnored
	state.DueAt = 0
	svc, fakes := withQueue(&testService{}, state)
	fakes.Targets.Rows = []*triage.ResponseTarget{{
		Classification: inboundmessage.ClassificationTender, Minutes: 30,
	}}

	result, err := svc.queue().Reopen(t.Context(), &ReopenRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Affected)

	reopened := fakes.Queue.State(state.ID)
	assert.Equal(t, inboundmessage.StatusInReview, reopened.Status)
	assert.Equal(t, state.ReceivedAt+30*60, reopened.DueAt)
	assert.Equal(t, []triage.ActivityKind{triage.ActivityReopened}, fakes.Activities.Kinds())
}

type stubDirectory struct {
	users []permissionports.PermittedUser
}

func (s *stubDirectory) ListUsersWithPermission(
	_ context.Context,
	_ permissionports.ListUsersWithPermissionRequest,
) ([]permissionports.PermittedUser, error) {
	return s.users, nil
}

func TestAssign_NeedsTheAssignPermissionToGiveWorkToSomeoneElse(t *testing.T) {
	t.Parallel()

	me := newID("usr_")
	other := newID("usr_")
	state := waitingState(newID("imsg_"))
	directory := &stubDirectory{users: []permissionports.PermittedUser{
		{UserID: me, Name: "Ada Admin"}, {UserID: other, Name: "Marcus Lee"},
	}}
	svc, fakes := withQueue(&testService{directory: directory}, state)

	_, err := svc.queue().Assign(t.Context(), &AssignRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		ActorID:    me,
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
		OwnerID:    other,
	})
	var denied *errortypes.AuthorizationError
	require.ErrorAs(t, err, &denied)

	result, err := svc.queue().Assign(t.Context(), &AssignRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		ActorID:    me,
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
		OwnerID:    me,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Affected)
	assert.Equal(t, me, fakes.Queue.State(state.ID).OwnerID)
	require.Len(t, fakes.Activities.Recorded, 1)
	assert.Equal(t, "Ada Admin", fakes.Activities.Recorded[0].Subject)

	fakes.Queue.States[state.ID].OwnerID = other
	_, err = svc.queue().Assign(t.Context(), &AssignRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		ActorID:    me,
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
	})
	require.ErrorAs(t, err, &denied, "taking someone else off a message is assigning it")

	_, err = svc.queue().Assign(t.Context(), &AssignRequest{
		TenantInfo:      inboundqueuetest.QueueTenant,
		ActorID:         me,
		Selection:       Selection{IDs: []pulid.ID{state.ID}},
		OwnerID:         newID("usr_"),
		CanAssignOthers: true,
	})
	require.Error(t, err, "only someone who can work Triage can be given a message")
}

func TestHandOff_ReassignsEveryWaitingMessageInBatches(t *testing.T) {
	t.Parallel()

	onCall := newID("usr_")
	directory := &stubDirectory{users: []permissionports.PermittedUser{
		{UserID: onCall, Name: "Marcus Lee"},
	}}
	states := make([]*inboundmessageports.QueueState, 0, inboundmessageports.HandOffBatchSize+3)
	ids := make([]pulid.ID, 0, cap(states))
	for range inboundmessageports.HandOffBatchSize + 3 {
		state := waitingState(newID("imsg_"))
		states = append(states, state)
		ids = append(ids, state.ID)
	}
	parked := states[0]
	parked.Parked = inboundmessage.ParkingSnoozed
	svc, fakes := withQueue(&testService{directory: directory}, states...)
	fakes.Queue.HandOffLeft = ids

	result, err := svc.queue().HandOff(t.Context(), &HandOffRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		ActorID:    newID("usr_"),
		ToUserID:   onCall,
	})
	require.NoError(t, err)
	assert.Equal(t, len(ids)-1, result.Affected, "a parked message is not handed off")
	assert.Equal(t, onCall, result.ToUserID)
	assert.Len(t, fakes.Queue.Updates, 2, "handed off in bounded batches")
	for _, update := range fakes.Queue.Updates {
		assert.LessOrEqual(t, len(update.IDs), inboundmessageports.HandOffBatchSize)
		require.NotNil(t, update.Patch.HandOff)
		assert.Equal(t, onCall, update.Patch.HandOff.To)
	}
	assert.Len(t, fakes.Activities.Recorded, len(ids)-1)
	assert.Equal(t, triage.ActivityHandedOff, fakes.Activities.Recorded[0].Kind)
	assert.Equal(t, onCall, fakes.Queue.State(states[1].ID).OwnerID)
	assert.True(t, fakes.Queue.State(parked.ID).OwnerID.IsNil())

	_, err = svc.queue().HandOff(t.Context(), &HandOffRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		ToUserID:   newID("usr_"),
	})
	require.Error(t, err)
}

func TestUnlink_ClearsTheReasonWithTheLastRecord(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.MatchedShipmentID = newID("shp_")
	state.MatchReason = "PRO in the subject"
	assert.True(t, UnlinkPatch(state, LinkedShipment).ClearReason)

	state.MatchedCustomerID = newID("cus_")
	patch := UnlinkPatch(state, LinkedShipment)
	assert.True(t, patch.ClearShipment)
	assert.False(t, patch.ClearReason, "the customer link still needs its reason")
}

func TestWaitForReply_ParksUntilTheSenderAnswers(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	repo := &inboundqueuetest.SettleRepo{Message: &inboundmessage.InboundMessage{ID: state.ID}}
	svc, fakes := withQueue(&testService{messageRepo: repo}, state)

	_, err := svc.queue().WaitForReply(t.Context(), &WaitForReplyRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		MessageID:  state.ID,
		Until:      inboundqueuetest.TestNow + 86400,
	})
	require.NoError(t, err)
	parked := fakes.Queue.State(state.ID)
	assert.Equal(t, inboundmessage.ParkingAwaitingReply, parked.Parked)
	assert.Equal(t, inboundqueuetest.TestNow+86400, parked.AwaitingUntil)
	assert.Equal(t, []pulid.ID{state.ID}, fakes.Wakes.Scheduled)

	_, err = svc.queue().StopWaiting(t.Context(), &MessageActionRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		MessageID:  state.ID,
	})
	require.NoError(t, err)
	assert.Empty(t, fakes.Queue.State(state.ID).Parked)
	assert.Equal(t, []pulid.ID{state.ID}, fakes.Wakes.Cancelled)

	_, err = svc.queue().
		Wake(t.Context(), &MessageActionRequest{TenantInfo: inboundqueuetest.QueueTenant, MessageID: state.ID})
	require.Error(t, err, "a message that is not snoozed cannot be woken")
}

type recordingTower struct {
	watchtowerports.WatchtowerProjector

	upserted []string
	resolved []string
}

func (r *recordingTower) Upsert(_ context.Context, input watchtowerports.WatchtowerItemInput) {
	r.upserted = append(r.upserted, input.SourceID)
}

func (r *recordingTower) Resolve(
	_ context.Context,
	_ pagination.TenantInfo,
	_ watchtower.SourceKind,
	sourceID string,
) {
	r.resolved = append(r.resolved, sourceID)
}

type recordingBriefing struct {
	invalidated []pagination.TenantInfo
}

func (r *recordingBriefing) InvalidateBriefing(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) {
	r.invalidated = append(r.invalidated, tenantInfo)
}

func TestNotifier_PublishesProjectsAndDropsTheBriefingOnce(t *testing.T) {
	t.Parallel()

	realtime := &inboundqueuetest.RecordingRealtime{}
	tower := &recordingTower{}
	briefing := &recordingBriefing{}
	notifier := newNotifier(notifierDeps{
		logger:     zap.NewNop(),
		realtime:   realtime,
		watchtower: tower,
		briefing:   briefing,
	})

	waiting := inboundmessageports.MessageFromState(waitingState(newID("imsg_")))
	settled := inboundmessageports.MessageFromState(waitingState(newID("imsg_")))
	settled.Status = inboundmessage.StatusActioned
	notifier.MessagesChanged(t.Context(), waiting, settled)

	assert.Len(t, realtime.Published, 2)
	assert.Equal(t, []string{waiting.ID.String()}, tower.upserted)
	assert.Equal(t, []string{settled.ID.String()}, tower.resolved)
	assert.Len(t, briefing.invalidated, 1, "one briefing per tenant, however many messages")

	many := make([]*inboundmessage.InboundMessage, 0, maxRecordInvalidations+1)
	for range maxRecordInvalidations + 1 {
		many = append(many, inboundmessageports.MessageFromState(waitingState(newID("imsg_"))))
	}
	realtime.Published = nil
	notifier.MessagesChanged(t.Context(), many...)
	require.Len(t, realtime.Published, 1, "a large change is one invalidation for the inbox")
	assert.True(t, realtime.Published[0].RecordID.IsNil())
}

func TestEventSink_WakesTheMessagesWaitingOnTheEvent(t *testing.T) {
	t.Parallel()

	shipmentID := newID("shp_")
	docType := newID("dt_")
	podWait := waitingState(newID("imsg_"))
	podWait.Parked = inboundmessage.ParkingSnoozed
	podWait.SnoozeEvent = inboundmessage.SnoozeEventDocumentAttached
	podWait.SnoozeWatchID = shipmentID
	podWait.SnoozeDocumentTypeID = docType
	paidWait := waitingState(newID("imsg_"))
	paidWait.Parked = inboundmessage.ParkingSnoozed
	paidWait.SnoozeEvent = inboundmessage.SnoozeEventInvoicePaid
	paidWait.SnoozeWatchID = shipmentID

	_, fakes := withQueue(&testService{}, podWait, paidWait)
	realtime := &inboundqueuetest.RecordingRealtime{}
	sink := &EventSink{
		l:            zap.NewNop(),
		db:           fakes.DB,
		queueRepo:    fakes.Queue,
		activityRepo: fakes.Activities,
		notifier:     newNotifier(notifierDeps{logger: zap.NewNop(), realtime: realtime}),
		now:          func() int64 { return inboundqueuetest.TestNow },
	}

	sink.DocumentAttached(t.Context(), inboundqueuetest.QueueTenant, shipmentID, newID("dt_"))
	assert.Equal(t, inboundmessage.ParkingSnoozed, fakes.Queue.State(podWait.ID).Parked,
		"a different document leaves it snoozed")

	sink.DocumentAttached(t.Context(), inboundqueuetest.QueueTenant, shipmentID, docType)
	assert.Empty(t, fakes.Queue.State(podWait.ID).Parked)
	assert.Equal(t, inboundmessage.ParkingSnoozed, fakes.Queue.State(paidWait.ID).Parked)
	require.Len(t, fakes.Activities.Recorded, 1)
	assert.Equal(t, triage.ActivityWoken, fakes.Activities.Recorded[0].Kind)
	assert.Equal(t, shipmentID, fakes.Activities.Recorded[0].RecordID)
	assert.Len(t, realtime.Published, 1)

	sink.InvoicePaid(t.Context(), inboundqueuetest.QueueTenant, shipmentID)
	assert.Empty(t, fakes.Queue.State(paidWait.ID).Parked)

	sink.ShipmentDelivered(t.Context(), inboundqueuetest.QueueTenant, pulid.Nil)
	assert.Len(t, fakes.Queue.WakeEvents, 3, "an event naming no shipment asks nothing")
}

func TestUpdateResponseTargets_ValidatesAndRecomputes(t *testing.T) {
	t.Parallel()

	svc, fakes := withQueue(&testService{})
	minutes := 90
	views, err := svc.queue().UpdateResponseTargets(t.Context(), &UpdateTargetsRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		Changes: []TargetChange{
			{Classification: inboundmessage.ClassificationTender, Minutes: &minutes},
			{Classification: inboundmessage.ClassificationInvoice},
		},
	})
	require.NoError(t, err)
	require.Len(t, fakes.Targets.Saved, 1)
	assert.Len(t, fakes.Targets.Saved[0].Upsert, 1)
	assert.Equal(t,
		[]inboundmessage.Classification{inboundmessage.ClassificationInvoice},
		fakes.Targets.Saved[0].Reset,
	)
	assert.Equal(t, [][]inboundmessage.Classification{{
		inboundmessage.ClassificationTender, inboundmessage.ClassificationInvoice,
	}}, fakes.Wakes.Recompute)
	assert.Len(t, views, len(inboundmessage.AllClassifications()))

	tooShort := 1
	_, err = svc.queue().UpdateResponseTargets(t.Context(), &UpdateTargetsRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		Changes: []TargetChange{
			{Classification: inboundmessage.ClassificationTender, Minutes: &tooShort},
		},
	})
	require.Error(t, err)

	_, err = svc.queue().UpdateResponseTargets(t.Context(), &UpdateTargetsRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		Changes: []TargetChange{
			{Classification: inboundmessage.ClassificationTender, Minutes: &minutes},
			{Classification: inboundmessage.ClassificationTender},
		},
	})
	require.Error(t, err, "a kind may be set once per change")
}

func TestResponseTargetViews_MarksTheDefaults(t *testing.T) {
	t.Parallel()

	views := responseTargetViews([]*triage.ResponseTarget{{
		Classification: inboundmessage.ClassificationProofOfDelivery, Minutes: 240,
	}})
	for _, view := range views {
		if view.Classification == inboundmessage.ClassificationProofOfDelivery {
			assert.Equal(t, 240, view.Minutes)
			assert.False(t, view.IsDefault)
			continue
		}
		assert.True(t, view.IsDefault)
		assert.Equal(t, triage.DefaultTargetMinutes(view.Classification), view.Minutes)
	}
}
