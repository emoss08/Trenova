package assistantthreads

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/turnlimits"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationschedule"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationscheduleports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newConversationThreads(
	definition *agentdefinition.Definition,
) (*Service, *assistanttest.StubConversations) {
	conversations := &assistanttest.StubConversations{Thread: &conversation.Thread{
		ID:                pulid.MustNew("athr_"),
		AgentDefinitionID: pulid.MustNew("agdef_"),
		Title:             "Dispatch",
	}}

	return &Service{
		logger:        zap.NewNop(),
		conversations: conversations,
		definitions:   &assistanttest.StubDefinitions{Definition: definition},
		permissions:   &agenttest.StubPermissions{},
		runtime: agentruntime.New(agentruntime.Params{
			Logger:      zap.NewNop(),
			Completion:  &assistanttest.ScriptedCompletion{},
			QueryTools:  &assistanttest.StubQueryRegistry{},
			ActionTools: &assistanttest.StubActionRegistry{},
			Permissions: &agenttest.StubPermissions{},
		}),
	}, conversations
}

func numberedMessages(from, count int) []conversation.Message {
	messages := make([]conversation.Message, 0, count)
	for i := range count {
		messages = append(messages, conversation.Message{
			ID:       pulid.MustNew("amsg_"),
			Sequence: from + i,
			Role:     conversation.RoleUser,
			Content:  "m",
		})
	}

	return messages
}

// A page is the newest N of what lies below the cursor, and one row past it
// says whether there is a page above without counting what is left.
func TestListMessages_PagesFromTheEndAndSaysWhetherMoreExist(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationThreads(assistanttest.Definition())
	conversations.Messages = numberedMessages(10, 3)
	conversations.Count = 13
	before := 13

	page, err := svc.ListMessages(t.Context(), &assistantports.ListThreadMessagesRequest{
		Thread:         conversationports.GetThreadRequest{ID: conversations.Thread.ID},
		Limit:          2,
		BeforeSequence: &before,
	})
	require.NoError(t, err)

	assert.Equal(t, 3, conversations.LastList.Limit, "one more than the page")
	require.NotNil(t, conversations.LastList.BeforeSequence)
	assert.Equal(t, 13, *conversations.LastList.BeforeSequence)
	assert.True(t, page.HasMore)
	require.Len(t, page.Results, 2)
	assert.Equal(t, 11, page.Results[0].Sequence, "the extra row is the oldest, and is dropped")
	assert.Equal(t, 12, page.Results[1].Sequence)
	assert.Equal(t, 13, page.Total)
	assert.Equal(t, turnlimits.MaxThreadMessages, page.Limit)
}

func TestListMessages_ReportsTheLastPageAsSuch(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationThreads(assistanttest.Definition())
	conversations.Messages = numberedMessages(0, 2)
	conversations.Count = 2

	page, err := svc.ListMessages(t.Context(), &assistantports.ListThreadMessagesRequest{
		Thread: conversationports.GetThreadRequest{ID: conversations.Thread.ID},
		Limit:  2,
	})
	require.NoError(t, err)

	assert.False(t, page.HasMore)
	assert.Len(t, page.Results, 2)
	assert.Nil(t, conversations.LastList.BeforeSequence)
}

