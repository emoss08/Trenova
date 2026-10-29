package inboundqueue

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A plan is the write it previews: the same checks, the same eligible messages and
the same patch, applied to a copy. Each test plans first, proves nothing was
written, then makes the write and holds the saved state to what the plan said.
*/

func requirePlannedThenSaved(
	t *testing.T,
	fakes *inboundqueuetest.QueueFakes,
	plan *QueuePlan,
	ids ...pulid.ID,
) {
	t.Helper()

	require.Len(t, plan.Changes, len(ids))
	for idx, id := range ids {
		assert.Equal(t, id, plan.Changes[idx].Before.ID)
		assert.Equal(t, plan.Changes[idx].After, fakes.Queue.State(id),
			"the saved message is what the plan said it would be")
	}
}

func TestPlanSnooze_IsWhatSnoozeSaves(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.MatchedShipmentID = newID("shp_")
	svc, fakes := withQueue(&testService{}, state)
	req := &SnoozeRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		ActorID:    newID("usr_"),
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
		Event:      inboundmessage.SnoozeEventShipmentDelivered,
	}

	plan, err := svc.queue().PlanSnooze(t.Context(), req)
	require.NoError(t, err)
	assert.Empty(t, fakes.Queue.Updates, "a plan writes nothing")
	assert.Empty(t, fakes.Activities.Recorded)
	require.Len(t, plan.Changes, 1)
	assert.Equal(t, inboundmessage.ParkingSnoozed, plan.Changes[0].After.Parked)
	assert.Equal(t, state.MatchedShipmentID, plan.Changes[0].After.SnoozeWatchID)
	assert.Empty(t, plan.Changes[0].Before.Parked)

	_, err = svc.queue().Snooze(t.Context(), req)
	require.NoError(t, err)
	requirePlannedThenSaved(t, fakes, plan, state.ID)
}

func TestPlanSnooze_RefusesWhatSnoozeRefuses(t *testing.T) {
	t.Parallel()

	settled := waitingState(newID("imsg_"))
	settled.Status = inboundmessage.StatusActioned
	svc, fakes := withQueue(&testService{}, settled)

	_, err := svc.queue().PlanSnooze(t.Context(), &SnoozeRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		Selection:  Selection{IDs: []pulid.ID{settled.ID}},
		Until:      inboundqueuetest.TestNow + 60,
	})
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)

	_, err = svc.queue().PlanSnooze(t.Context(), &SnoozeRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		Selection:  Selection{IDs: []pulid.ID{settled.ID}},
		Until:      inboundqueuetest.TestNow - 60,
	})
	require.Error(t, err, "a snooze has to end in the future")
	assert.Empty(t, fakes.Queue.Updates)
}

func TestPlanReopen_LeavesWaitingMessagesAloneAndCountsThem(t *testing.T) {
	t.Parallel()

	settled := waitingState(newID("imsg_"))
	settled.Status = inboundmessage.StatusIgnored
	settled.DueAt = 0
	waiting := waitingState(newID("imsg_"))
	svc, fakes := withQueue(&testService{}, settled, waiting)
	fakes.Targets.Rows = []*triage.ResponseTarget{{
		Classification: inboundmessage.ClassificationTender, Minutes: 30,
	}}
	req := &ReopenRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		Selection:  Selection{IDs: []pulid.ID{settled.ID, waiting.ID, newID("imsg_")}},
	}

	plan, err := svc.queue().PlanReopen(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Unchanged, "a message still waiting is not reopened")
	assert.Equal(t, 1, plan.Missing, "a message not in the organization is counted, not changed")
	require.Len(t, plan.Changes, 1)
	assert.Equal(t, inboundmessage.StatusInReview, plan.Changes[0].After.Status)
	assert.Equal(t, settled.ReceivedAt+30*60, plan.Changes[0].After.DueAt)
	assert.Empty(t, fakes.Queue.Updates)

	_, err = svc.queue().Reopen(t.Context(), req)
	require.NoError(t, err)
	requirePlannedThenSaved(t, fakes, plan, settled.ID)
}

