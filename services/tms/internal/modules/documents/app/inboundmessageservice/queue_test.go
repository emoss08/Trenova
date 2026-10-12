package inboundmessageservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var queueTenant = pagination.TenantInfo{OrgID: "org_01J0000000000000000000000A", BuID: "bu_01J00000000000000000000000B"}

func waitingState(id pulid.ID) *inboundmessageports.QueueState {
	return &inboundmessageports.QueueState{
		ID:             id,
		OrganizationID: queueTenant.OrgID,
		BusinessUnitID: queueTenant.BuID,
		Status:         inboundmessage.StatusInReview,
		Classification: inboundmessage.ClassificationTender,
		ReceivedAt:     testNow - 3600,
		DueAt:          testNow + 3600,
		FromAddress:    "ops@northwind.example",
		SenderDomain:   "northwind.example",
		ThreadKey:      "thread-" + id.String(),
	}
}

func newID(prefix string) pulid.ID { return pulid.MustNew(prefix) }

func TestPrepareList_SortsTheFolderByDueOrNewest(t *testing.T) {
	t.Parallel()

	due := &inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{Query: "  tender "},
		Lane:   inboundmessageports.LaneWaiting,
		Sort:   inboundmessageports.SortDue,
	}
	require.NoError(t, PrepareList(due))
	assert.Equal(t, []domaintypes.SortField{
		{Field: "dueAt", Direction: dbtype.SortDirectionAsc},
		{Field: "receivedAt", Direction: dbtype.SortDirectionDesc},
	}, due.Filter.Sort)
	assert.Equal(t, "tender", due.Filter.Query)

	plain := &inboundmessageports.ListInboundMessagesRequest{Filter: &pagination.QueryOptions{}}
	require.NoError(t, PrepareList(plain))
	assert.Equal(t, inboundmessageports.SortNewest, plain.Sort)
	assert.Equal(t,
		[]domaintypes.SortField{{Field: "receivedAt", Direction: dbtype.SortDirectionDesc}},
		plain.Filter.Sort,
	)

	thread := &inboundmessageports.ListInboundMessagesRequest{
		Filter:    &pagination.QueryOptions{},
		Sort:      inboundmessageports.SortDue,
		ThreadKey: "abc@northwind.example",
	}
	require.NoError(t, PrepareList(thread))
	assert.Equal(t, inboundmessageports.SortNewest, thread.Sort, "a thread reads newest first")
}

func TestPrepareList_RefusesAFolderItCannotAnswer(t *testing.T) {
	t.Parallel()

	assert.Error(t, PrepareList(&inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{},
		Lane:   inboundmessageports.Lane("Archive"),
	}))
	assert.Error(t, PrepareList(&inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{},
		Sort:   inboundmessageports.Sort("Oldest"),
	}))
	assert.Error(t, PrepareList(&inboundmessageports.ListInboundMessagesRequest{
		Filter: &pagination.QueryOptions{},
		Lane:   inboundmessageports.LaneAssignedToMe,
	}), "assigned to me needs a viewer")
	assert.Error(t, PrepareList(&inboundmessageports.ListInboundMessagesRequest{
		Filter:    &pagination.QueryOptions{},
		ThreadKey: strings.Repeat("x", inboundmessage.MaxThreadKeyLength+1),
	}))
}

