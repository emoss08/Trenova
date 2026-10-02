package assistantservice

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// meetingConversations makes the history read and the room count each wait
// for the other to start, so a prepare that runs them one after the other
// finds the other never began.
type meetingConversations struct {
	*stubConversations

	listing  chan struct{}
	counting chan struct{}

	listSawCount atomic.Bool
	countSawList atomic.Bool
}

func (m *meetingConversations) ListMessages(
	ctx context.Context,
	req repositories.ListMessagesRequest,
) ([]conversation.Message, error) {
	close(m.listing)
	m.listSawCount.Store(waitFor(ctx, m.counting))

	return m.stubConversations.ListMessages(ctx, req)
}

func (m *meetingConversations) CountMessages(
	ctx context.Context,
	req repositories.CountMessagesRequest,
) (int, error) {
	close(m.counting)
	m.countSawList.Store(waitFor(ctx, m.listing))

	return m.stubConversations.CountMessages(ctx, req)
}

/*
Every check a question passes before it is answered used to run one after the
other: the files, the agent, the budget, the room left in the conversation,
the history. They read different things and none needs another's answer, so
the person waited for the sum of a dozen reads before the model was asked
anything.
*/
func TestPrepareTurn_ReadsTheHistoryWhileTheRoomIsCounted(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationService(&scriptedCompletion{}, testDefinition())
	meeting := &meetingConversations{
		stubConversations: conversations,
		listing:           make(chan struct{}),
		counting:          make(chan struct{}),
	}
	svc.conversations = meeting
	actor := testActor()

	plan, err := svc.PrepareTurn(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:   conversations.thread.ID,
		Content:    "Where is S1?",
		TenantInfo: actor.TenantInfo(),
	}, actor)

	require.NoError(t, err)
	require.NotNil(t, plan)
	assert.True(t, meeting.listSawCount.Load(), "the history waited for the count")
	assert.True(t, meeting.countSawList.Load(), "the count waited for the history")
}

// Checks that run together still answer in the order they always did: a
// question that fails two of them is told about the first.
func TestPrepareTurn_ReportsTheFirstFailingCheckInItsOldOrder(t *testing.T) {
	t.Parallel()

	definition := testDefinition()
	definition.Enabled = false
	svc, conversations := newConversationService(&scriptedCompletion{}, definition)
	conversations.count = maxThreadMessages
	actor := testActor()

	_, err := svc.PrepareTurn(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:   conversations.thread.ID,
		Content:    "Where is S1?",
		TenantInfo: actor.TenantInfo(),
	}, actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
	assert.NotContains(t, err.Error(), "limit")
}

// meetingDocuments holds each document read until every other expected read
// has started, and counts how many found the rest already under way.
type meetingDocuments struct {
	repositories.DocumentRepository

	docs    map[pulid.ID]*document.Document
	arrived sync.WaitGroup
	all     chan struct{}
	once    sync.Once
	met     atomic.Int32
}

func newMeetingDocuments(docs map[pulid.ID]*document.Document) *meetingDocuments {
	m := &meetingDocuments{docs: docs, all: make(chan struct{})}
	m.arrived.Add(len(docs))
	go func() {
		m.arrived.Wait()
		m.once.Do(func() { close(m.all) })
	}()

	return m
}

func (m *meetingDocuments) GetByID(
	ctx context.Context,
	req repositories.GetDocumentByIDRequest,
) (*document.Document, error) {
	m.arrived.Done()
	if waitFor(ctx, m.all) {
		m.met.Add(1)
	}
	if doc, ok := m.docs[req.ID]; ok {
		return doc, nil
	}

	return nil, errors.New("not found")
}

// The files attached to one message are read side by side, and come back in
// the order they were attached.
func TestPrepareTurn_ReadsAttachedFilesTogether(t *testing.T) {
	t.Parallel()

	svc, conversations := newConversationService(&scriptedCompletion{}, testDefinition())
	actor := testActor()
	first := ownDocument(conversations.thread, actor)
	first.OriginalName = "rate-con.pdf"
	second := ownDocument(conversations.thread, actor)
	second.OriginalName = "bol.pdf"
	documents := newMeetingDocuments(map[pulid.ID]*document.Document{
		first.ID:  first,
		second.ID: second,
	})
	svc.documents = documents

	plan, err := svc.PrepareTurn(t.Context(), &serviceports.SendMessageRequest{
		ThreadID:              conversations.thread.ID,
		Content:               "Compare these",
		TenantInfo:            actor.TenantInfo(),
		AttachmentDocumentIDs: []pulid.ID{first.ID, second.ID},
	}, actor)

	require.NoError(t, err)
	assert.EqualValues(t, 2, documents.met.Load(), "each file waited for the other")
	require.Len(t, plan.Attachments, 2)
	assert.Equal(t, "rate-con.pdf", plan.Attachments[0].FileName)
	assert.Equal(t, "bol.pdf", plan.Attachments[1].FileName)
}