func TestPlanAssign_IsWhatAssignSaves(t *testing.T) {
	t.Parallel()

	owner := newID("usr_")
	state := waitingState(newID("imsg_"))
	directory := &stubDirectory{users: []permissionports.PermittedUser{
		{UserID: owner, Name: "Marcus Lee"},
	}}
	svc, fakes := withQueue(&testService{directory: directory}, state)
	req := &AssignRequest{
		TenantInfo:      inboundqueuetest.QueueTenant,
		ActorID:         newID("usr_"),
		Selection:       Selection{IDs: []pulid.ID{state.ID}},
		OwnerID:         owner,
		CanAssignOthers: true,
	}

	plan, err := svc.queue().PlanAssign(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, plan.Changes, 1)
	assert.Equal(t, owner, plan.Changes[0].After.OwnerID)
	assert.Empty(t, fakes.Queue.Updates)

	_, err = svc.queue().Assign(t.Context(), req)
	require.NoError(t, err)
	requirePlannedThenSaved(t, fakes, plan, state.ID)

	_, err = svc.queue().PlanAssign(t.Context(), &AssignRequest{
		TenantInfo:      inboundqueuetest.QueueTenant,
		Selection:       Selection{IDs: []pulid.ID{state.ID}},
		OwnerID:         newID("usr_"),
		CanAssignOthers: true,
	})
	require.Error(t, err, "only someone who can work Triage can be given a message")
}

func TestPlanWakeAndStopWaiting_EachTakesOnlyItsOwnParking(t *testing.T) {
	t.Parallel()

	snoozed := waitingState(newID("imsg_"))
	snoozed.Parked = inboundmessage.ParkingSnoozed
	snoozed.SnoozedUntil = inboundqueuetest.TestNow + 600
	awaiting := waitingState(newID("imsg_"))
	awaiting.Parked = inboundmessage.ParkingAwaitingReply
	awaiting.AwaitingUntil = inboundqueuetest.TestNow + 600
	repo := &inboundqueuetest.SettleRepo{Message: &inboundmessage.InboundMessage{ID: snoozed.ID}}
	svc, fakes := withQueue(&testService{messageRepo: repo}, snoozed, awaiting)

	wake := &MessageActionRequest{TenantInfo: inboundqueuetest.QueueTenant, MessageID: snoozed.ID}
	stop := &MessageActionRequest{TenantInfo: inboundqueuetest.QueueTenant, MessageID: awaiting.ID}

	_, err := svc.queue().PlanStopWaiting(t.Context(), wake)
	require.Error(t, err, "a snoozed message is not waiting on a reply")
	_, err = svc.queue().PlanWake(t.Context(), stop)
	require.Error(t, err, "a message waiting on a reply is not snoozed")

	woken, err := svc.queue().PlanWake(t.Context(), wake)
	require.NoError(t, err)
	require.Len(t, woken.Changes, 1)
	assert.Empty(t, woken.Changes[0].After.Parked)
	assert.Zero(t, woken.Changes[0].After.SnoozedUntil)

	stopped, err := svc.queue().PlanStopWaiting(t.Context(), stop)
	require.NoError(t, err)
	require.Len(t, stopped.Changes, 1)
	assert.Zero(t, stopped.Changes[0].After.AwaitingUntil)
	assert.Empty(t, fakes.Queue.Updates)

	_, err = svc.queue().Wake(t.Context(), wake)
	require.NoError(t, err)
	requirePlannedThenSaved(t, fakes, woken, snoozed.ID)

	repo.Message = &inboundmessage.InboundMessage{ID: awaiting.ID}
	_, err = svc.queue().StopWaiting(t.Context(), stop)
	require.NoError(t, err)
	requirePlannedThenSaved(t, fakes, stopped, awaiting.ID)
}

func TestPlanWaitForReply_IsWhatWaitForReplySaves(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	repo := &inboundqueuetest.SettleRepo{Message: &inboundmessage.InboundMessage{ID: state.ID}}
	svc, fakes := withQueue(&testService{messageRepo: repo}, state)
	req := &WaitForReplyRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		MessageID:  state.ID,
		Until:      inboundqueuetest.TestNow + 86400,
	}

	plan, err := svc.queue().PlanWaitForReply(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, plan.Changes, 1)
	assert.Equal(t, inboundmessage.ParkingAwaitingReply, plan.Changes[0].After.Parked)
	assert.Equal(t, inboundqueuetest.TestNow+86400, plan.Changes[0].After.AwaitingUntil)
	assert.Empty(t, fakes.Queue.Updates)

	_, err = svc.queue().WaitForReply(t.Context(), req)
	require.NoError(t, err)
	requirePlannedThenSaved(t, fakes, plan, state.ID)

	_, err = svc.queue().PlanWaitForReply(t.Context(), &WaitForReplyRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		MessageID:  state.ID,
		Until:      inboundqueuetest.TestNow + inboundmessage.MaxAwaitSeconds + 1,
	})
	require.Error(t, err, "a message waits on a reply for at most 30 days")
}

