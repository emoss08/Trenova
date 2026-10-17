package inboundmessageservice

import (
	"testing"

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
	fakes *queueFakes,
	plan *QueuePlan,
	ids ...pulid.ID,
) {
	t.Helper()

	require.Len(t, plan.Changes, len(ids))
	for idx, id := range ids {
		assert.Equal(t, id, plan.Changes[idx].Before.ID)
		assert.Equal(t, plan.Changes[idx].After, fakes.queue.state(id),
			"the saved message is what the plan said it would be")
	}
}

func TestPlanSnooze_IsWhatSnoozeSaves(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.MatchedShipmentID = newID("shp_")
	svc, fakes := withQueue(&Service{}, state)
	req := &SnoozeRequest{
		TenantInfo: queueTenant,
		ActorID:    newID("usr_"),
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
		Event:      inboundmessage.SnoozeEventShipmentDelivered,
	}

	plan, err := svc.PlanSnooze(t.Context(), req)
	require.NoError(t, err)
	assert.Empty(t, fakes.queue.updates, "a plan writes nothing")
	assert.Empty(t, fakes.activities.recorded)
	require.Len(t, plan.Changes, 1)
	assert.Equal(t, inboundmessage.ParkingSnoozed, plan.Changes[0].After.Parked)
	assert.Equal(t, state.MatchedShipmentID, plan.Changes[0].After.SnoozeWatchID)
	assert.Empty(t, plan.Changes[0].Before.Parked)

	_, err = svc.Snooze(t.Context(), req)
	require.NoError(t, err)
	requirePlannedThenSaved(t, fakes, plan, state.ID)
}

func TestPlanSnooze_RefusesWhatSnoozeRefuses(t *testing.T) {
	t.Parallel()

	settled := waitingState(newID("imsg_"))
	settled.Status = inboundmessage.StatusActioned
	svc, fakes := withQueue(&Service{}, settled)

	_, err := svc.PlanSnooze(t.Context(), &SnoozeRequest{
		TenantInfo: queueTenant,
		Selection:  Selection{IDs: []pulid.ID{settled.ID}},
		Until:      testNow + 60,
	})
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)

	_, err = svc.PlanSnooze(t.Context(), &SnoozeRequest{
		TenantInfo: queueTenant,
		Selection:  Selection{IDs: []pulid.ID{settled.ID}},
		Until:      testNow - 60,
	})
	require.Error(t, err, "a snooze has to end in the future")
	assert.Empty(t, fakes.queue.updates)
}