func TestListMessages_ClampsThePageSize(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationThreads(assistanttest.Definition())

	_, err := svc.ListMessages(t.Context(), &assistantports.ListThreadMessagesRequest{
		Thread: conversationports.GetThreadRequest{ID: conversations.Thread.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, defaultPageLimit+1, conversations.LastList.Limit)

	_, err = svc.ListMessages(t.Context(), &assistantports.ListThreadMessagesRequest{
		Thread: conversationports.GetThreadRequest{ID: conversations.Thread.ID},
		Limit:  100000,
	})
	require.NoError(t, err)
	assert.Equal(t, maxPageLimit+1, conversations.LastList.Limit)
}

type deletingConversations struct {
	conversationports.ConversationRepository

	err error
}

func (d *deletingConversations) DeleteThread(
	context.Context,
	*conversationports.GetThreadRequest,
) error {
	return d.err
}

// A deleted conversation's undecided proposals stayed in the decisions inbox,
// and approving one followed up into a thread that no longer existed. They are
// withdrawn with it, and only once the delete has gone through.
func TestDeleteThread_WithdrawsWhatItLeftWaiting(t *testing.T) {
	t.Parallel()

	threadID := pulid.MustNew("athr_")
	proposals := &assistanttest.StubProposalRepo{}
	svc := &Service{
		logger:        zap.NewNop(),
		conversations: &deletingConversations{},
		proposals:     proposals,
	}

	require.NoError(
		t,
		svc.DeleteThread(t.Context(), &conversationports.GetThreadRequest{ID: threadID}),
	)
	assert.Equal(t, []pulid.ID{threadID}, proposals.ExpiredThreads)

	refused := &assistanttest.StubProposalRepo{}
	svc = &Service{
		logger:        zap.NewNop(),
		conversations: &deletingConversations{err: errors.New("not yours")},
		proposals:     refused,
	}
	require.Error(
		t,
		svc.DeleteThread(t.Context(), &conversationports.GetThreadRequest{ID: threadID}),
	)
	assert.Empty(t, refused.ExpiredThreads, "a delete that did not happen withdraws nothing")
}

// heldSchedules stands in for the schedule service: the schedules a
// conversation keeps, whether they were read before it was deleted, and what
// was forgotten.
type heldSchedules struct {
	held       []*conversationschedule.Schedule
	readErr    error
	asked      []conversationscheduleports.ThreadSchedulesRequest
	readBefore bool
	forgot     []*conversationschedule.Schedule
	deleted    *bool
}

func (h *heldSchedules) OfThread(
	_ context.Context,
	req conversationscheduleports.ThreadSchedulesRequest,
) ([]*conversationschedule.Schedule, error) {
	h.asked = append(h.asked, req)
	h.readBefore = !*h.deleted

	return h.held, h.readErr
}

func (h *heldSchedules) Forget(_ context.Context, schedules []*conversationschedule.Schedule) {
	h.forgot = append(h.forgot, schedules...)
}

type recordingDelete struct {
	conversationports.ConversationRepository

	err     error
	deleted bool
}

func (d *recordingDelete) DeleteThread(context.Context, *conversationports.GetThreadRequest) error {
	if d.err != nil {
		return d.err
	}
	d.deleted = true

	return nil
}

/*
The requests scheduled in a conversation are deleted with it, by cascade, so
they are read while there is still something to read, and once the
conversation is gone their Temporal schedules are removed and their owner's
tabs told. A delete that did not happen forgets nothing, and one whose
schedules could not be read still goes through: the reconcile removes what
is left.
*/
func TestDeleteThread_ForgetsTheSchedulesItKept(t *testing.T) {
	t.Parallel()

	req := &conversationports.GetThreadRequest{
		ID:     pulid.MustNew("athr_"),
		UserID: pulid.MustNew("usr_"),
	}
	held := []*conversationschedule.Schedule{
		{ID: pulid.MustNew("csch_")},
		{ID: pulid.MustNew("csch_")},
	}

	conversations := &recordingDelete{}
	schedules := &heldSchedules{held: held, deleted: &conversations.deleted}
	svc := &Service{logger: zap.NewNop(), conversations: conversations, schedules: schedules}
	require.NoError(t, svc.DeleteThread(t.Context(), req))
	assert.True(t, schedules.readBefore, "the schedules are read before the cascade takes them")
	require.Len(t, schedules.asked, 1)
	assert.Equal(t, req.ID, schedules.asked[0].ThreadID)
	assert.Equal(t, req.UserID, schedules.asked[0].UserID)
	assert.Equal(t, held, schedules.forgot)

	refused := &recordingDelete{err: errors.New("not yours")}
	kept := &heldSchedules{held: held, deleted: &refused.deleted}
	svc = &Service{logger: zap.NewNop(), conversations: refused, schedules: kept}
	require.Error(t, svc.DeleteThread(t.Context(), req))
	assert.Empty(t, kept.forgot, "a delete that did not happen forgets no schedule")

	unread := &recordingDelete{}
	failing := &heldSchedules{readErr: errors.New("unavailable"), deleted: &unread.deleted}
	svc = &Service{logger: zap.NewNop(), conversations: unread, schedules: failing}
	require.NoError(t, svc.DeleteThread(t.Context(), req))
	assert.True(t, unread.deleted, "a failed read does not stop the delete")
	assert.Empty(t, failing.forgot)

	plain := &recordingDelete{}
	svc = &Service{logger: zap.NewNop(), conversations: plain}
	require.NoError(
		t,
		svc.DeleteThread(t.Context(), req),
		"without the schedule service it deletes",
	)
	assert.True(t, plain.deleted)
}
