package assistantjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

// notices keeps notifications the way the real store does: one created is
// found again by its event type and correlation.
type notices struct {
	created []*notification.Notification
	asked   []repositories.ExistsRecentNotificationRequest
}

func (n *notices) ExistsRecent(
	_ context.Context,
	req repositories.ExistsRecentNotificationRequest,
) (bool, error) {
	n.asked = append(n.asked, req)
	for _, created := range n.created {
		if created.EventType == req.EventType &&
			created.CorrelationID != nil && *created.CorrelationID == req.CorrelationID {
			return true, nil
		}
	}

	return false, nil
}

func (n *notices) Create(
	_ context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	n.created = append(n.created, entity)

	return entity, nil
}

// conversations is the person's conversation, and what was done to it.
type conversations struct {
	thread  *conversation.Thread
	err     error
	read    repositories.GetThreadRequest
	updated []*serviceports.UpdateThreadRequest
	actor   *serviceports.RequestActor
}

func (c *conversations) GetThread(
	_ context.Context,
	req repositories.GetThreadRequest,
) (*conversation.Thread, error) {
	c.read = req
	if c.err != nil {
		return nil, c.err
	}

	return c.thread, nil
}

func (c *conversations) UpdateThread(
	_ context.Context,
	req *serviceports.UpdateThreadRequest,
	actor *serviceports.RequestActor,
) (*conversation.Thread, error) {
	c.updated = append(c.updated, req)
	c.actor = actor
	if req.Keep && c.thread.Origin.Keepable() {
		c.thread.Origin = conversation.ThreadOriginDesk
	}

	return c.thread, nil
}

func notifyInput(status conversation.AssistantTurnStatus) *NotifyUnseenTurnInput {
	userID := pulid.MustNew("usr_")

	return &NotifyUnseenTurnInput{
		Payload: &AssistantTurnPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
				UserID:         userID,
				Timestamp:      1_700_000_000,
			},
			TurnID:   pulid.MustNew("atrn_"),
			ThreadID: pulid.MustNew("athr_"),
			Actor: serviceports.RequestActor{
				PrincipalType: serviceports.PrincipalTypeUser,
				PrincipalID:   userID,
				UserID:        userID,
			},
		},
		Status: status,
	}
}

func threadFor(in *NotifyUnseenTurnInput, origin conversation.ThreadOrigin) *conversation.Thread {
	return &conversation.Thread{
		ID:             in.Payload.ThreadID,
		OrganizationID: in.Payload.OrganizationID,
		BusinessUnitID: in.Payload.BusinessUnitID,
		UserID:         in.Payload.Actor.UserID,
		Title:          "Late loads this week",
		Origin:         origin,
	}
}

func notifyActivities(threads *conversations, sent *notices) *Activities {
	a := NewActivities(ActivitiesParams{Logger: zap.NewNop()})
	a.threads = threads
	a.notifications = sent

	return a
}

func notify(t *testing.T, a *Activities, in *NotifyUnseenTurnInput) {
	t.Helper()

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(a)

	_, err := env.ExecuteActivity(a.NotifyUnseenTurnActivity, in)
	require.NoError(t, err)
}

func TestNotifyUnseenTurn_TellsThePersonWhoAskedWhereTheirReplyIs(t *testing.T) {
	t.Parallel()

	in := notifyInput(conversation.AssistantTurnStatusCompleted)
	threads := &conversations{thread: threadFor(in, conversation.ThreadOriginDesk)}
	sent := &notices{}

	notify(t, notifyActivities(threads, sent), in)

	require.Len(t, sent.created, 1)
	created := sent.created[0]
	assert.Equal(t, in.Payload.OrganizationID, created.OrganizationID)
	require.NotNil(t, created.BusinessUnitID)
	assert.Equal(t, in.Payload.BusinessUnitID, *created.BusinessUnitID)
	require.NotNil(t, created.TargetUserID)
	assert.Equal(t, in.Payload.Actor.UserID, *created.TargetUserID)
	assert.Equal(t, notification.ChannelUser, created.Channel)
	assert.Equal(t, ReplyReadyKind, created.EventType)
	assert.Equal(t, "Your reply is ready", created.Title)
	assert.NotContains(t, created.Title+created.Message, "Late loads this week",
		"a notification reaches every socket in the organization, so it never names the conversation")

	assert.Equal(t, map[string]any{
		"kind":     "assistant_reply_ready",
		"threadId": in.Payload.ThreadID.String(),
		"turnId":   in.Payload.TurnID.String(),
		"status":   "Completed",
		"link":     threadLink(in.Payload.ThreadID),
	}, created.Data, "the client reads exactly these keys")

	assert.Equal(t, in.Payload.Actor.UserID, threads.read.UserID,
		"the conversation is read as the person's own")
	assert.Empty(t, threads.updated, "a listed conversation is left as it is")
}