func TestFoldCounts_SplitsWaitingParkedAndMine(t *testing.T) {
	t.Parallel()

	intake := newID("imbx_")
	counts := FoldCounts([]inboundmessageports.InboundMessageCount{
		{Status: inboundmessage.StatusInReview, Classification: inboundmessage.ClassificationTender,
			MailboxID: intake, Mine: true, Overdue: true, Count: 2},
		{Status: inboundmessage.StatusInReview, Classification: inboundmessage.ClassificationTender,
			MailboxID: intake, Count: 3},
		{Status: inboundmessage.StatusInReview, Classification: inboundmessage.ClassificationTender,
			MailboxID: intake, Parked: inboundmessage.ParkingSnoozed, Mine: true, Overdue: true, Count: 4},
		{Status: inboundmessage.StatusInReview, Classification: inboundmessage.ClassificationInvoice,
			MailboxID: intake, Parked: inboundmessage.ParkingAwaitingReply, Count: 1},
		{Status: inboundmessage.StatusQuarantined, MailboxID: intake, Count: 1},
		{Status: inboundmessage.StatusProcessing, MailboxID: intake, Count: 5},
		{Status: inboundmessage.StatusClassified, MailboxID: intake, Count: 1},
		{Status: inboundmessage.StatusActioned, Classification: inboundmessage.ClassificationTender,
			MailboxID: intake, Count: 7},
		{Status: inboundmessage.StatusIgnored, MailboxID: intake, Count: 2},
	})

	assert.Equal(t, 6, counts.Waiting, "parked messages are not waiting")
	assert.Equal(t, 4, counts.Snoozed)
	assert.Equal(t, 1, counts.WaitingOnThem)
	assert.Equal(t, 2, counts.AssignedToMe, "a snoozed message of mine is not waiting on me")
	assert.Equal(t, 2, counts.Overdue, "a snoozed message is not overdue")
	assert.Equal(t, 6, counts.Processing)
	assert.Equal(t, 7, counts.Handled)
	assert.Equal(t, 2, counts.Ignored)
	assert.Equal(t, 1, counts.Quarantined)
	assert.Equal(t, 26, counts.Total)

	for _, row := range counts.ByClassification {
		if row.Classification == inboundmessage.ClassificationTender {
			assert.Equal(t, 16, row.Total)
			assert.Equal(t, 5, row.Waiting, "a kind's waiting leaves the parked ones out")
		}
	}
	require.Len(t, counts.ByMailbox, 1)
	assert.Equal(t, 6, counts.ByMailbox[0].Waiting)
}

type stubReplies struct {
	inboundmessageports.InboundReplyReader

	keys  map[string]string
	asked [][]string
}

func (s *stubReplies) ThreadKeyForMessageKeys(
	_ context.Context,
	_ pagination.TenantInfo,
	messageKeys []string,
) (string, bool, error) {
	s.asked = append(s.asked, messageKeys)
	for _, key := range messageKeys {
		if thread, ok := s.keys[key]; ok {
			return thread, true, nil
		}
	}

	return "", false, nil
}

func TestThreadKeyFor_FollowsTheFirstKnownMessageNamed(t *testing.T) {
	t.Parallel()

	svc, fakes := withQueue(&Service{})
	fakes.queue.threadKeys["older@acme.example"] = "thread-older"
	fakes.queue.threadKeys["parent@acme.example"] = "thread-parent"

	key, err := svc.threadKeyFor(t.Context(), threadLookup{
		tenant:     queueTenant,
		ownKey:     "child@acme.example",
		inReplyTo:  "<Parent@Acme.example>",
		references: []string{"<older@acme.example>", "<parent@acme.example>"},
	})
	require.NoError(t, err)
	assert.Equal(t, "thread-parent", key, "In-Reply-To is the closest relative")

	key, err = svc.threadKeyFor(t.Context(), threadLookup{
		tenant:     queueTenant,
		references: []string{"<older@acme.example>", "<unknown@acme.example>"},
	})
	require.NoError(t, err)
	assert.Equal(t, "thread-older", key, "references are read newest first")

	key, err = svc.threadKeyFor(t.Context(), threadLookup{tenant: queueTenant})
	require.NoError(t, err)
	assert.Empty(t, key, "a message naming nothing starts its own thread")
}

func TestThreadKeyFor_PrefersOurReplyWhenItIsCloserThanAKnownMessage(t *testing.T) {
	t.Parallel()

	replies := &stubReplies{keys: map[string]string{"reply@trenova.example": "thread-from-reply"}}
	svc, fakes := withQueue(&Service{replies: replies})
	fakes.queue.threadKeys["original@acme.example"] = "thread-original"

	key, err := svc.threadKeyFor(t.Context(), threadLookup{
		tenant:     queueTenant,
		inReplyTo:  "<reply@trenova.example>",
		references: []string{"<original@acme.example>", "<reply@trenova.example>"},
	})
	require.NoError(t, err)
	assert.Equal(t, "thread-from-reply", key)
	require.Len(t, replies.asked, 1)
	assert.Equal(t, []string{"reply@trenova.example"}, replies.asked[0],
		"only the names closer than the known inbound message are asked")
}