func TestPlanUnlink_IsWhatUnlinkSaves(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.MatchedShipmentID = newID("shp_")
	state.MatchReason = "PRO in the subject"
	repo := &inboundqueuetest.SettleRepo{Message: &inboundmessage.InboundMessage{ID: state.ID}}
	svc, fakes := withQueue(&testService{messageRepo: repo}, state)
	req := &UnlinkRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		MessageID:  state.ID,
		Record:     LinkedShipment,
	}

	plan, err := svc.queue().PlanUnlink(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, plan.Changes, 1)
	assert.True(t, plan.Changes[0].After.MatchedShipmentID.IsNil())
	assert.Empty(t, plan.Changes[0].After.MatchReason)
	assert.Empty(t, fakes.Queue.Updates)

	_, err = svc.queue().Unlink(t.Context(), req)
	require.NoError(t, err)
	requirePlannedThenSaved(t, fakes, plan, state.ID)

	_, err = svc.queue().PlanUnlink(t.Context(), &UnlinkRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		MessageID:  state.ID,
		Record:     LinkedCarrier,
	})
	require.Error(t, err, "a message not linked to a carrier has no carrier to unlink")
}

func TestPlanHandOff_PlansOneBatchAndSaysWhenThereIsMore(t *testing.T) {
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
	states[0].Parked = inboundmessage.ParkingSnoozed
	svc, fakes := withQueue(&testService{directory: directory}, states...)
	fakes.Queue.HandOffLeft = ids

	plan, err := svc.queue().PlanHandOff(t.Context(), &HandOffRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		ToUserID:   onCall,
	})
	require.NoError(t, err)
	assert.True(t, plan.More, "a full batch says there may be more behind it")
	assert.Len(t, plan.Changes, inboundmessageports.HandOffBatchSize-1)
	assert.Equal(t, 1, plan.Unchanged, "a parked message is not handed off")
	for _, change := range plan.Changes {
		assert.Equal(t, onCall, change.After.OwnerID)
	}
	assert.Empty(t, fakes.Queue.Updates)

	_, err = svc.queue().PlanHandOff(t.Context(), &HandOffRequest{
		TenantInfo: inboundqueuetest.QueueTenant,
		ToUserID:   newID("usr_"),
	})
	require.Error(t, err, "the queue goes only to someone who can work Triage")
}

func TestQueuePatch_ApplyTo_ParksOneWayAtATime(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.MatchedShipmentID = newID("shp_")
	state.Parked = inboundmessage.ParkingAwaitingReply
	state.AwaitingUntil = inboundqueuetest.TestNow + 600

	snooze := inboundmessageports.QueuePatch{Park: &inboundmessageports.ParkPatch{
		Parking:       inboundmessage.ParkingSnoozed,
		SnoozeEvent:   inboundmessage.SnoozeEventShipmentDelivered,
		WatchShipment: true,
	}}
	snooze.ApplyTo(state)
	assert.Equal(t, inboundmessage.ParkingSnoozed, state.Parked)
	assert.Zero(t, state.AwaitingUntil, "snoozing ends the wait on a reply")
	assert.Equal(t, state.MatchedShipmentID, state.SnoozeWatchID)

	await := inboundmessageports.QueuePatch{Park: &inboundmessageports.ParkPatch{
		Parking:       inboundmessage.ParkingAwaitingReply,
		AwaitingUntil: inboundqueuetest.TestNow + 900,
	}}
	await.ApplyTo(state)
	assert.Equal(t, inboundmessage.ParkingAwaitingReply, state.Parked)
	assert.Empty(t, state.SnoozeEvent, "waiting on a reply ends the snooze")
	assert.True(t, state.SnoozeWatchID.IsNil())
	assert.Equal(t, inboundqueuetest.TestNow+900, state.AwaitingUntil)

	note := inboundmessageports.QueuePatch{Review: &inboundmessageports.ReviewPatch{}}
	state.ReviewNote = "kept"
	note.ApplyTo(state)
	assert.Equal(t, "kept", state.ReviewNote, "an empty note leaves the one on record")
}