/*
A quick question's conversation is hidden until it is kept, and the sweep of
unkept ones deletes it. A notification leading there would lead nowhere, so
the conversation is kept first, as the person who asked.
*/
func TestNotifyUnseenTurn_KeepsAQuickQuestionItLeadsTo(t *testing.T) {
	t.Parallel()

	in := notifyInput(conversation.AssistantTurnStatusCompleted)
	threads := &conversations{thread: threadFor(in, conversation.ThreadOriginAsk)}
	sent := &notices{}

	notify(t, notifyActivities(threads, sent), in)

	require.Len(t, threads.updated, 1)
	assert.True(t, threads.updated[0].Keep)
	assert.Equal(t, in.Payload.ThreadID, threads.updated[0].ThreadID)
	require.NotNil(t, threads.actor)
	assert.Equal(t, in.Payload.Actor.UserID, threads.actor.UserID)
	assert.True(t, threads.thread.Origin.Listed())
	require.Len(t, sent.created, 1)
}

func TestNotifyUnseenTurn_LeavesAPageConversationToItsPage(t *testing.T) {
	t.Parallel()

	for _, origin := range []conversation.ThreadOrigin{
		conversation.ThreadOriginImport,
		conversation.ThreadOriginFormula,
	} {
		in := notifyInput(conversation.AssistantTurnStatusCompleted)
		threads := &conversations{thread: threadFor(in, origin)}
		sent := &notices{}

		notify(t, notifyActivities(threads, sent), in)

		assert.Empty(t, sent.created, string(origin))
		assert.Empty(t, threads.updated, string(origin))
		assert.Equal(t, origin, threads.thread.Origin)
	}
}

// A retry of an attempt that already told the person does not tell them
// again.
func TestNotifyUnseenTurn_NeverTellsThePersonTwice(t *testing.T) {
	t.Parallel()

	in := notifyInput(conversation.AssistantTurnStatusCompleted)
	threads := &conversations{thread: threadFor(in, conversation.ThreadOriginAsk)}
	sent := &notices{}
	a := notifyActivities(threads, sent)

	notify(t, a, in)
	notify(t, a, in)

	assert.Len(t, sent.created, 1)
	assert.Len(t, threads.updated, 1)
	require.NotEmpty(t, sent.asked)
	assert.Equal(t, in.Payload.TurnID.String(), sent.asked[0].CorrelationID)
	assert.Equal(t, in.Payload.Timestamp, sent.asked[0].Since)
}

// A conversation deleted while its reply was being written leaves nothing to
// lead anybody to.
func TestNotifyUnseenTurn_SaysNothingAboutAConversationThatIsGone(t *testing.T) {
	t.Parallel()

	in := notifyInput(conversation.AssistantTurnStatusCompleted)
	threads := &conversations{err: errortypes.NewNotFoundError("Thread not found")}
	sent := &notices{}

	notify(t, notifyActivities(threads, sent), in)

	assert.Empty(t, sent.created)
}

func TestNotifyUnseenTurn_RetriesAConversationItCouldNotRead(t *testing.T) {
	t.Parallel()

	in := notifyInput(conversation.AssistantTurnStatusCompleted)
	threads := &conversations{err: errors.New("database is down")}
	sent := &notices{}
	a := notifyActivities(threads, sent)

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(a)
	_, err := env.ExecuteActivity(a.NotifyUnseenTurnActivity, in)

	require.Error(t, err)
	assert.Empty(t, sent.created)
}