func TestValidateSnooze(t *testing.T) {
	t.Parallel()

	docType := newID("dt_")
	cases := []struct {
		name string
		req  SnoozeRequest
		ok   bool
	}{
		{name: "until a time", req: SnoozeRequest{Until: testNow + 3600}, ok: true},
		{name: "nothing", req: SnoozeRequest{}},
		{name: "both", req: SnoozeRequest{
			Until: testNow + 60, Event: inboundmessage.SnoozeEventInvoicePaid,
		}},
		{name: "in the past", req: SnoozeRequest{Until: testNow - 1}},
		{name: "beyond ninety days", req: SnoozeRequest{
			Until: testNow + inboundmessage.MaxSnoozeSeconds + 1,
		}},
		{name: "exactly ninety days", req: SnoozeRequest{
			Until: testNow + inboundmessage.MaxSnoozeSeconds,
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
			err := ValidateSnooze(&tc.req, testNow)
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

	assert.NoError(t, ValidateAwaitUntil(testNow+inboundmessage.MaxAwaitSeconds, testNow))
	assert.Error(t, ValidateAwaitUntil(testNow+inboundmessage.MaxAwaitSeconds+1, testNow))
	assert.Error(t, ValidateAwaitUntil(testNow, testNow))
}

func TestSnooze_ParksAndSchedulesTheWakeUp(t *testing.T) {
	t.Parallel()

	id := newID("imsg_")
	actor := newID("usr_")
	svc, fakes := withQueue(&Service{}, waitingState(id))

	result, err := svc.Snooze(t.Context(), &SnoozeRequest{
		TenantInfo: queueTenant,
		ActorID:    actor,
		Selection:  Selection{IDs: []pulid.ID{id}},
		Until:      testNow + 7200,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Affected)

	state := fakes.queue.state(id)
	assert.Equal(t, inboundmessage.ParkingSnoozed, state.Parked)
	assert.Equal(t, testNow+7200, state.SnoozedUntil)
	assert.Equal(t, []pulid.ID{id}, fakes.wakes.scheduled)
	require.Len(t, fakes.activities.recorded, 1)
	activity := fakes.activities.recorded[0]
	assert.Equal(t, triage.ActivitySnoozed, activity.Kind)
	assert.Equal(t, triage.ActorPerson, activity.ActorKind)
	assert.Equal(t, actor, activity.ActorID)
	assert.Equal(t, testNow+7200, activity.Until)
}

func TestSnooze_RefusesASettledMessageAndAnEventWithNoShipment(t *testing.T) {
	t.Parallel()

	settled := waitingState(newID("imsg_"))
	settled.Status = inboundmessage.StatusActioned
	svc, fakes := withQueue(&Service{}, settled)

	_, err := svc.Snooze(t.Context(), &SnoozeRequest{
		TenantInfo: queueTenant,
		Selection:  Selection{IDs: []pulid.ID{settled.ID}},
		Until:      testNow + 60,
	})
	require.Error(t, err)
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)

	unlinked := waitingState(newID("imsg_"))
	fakes.queue.states[unlinked.ID] = unlinked
	_, err = svc.Snooze(t.Context(), &SnoozeRequest{
		TenantInfo: queueTenant,
		Selection:  Selection{IDs: []pulid.ID{unlinked.ID}},
		Event:      inboundmessage.SnoozeEventShipmentDelivered,
	})
	require.Error(t, err)
	assert.Empty(t, fakes.queue.updates, "nothing was written")
}

func TestSnooze_WatchesTheMatchedShipmentForAnEvent(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.MatchedShipmentID = newID("shp_")
	svc, fakes := withQueue(&Service{}, state)

	_, err := svc.Snooze(t.Context(), &SnoozeRequest{
		TenantInfo: queueTenant,
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
		Event:      inboundmessage.SnoozeEventShipmentDelivered,
	})
	require.NoError(t, err)

	parked := fakes.queue.state(state.ID)
	assert.Equal(t, state.MatchedShipmentID, parked.SnoozeWatchID)
	assert.Empty(t, fakes.wakes.scheduled, "an event snooze has no timer")
}

func TestWakeIfDue_IsIdempotent(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.Parked = inboundmessage.ParkingSnoozed
	state.SnoozedUntil = testNow + 600
	svc, fakes := withQueue(&Service{}, state)

	early, err := svc.WakeIfDue(t.Context(), queueTenant, state.ID, testNow)
	require.NoError(t, err)
	assert.False(t, early.Woken)
	assert.True(t, early.Schedule.Open)
	assert.Equal(t, testNow+600, early.Schedule.WakeAt)

	woken, err := svc.WakeIfDue(t.Context(), queueTenant, state.ID, testNow+600)
	require.NoError(t, err)
	assert.True(t, woken.Woken)
	assert.Empty(t, fakes.queue.state(state.ID).Parked)

	again, err := svc.WakeIfDue(t.Context(), queueTenant, state.ID, testNow+601)
	require.NoError(t, err)
	assert.False(t, again.Woken)
	assert.False(t, again.Schedule.Open, "a woken message has nothing left to wait for")
	assert.Equal(t, []triage.ActivityKind{triage.ActivityWoken}, fakes.activities.kinds())
}

func TestWakeIfDue_StopsWaitingOnAReplyThatNeverCame(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.Parked = inboundmessage.ParkingAwaitingReply
	state.AwaitingUntil = testNow
	svc, fakes := withQueue(&Service{}, state)

	result, err := svc.WakeIfDue(t.Context(), queueTenant, state.ID, testNow)
	require.NoError(t, err)
	assert.True(t, result.Woken)
	assert.Equal(t, []triage.ActivityKind{triage.ActivityStoppedWaiting}, fakes.activities.kinds())
	assert.Equal(t, triage.ActorSystem, fakes.activities.recorded[0].ActorKind)
}

func TestReconcileOverdueWakes_WakesWhatALostTimerMissed(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.Parked = inboundmessage.ParkingSnoozed
	state.SnoozedUntil = testNow - 7200
	svc, fakes := withQueue(&Service{}, state)
	fakes.queue.overdue = []inboundmessageports.ParkedRef{{
		ID: state.ID, OrganizationID: queueTenant.OrgID, BusinessUnitID: queueTenant.BuID,
	}}

	woken, err := svc.ReconcileOverdueWakes(t.Context(), testNow)
	require.NoError(t, err)
	assert.Equal(t, 1, woken)
	assert.Empty(t, fakes.queue.state(state.ID).Parked)
}

func TestResolveSelection_IsBounded(t *testing.T) {
	t.Parallel()

	svc, fakes := withQueue(&Service{})
	ids := make([]pulid.ID, inboundmessageports.MaxBulkSelection+1)
	for idx := range ids {
		ids[idx] = newID("imsg_")
	}

	_, err := svc.resolveSelection(t.Context(), queueTenant, Selection{IDs: ids})
	require.Error(t, err)

	_, err = svc.resolveSelection(t.Context(), queueTenant, Selection{})
	require.Error(t, err)

	fakes.queue.selected = ids[:3]
	got, err := svc.resolveSelection(t.Context(), queueTenant, Selection{
		Folder: &inboundmessageports.ListInboundMessagesRequest{Lane: inboundmessageports.LaneWaiting},
	})
	require.NoError(t, err)
	assert.Len(t, got, 3)
	require.NotNil(t, fakes.queue.selectReq)
	assert.Equal(t, inboundmessageports.MaxBulkSelection, fakes.queue.selectReq.Limit)
	assert.Equal(t, queueTenant, fakes.queue.selectReq.Folder.Filter.TenantInfo,
		"the folder is always read inside the caller's tenant")

	deduped, err := svc.resolveSelection(t.Context(), queueTenant, Selection{
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
	snoozed.SnoozedUntil = testNow + 600
	done := waitingState(newID("imsg_"))
	done.Status = inboundmessage.StatusActioned
	actor := newID("usr_")
	svc, fakes := withQueue(&Service{}, waiting, snoozed, done)

	result, err := svc.ReviewMany(t.Context(), &BulkReviewRequest{
		TenantInfo: queueTenant,
		ActorID:    actor,
		Selection:  Selection{IDs: []pulid.ID{waiting.ID, snoozed.ID, done.ID}},
		Status:     inboundmessage.StatusActioned,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.Affected, "a message already handled is not handled again")
	assert.Nil(t, result.RuleID)
	assert.Equal(t, 1, fakes.db.txs, "one transaction for the whole selection")
	require.Len(t, fakes.queue.updates, 1)
	assert.Len(t, fakes.queue.updates[0].IDs, 2, "one statement for the whole selection")
	assert.Equal(t, []pulid.ID{snoozed.ID}, fakes.wakes.cancelled)
	assert.Equal(t,
		[]triage.ActivityKind{triage.ActivityHandled, triage.ActivityHandled},
		fakes.activities.kinds(),
	)
	assert.Zero(t, fakes.queue.state(waiting.ID).DueAt, "a settled message is due nowhere")

	_, err = svc.ReviewMany(t.Context(), &BulkReviewRequest{
		TenantInfo: queueTenant,
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
	svc, fakes := withQueue(&Service{ruleTeacher: teacher}, state)

	result, err := svc.ReviewMany(t.Context(), &BulkReviewRequest{
		TenantInfo: queueTenant,
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
		Status:     inboundmessage.StatusIgnored,
		Teach:      inboundmessage.NovaActionIgnore,
	})
	require.NoError(t, err)
	require.NotNil(t, result.RuleID)
	require.Len(t, teacher.taught, 1)
	assert.Equal(t, inboundmessageports.TeachRequest{
		TenantInfo:     queueTenant,
		MessageID:      state.ID,
		Classification: inboundmessage.ClassificationTender,
		SenderDomain:   "northwind.example",
		MailboxID:      state.MailboxID,
		Action:         inboundmessage.NovaActionIgnore,
	}, teacher.taught[0])
	assert.Contains(t, fakes.activities.kinds(), triage.ActivityRuleCreated)
}

func TestReopen_PutsSettledMessagesBackWithAFreshTarget(t *testing.T) {
	t.Parallel()

	state := waitingState(newID("imsg_"))
	state.Status = inboundmessage.StatusIgnored
	state.DueAt = 0
	svc, fakes := withQueue(&Service{}, state)
	fakes.targets.rows = []*triage.ResponseTarget{{
		Classification: inboundmessage.ClassificationTender, Minutes: 30,
	}}

	result, err := svc.Reopen(t.Context(), &ReopenRequest{
		TenantInfo: queueTenant,
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Affected)

	reopened := fakes.queue.state(state.ID)
	assert.Equal(t, inboundmessage.StatusInReview, reopened.Status)
	assert.Equal(t, state.ReceivedAt+30*60, reopened.DueAt)
	assert.Equal(t, []triage.ActivityKind{triage.ActivityReopened}, fakes.activities.kinds())
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
	svc, fakes := withQueue(&Service{directory: directory}, state)

	_, err := svc.Assign(t.Context(), &AssignRequest{
		TenantInfo: queueTenant,
		ActorID:    me,
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
		OwnerID:    other,
	})
	var denied *errortypes.AuthorizationError
	require.ErrorAs(t, err, &denied)

	result, err := svc.Assign(t.Context(), &AssignRequest{
		TenantInfo: queueTenant,
		ActorID:    me,
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
		OwnerID:    me,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Affected)
	assert.Equal(t, me, fakes.queue.state(state.ID).OwnerID)
	require.Len(t, fakes.activities.recorded, 1)
	assert.Equal(t, "Ada Admin", fakes.activities.recorded[0].Subject)

	fakes.queue.states[state.ID].OwnerID = other
	_, err = svc.Assign(t.Context(), &AssignRequest{
		TenantInfo: queueTenant,
		ActorID:    me,
		Selection:  Selection{IDs: []pulid.ID{state.ID}},
	})
	require.ErrorAs(t, err, &denied, "taking someone else off a message is assigning it")

	_, err = svc.Assign(t.Context(), &AssignRequest{
		TenantInfo:      queueTenant,
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
	svc, fakes := withQueue(&Service{directory: directory}, states...)
	fakes.queue.handOffLeft = ids

	result, err := svc.HandOff(t.Context(), &HandOffRequest{
		TenantInfo: queueTenant,
		ActorID:    newID("usr_"),
		ToUserID:   onCall,
	})
	require.NoError(t, err)
	assert.Equal(t, len(ids)-1, result.Affected, "a parked message is not handed off")
	assert.Equal(t, onCall, result.ToUserID)
	assert.Len(t, fakes.queue.updates, 2, "handed off in bounded batches")
	for _, update := range fakes.queue.updates {
		assert.LessOrEqual(t, len(update.IDs), inboundmessageports.HandOffBatchSize)
		require.NotNil(t, update.Patch.HandOff)
		assert.Equal(t, onCall, update.Patch.HandOff.To)
	}
	assert.Len(t, fakes.activities.recorded, len(ids)-1)
	assert.Equal(t, triage.ActivityHandedOff, fakes.activities.recorded[0].Kind)
	assert.Equal(t, onCall, fakes.queue.state(states[1].ID).OwnerID)
	assert.True(t, fakes.queue.state(parked.ID).OwnerID.IsNil())

	_, err = svc.HandOff(t.Context(), &HandOffRequest{
		TenantInfo: queueTenant,
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
	repo := &settleRepo{message: &inboundmessage.InboundMessage{ID: state.ID}}
	svc, fakes := withQueue(&Service{messageRepo: repo}, state)

	_, err := svc.WaitForReply(t.Context(), &WaitForReplyRequest{
		TenantInfo: queueTenant,
		MessageID:  state.ID,
		Until:      testNow + 86400,
	})
	require.NoError(t, err)
	parked := fakes.queue.state(state.ID)
	assert.Equal(t, inboundmessage.ParkingAwaitingReply, parked.Parked)
	assert.Equal(t, testNow+86400, parked.AwaitingUntil)
	assert.Equal(t, []pulid.ID{state.ID}, fakes.wakes.scheduled)

	_, err = svc.StopWaiting(t.Context(), &MessageActionRequest{
		TenantInfo: queueTenant,
		MessageID:  state.ID,
	})
	require.NoError(t, err)
	assert.Empty(t, fakes.queue.state(state.ID).Parked)
	assert.Equal(t, []pulid.ID{state.ID}, fakes.wakes.cancelled)

	_, err = svc.Wake(t.Context(), &MessageActionRequest{TenantInfo: queueTenant, MessageID: state.ID})
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

func (r *recordingBriefing) InvalidateBriefing(_ context.Context, tenantInfo pagination.TenantInfo) {
	r.invalidated = append(r.invalidated, tenantInfo)
}

func TestNotifier_PublishesProjectsAndDropsTheBriefingOnce(t *testing.T) {
	t.Parallel()

	realtime := &recordingRealtime{}
	tower := &recordingTower{}
	briefing := &recordingBriefing{}
	notifier := newNotifier(notifierDeps{
		logger:     zap.NewNop(),
		realtime:   realtime,
		watchtower: tower,
		briefing:   briefing,
	})

	waiting := messageFromState(waitingState(newID("imsg_")))
	settled := messageFromState(waitingState(newID("imsg_")))
	settled.Status = inboundmessage.StatusActioned
	notifier.MessagesChanged(t.Context(), waiting, settled)

	assert.Len(t, realtime.published, 2)
	assert.Equal(t, []string{waiting.ID.String()}, tower.upserted)
	assert.Equal(t, []string{settled.ID.String()}, tower.resolved)
	assert.Len(t, briefing.invalidated, 1, "one briefing per tenant, however many messages")

	many := make([]*inboundmessage.InboundMessage, 0, maxRecordInvalidations+1)
	for range maxRecordInvalidations + 1 {
		many = append(many, messageFromState(waitingState(newID("imsg_"))))
	}
	realtime.published = nil
	notifier.MessagesChanged(t.Context(), many...)
	require.Len(t, realtime.published, 1, "a large change is one invalidation for the inbox")
	assert.True(t, realtime.published[0].RecordID.IsNil())
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

	_, fakes := withQueue(&Service{}, podWait, paidWait)
	realtime := &recordingRealtime{}
	sink := &EventSink{
		l:            zap.NewNop(),
		db:           fakes.db,
		queueRepo:    fakes.queue,
		activityRepo: fakes.activities,
		notifier:     newNotifier(notifierDeps{logger: zap.NewNop(), realtime: realtime}),
		now:          func() int64 { return testNow },
	}

	sink.DocumentAttached(t.Context(), queueTenant, shipmentID, newID("dt_"))
	assert.Equal(t, inboundmessage.ParkingSnoozed, fakes.queue.state(podWait.ID).Parked,
		"a different document leaves it snoozed")

	sink.DocumentAttached(t.Context(), queueTenant, shipmentID, docType)
	assert.Empty(t, fakes.queue.state(podWait.ID).Parked)
	assert.Equal(t, inboundmessage.ParkingSnoozed, fakes.queue.state(paidWait.ID).Parked)
	require.Len(t, fakes.activities.recorded, 1)
	assert.Equal(t, triage.ActivityWoken, fakes.activities.recorded[0].Kind)
	assert.Equal(t, shipmentID, fakes.activities.recorded[0].RecordID)
	assert.Len(t, realtime.published, 1)

	sink.InvoicePaid(t.Context(), queueTenant, shipmentID)
	assert.Empty(t, fakes.queue.state(paidWait.ID).Parked)

	sink.ShipmentDelivered(t.Context(), queueTenant, pulid.Nil)
	assert.Len(t, fakes.queue.wakeEvents, 3, "an event naming no shipment asks nothing")
}

func TestUpdateResponseTargets_ValidatesAndRecomputes(t *testing.T) {
	t.Parallel()

	svc, fakes := withQueue(&Service{})
	minutes := 90
	views, err := svc.UpdateResponseTargets(t.Context(), &UpdateTargetsRequest{
		TenantInfo: queueTenant,
		Changes: []TargetChange{
			{Classification: inboundmessage.ClassificationTender, Minutes: &minutes},
			{Classification: inboundmessage.ClassificationInvoice},
		},
	})
	require.NoError(t, err)
	require.Len(t, fakes.targets.saved, 1)
	assert.Len(t, fakes.targets.saved[0].Upsert, 1)
	assert.Equal(t,
		[]inboundmessage.Classification{inboundmessage.ClassificationInvoice},
		fakes.targets.saved[0].Reset,
	)
	assert.Equal(t, [][]inboundmessage.Classification{{
		inboundmessage.ClassificationTender, inboundmessage.ClassificationInvoice,
	}}, fakes.wakes.recompute)
	assert.Len(t, views, len(inboundmessage.AllClassifications()))

	tooShort := 1
	_, err = svc.UpdateResponseTargets(t.Context(), &UpdateTargetsRequest{
		TenantInfo: queueTenant,
		Changes: []TargetChange{
			{Classification: inboundmessage.ClassificationTender, Minutes: &tooShort},
		},
	})
	require.Error(t, err)

	_, err = svc.UpdateResponseTargets(t.Context(), &UpdateTargetsRequest{
		TenantInfo: queueTenant,
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

func TestApplyDue_StartsTheClockOnlyForWorkOwed(t *testing.T) {
	t.Parallel()

	targets := triage.ResolveTargets(nil)
	message := &inboundmessage.InboundMessage{
		ReceivedAt:     testNow,
		Status:         inboundmessage.StatusInReview,
		Classification: inboundmessage.ClassificationStatusRequest,
	}
	ApplyDue(message, targets)
	assert.Equal(t, testNow+60*60, message.DueAt)

	message.Status = inboundmessage.StatusActioned
	message.Parked = inboundmessage.ParkingSnoozed
	ApplyDue(message, targets)
	assert.Zero(t, message.DueAt)
	assert.Empty(t, message.Parked)
}