func TestPlanReopen_LeavesWaitingMessagesAloneAndCountsThem(t *testing.T) {
	t.Parallel()

	settled := waitingState(newID("imsg_"))
	settled.Status = inboundmessage.StatusIgnored
	settled.DueAt = 0
	waiting := waitingState(newID("imsg_"))
	svc, fakes := withQueue(&Service{}, settled, waiting)
	fakes.targets.rows = []*triage.ResponseTarget{{
		Classification: inboundmessage.ClassificationTender, Minutes: 30,
	}}
	req := &ReopenRequest{
		TenantInfo: queueTenant,
		Selection:  Selection{IDs: []pulid.ID{settled.ID, waiting.ID, newID("imsg_")}},
	}

	plan, err := svc.PlanReopen(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.Unchanged, "a message still waiting is not reopened")
	assert.Equal(t, 1, plan.Missing, "a message not in the organization is counted, not changed")
	require.Len(t, plan.Changes, 1)
	assert.Equal(t, inboundmessage.StatusInReview, plan.Changes[0].After.Status)
	assert.Equal(t, settled.ReceivedAt+30*60, plan.Changes[0].After.DueAt)
	assert.Empty(t, fakes.queue.updates)

	_, err = svc.Reopen(t.Context(), req)
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
	svc, fakes := withQueue(&Service{directory: directory}, state)
	req := &AssignRequest{
		TenantInfo:      queueTenant,
		ActorID:         newID("usr_"),
		Selection:       Selection{IDs: []pulid.ID{state.ID}},
		OwnerID:         owner,
		CanAssignOthers: true,
	}

	plan, err := svc.PlanAssign(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, plan.Changes, 1)
	assert.Equal(t, owner, plan.Changes[0].After.OwnerID)
	assert.Empty(t, fakes.queue.updates)

	_, err = svc.Assign(t.Context(), req)
	require.NoError(t, err)
	requirePlannedThenSaved(t, fakes, plan, state.ID)

	_, err = svc.PlanAssign(t.Context(), &AssignRequest{
		TenantInfo:      queueTenant,
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
	snoozed.SnoozedUntil = testNow + 600
	awaiting := waitingState(newID("imsg_"))
	awaiting.Parked = inboundmessage.ParkingAwaitingReply
	awaiting.AwaitingUntil = testNow + 600
	repo := &settleRepo{message: &inboundmessage.InboundMessage{ID: snoozed.ID}}
	svc, fakes := withQueue(&Service{messageRepo: repo}, snoozed, awaiting)

	wake := &MessageActionRequest{TenantInfo: queueTenant, MessageID: snoozed.ID}
	stop := &MessageActionRequest{TenantInfo: queueTenant, MessageID: awaiting.ID}

	_, err := svc.PlanStopWaiting(t.Context(), wake)
	require.Error(t, err, "a snoozed message is not waiting on a reply")
	_, err = svc.PlanWake(t.Context(), stop)
	require.Error(t, err, "a message waiting on a reply is not snoozed")

	woken, err := svc.PlanWake(t.Context(), wake)
	require.NoError(t, err)
	require.Len(t, woken.Changes, 1)
	assert.Empty(t, woken.Changes[0].After.Parked)
	assert.Zero(t, woken.Changes[0].After.SnoozedUntil)

	stopped, err := svc.PlanStopWaiting(t.Context(), stop)
	require.NoError(t, err)
	require.Len(t, stopped.Changes, 1)
	assert.Zero(t, stopped.Changes[0].After.AwaitingUntil)
	assert.Empty(t, fakes.queue.updates)

	_, err = svc.Wake(t.Context(), wake)
	require.NoError(t, err)
	requirePlannedThenSaved(t, fakes, woken, snoozed.ID)

	repo.message = &inboundmessage.InboundMessage{ID: awaiting.ID}
	_, err = svc.StopWaiting(t.Context(), stop)
	require.NoError(t, err)
	requirePlannedThenSaved(t, fakes, stopped, awaiting.ID)
}

func TestPlanWaitForReply_IsWhatWaitForReplySaves(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	repo := &settleRepo{message: &inboundmessage.InboundMessage{ID: state.ID}}
	svc, fakes := withQueue(&Service{messageRepo: repo}, state)
	req := &WaitForReplyRequest{
		TenantInfo: queueTenant,
		MessageID:  state.ID,
		Until:      testNow + 86400,
	}

	plan, err := svc.PlanWaitForReply(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, plan.Changes, 1)
	assert.Equal(t, inboundmessage.ParkingAwaitingReply, plan.Changes[0].After.Parked)
	assert.Equal(t, testNow+86400, plan.Changes[0].After.AwaitingUntil)
	assert.Empty(t, fakes.queue.updates)

	_, err = svc.WaitForReply(t.Context(), req)
	require.NoError(t, err)
	requirePlannedThenSaved(t, fakes, plan, state.ID)

	_, err = svc.PlanWaitForReply(t.Context(), &WaitForReplyRequest{
		TenantInfo: queueTenant,
		MessageID:  state.ID,
		Until:      testNow + inboundmessage.MaxAwaitSeconds + 1,
	})
	require.Error(t, err, "a message waits on a reply for at most 30 days")
}

func TestPlanUnlink_IsWhatUnlinkSaves(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.MatchedShipmentID = newID("shp_")
	state.MatchReason = "PRO in the subject"
	repo := &settleRepo{message: &inboundmessage.InboundMessage{ID: state.ID}}
	svc, fakes := withQueue(&Service{messageRepo: repo}, state)
	req := &UnlinkRequest{TenantInfo: queueTenant, MessageID: state.ID, Record: LinkedShipment}

	plan, err := svc.PlanUnlink(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, plan.Changes, 1)
	assert.True(t, plan.Changes[0].After.MatchedShipmentID.IsNil())
	assert.Empty(t, plan.Changes[0].After.MatchReason)
	assert.Empty(t, fakes.queue.updates)

	_, err = svc.Unlink(t.Context(), req)
	require.NoError(t, err)
	requirePlannedThenSaved(t, fakes, plan, state.ID)

	_, err = svc.PlanUnlink(t.Context(), &UnlinkRequest{
		TenantInfo: queueTenant,
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
	svc, fakes := withQueue(&Service{directory: directory}, states...)
	fakes.queue.handOffLeft = ids

	plan, err := svc.PlanHandOff(t.Context(), &HandOffRequest{
		TenantInfo: queueTenant,
		ToUserID:   onCall,
	})
	require.NoError(t, err)
	assert.True(t, plan.More, "a full batch says there may be more behind it")
	assert.Len(t, plan.Changes, inboundmessageports.HandOffBatchSize-1)
	assert.Equal(t, 1, plan.Unchanged, "a parked message is not handed off")
	for _, change := range plan.Changes {
		assert.Equal(t, onCall, change.After.OwnerID)
	}
	assert.Empty(t, fakes.queue.updates)

	_, err = svc.PlanHandOff(t.Context(), &HandOffRequest{
		TenantInfo: queueTenant,
		ToUserID:   newID("usr_"),
	})
	require.Error(t, err, "the queue goes only to someone who can work Triage")
}

func TestQueuePatch_ApplyTo_ParksOneWayAtATime(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.MatchedShipmentID = newID("shp_")
	state.Parked = inboundmessage.ParkingAwaitingReply
	state.AwaitingUntil = testNow + 600

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
		AwaitingUntil: testNow + 900,
	}}
	await.ApplyTo(state)
	assert.Equal(t, inboundmessage.ParkingAwaitingReply, state.Parked)
	assert.Empty(t, state.SnoozeEvent, "waiting on a reply ends the snooze")
	assert.True(t, state.SnoozeWatchID.IsNil())
	assert.Equal(t, testNow+900, state.AwaitingUntil)

	note := inboundmessageports.QueuePatch{Review: &inboundmessageports.ReviewPatch{}}
	state.ReviewNote = "kept"
	note.ApplyTo(state)
	assert.Equal(t, "kept", state.ReviewNote, "an empty note leaves the one on record")
}