func TestReplyReadyWording(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   conversation.AssistantTurnStatus
		title    string
		message  string
		priority notification.Priority
	}{
		{
			name:     "answered",
			status:   conversation.AssistantTurnStatusCompleted,
			title:    "Your reply is ready",
			message:  "Open the conversation to read it.",
			priority: notification.PriorityMedium,
		},
		{
			name:     "refused",
			status:   conversation.AssistantTurnStatusRefused,
			title:    "Your reply is ready",
			message:  "Open the conversation to read it.",
			priority: notification.PriorityMedium,
		},
		{
			name:     "failed",
			status:   conversation.AssistantTurnStatusFailed,
			title:    "Your reply could not finish",
			message:  "The assistant stopped before it finished. Open the conversation to see what was kept and ask again.",
			priority: notification.PriorityHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			title, message, priority := replyReadyWording(tt.status)

			assert.Equal(t, tt.title, title)
			assert.Equal(t, tt.message, message)
			assert.Equal(t, tt.priority, priority)
		})
	}
}

func TestNotifiesUnseen(t *testing.T) {
	t.Parallel()

	assert.True(t, notifiesUnseen(conversation.AssistantTurnStatusCompleted))
	assert.True(t, notifiesUnseen(conversation.AssistantTurnStatusRefused))
	assert.True(t, notifiesUnseen(conversation.AssistantTurnStatusFailed))
	assert.False(t, notifiesUnseen(conversation.AssistantTurnStatusStopped))
	assert.False(t, notifiesUnseen(conversation.AssistantTurnStatusRunning))
	assert.False(t, notifiesUnseen(conversation.AssistantTurnStatusPending))
}

// The link opens the conversation where the record-link registry says
// conversations open, and the Desk until the registry names one.
func TestThreadLink_ComesFromTheRecordLinkRegistry(t *testing.T) {
	t.Parallel()

	threadID := pulid.MustNew("athr_")
	want := deskPath
	if path, ok := productguide.RecordPath(threadRecordEntity, threadID.String()); ok {
		want = path
		assert.Contains(t, path, threadID.String())
	}

	assert.Equal(t, want, threadLink(threadID))
}

// cancels is the engine running turns, as the canceller sees it.
type cancels struct {
	serviceports.WorkflowStarter

	err       error
	cancelled string
}

func (c *cancels) CancelWorkflow(_ context.Context, workflowID, _ string) error {
	c.cancelled = workflowID

	return c.err
}

/*
The stop pressed in a conversation and the stop made by signing out go through
this one mapping, so both tell a turn that was stopped from one whose
execution is gone and that has to be closed by hand.
*/
func TestTurnCanceller_SaysWhenNoExecutionCarriesTheTurn(t *testing.T) {
	t.Parallel()

	workflows := &cancels{err: serviceerror.NewNotFound("workflow not found")}
	canceller := NewTurnCanceller(workflows)

	err := canceller.CancelTurn(t.Context(), "assistant-turn:atrn_1")

	require.ErrorIs(t, err, serviceports.ErrNoTurnExecution)
	assert.Equal(t, "assistant-turn:atrn_1", workflows.cancelled)
}

func TestTurnCanceller_PassesOnAnyOtherFailure(t *testing.T) {
	t.Parallel()

	down := errors.New("temporal is down")
	canceller := NewTurnCanceller(&cancels{err: down})

	err := canceller.CancelTurn(t.Context(), "assistant-turn:atrn_1")

	require.ErrorIs(t, err, down)
	assert.NotErrorIs(t, err, serviceports.ErrNoTurnExecution)
}

func TestTurnCanceller_CancelsTheExecution(t *testing.T) {
	t.Parallel()

	workflows := &cancels{}

	require.NoError(t, NewTurnCanceller(workflows).CancelTurn(t.Context(), "assistant-turn:atrn_1"))
	assert.Equal(t, "assistant-turn:atrn_1", workflows.cancelled)
}
