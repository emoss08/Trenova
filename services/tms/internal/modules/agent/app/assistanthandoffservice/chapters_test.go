package assistanthandoffservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationchapter"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeChapterSpans struct {
	assistantports.AssistantChapterService

	span *assistantports.ChapterSpan
	err  error
	req  *assistantports.ChapterSpanRequest
}

func (f *fakeChapterSpans) Span(
	_ context.Context,
	req assistantports.ChapterSpanRequest,
) (*assistantports.ChapterSpan, error) {
	f.req = &req
	if f.err != nil {
		return nil, f.err
	}

	return f.span, nil
}

func through(sequence int) *int { return &sequence }

func spanOf(title string, from int, last *int) *assistantports.ChapterSpan {
	return &assistantports.ChapterSpan{
		Chapter: &conversationchapter.Chapter{ID: pulid.MustNew("achp_"), Title: title},
		From:    from,
		Through: last,
	}
}

func TestHistoryRequest_ReadsTheNewestOfTheWholeConversationWithoutAChapter(t *testing.T) {
	t.Parallel()

	origin := &conversation.Thread{ID: pulid.MustNew("athr_")}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	req := historyRequest(origin, tenant, nil)

	assert.Equal(t, assistantports.ListMessagesRequest{
		ThreadID:     origin.ID,
		TenantInfo:   tenant,
		Limit:        120,
		ExcludeKinds: conversation.ModelHiddenKinds(),
	}, req)
}

func TestHistoryRequest_ReadsAChapterFromItsFirstMessage(t *testing.T) {
	t.Parallel()

	origin := &conversation.Thread{ID: pulid.MustNew("athr_")}

	req := historyRequest(origin, pagination.TenantInfo{}, spanOf("Acme", 5, nil))

	require.NotNil(t, req.AfterSequence)
	assert.Equal(t, 4, *req.AfterSequence, "the chapter's first message is read")
	assert.Nil(t, req.BeforeSequence, "a chapter that runs to the end has no upper bound")
	assert.Equal(t, historyMessages, req.Limit)
	assert.Equal(t, origin.ID, req.ThreadID)
	assert.Equal(t, conversation.ModelHiddenKinds(), req.ExcludeKinds)
}

func TestHistoryRequest_StopsAtAChaptersLastMessage(t *testing.T) {
	t.Parallel()

	origin := &conversation.Thread{ID: pulid.MustNew("athr_")}

	req := historyRequest(origin, pagination.TenantInfo{}, spanOf("Acme", 5, through(9)))

	require.NotNil(t, req.AfterSequence)
	require.NotNil(t, req.BeforeSequence)
	assert.Equal(t, 4, *req.AfterSequence)
	assert.Equal(t, 10, *req.BeforeSequence, "the chapter's last message is read")
}

func TestHandoffTitle_NamesTheNewConversationForTheChapter(t *testing.T) {
	t.Parallel()

	origin := &conversation.Thread{Title: "Monday work"}

	assert.Equal(t, "Monday work", handoffTitle(origin, nil))
	assert.Equal(
		t,
		"Acme invoice dispute",
		handoffTitle(origin, spanOf("Acme invoice dispute", 5, nil)),
	)
}

func (f *handoffFixture) handoffChapter(
	t *testing.T,
	chapterID, agentID pulid.ID,
) (*assistantports.HandoffThreadResult, error) {
	t.Helper()

	return f.svc.Handoff(t.Context(), &assistantports.HandoffThreadRequest{
		ThreadID:          f.origin.ID,
		AgentDefinitionID: agentID,
		ChapterID:         chapterID,
	}, &permissionports.RequestActor{PrincipalType: permissionports.PrincipalTypeUser, UserID: f.user})
}

/*
A chapter can be taken on its own to a fresh conversation, even with the same
agent: the new conversation is named for the chapter and briefed from its
messages alone.
*/
func TestHandoff_AChapterGoesOnItsOwnEvenToTheSameAgent(t *testing.T) {
	t.Parallel()

	f := newHandoffFixture()
	span := spanOf("Acme invoice dispute", 6, through(13))
	chapters := &fakeChapterSpans{span: span}
	f.svc.chapters = chapters

	result, err := f.handoffChapter(t, span.Chapter.ID, f.origin.AgentDefinitionID)
	require.NoError(t, err)
	require.NotNil(t, result.Thread)

	require.NotNil(t, chapters.req)
	assert.Equal(t, span.Chapter.ID, chapters.req.ID)
	assert.Equal(t, f.origin.ID, chapters.req.ThreadID)
	assert.Equal(t, f.user, chapters.req.UserID, "the chapter is read as the person's own")

	assert.Equal(t, "Acme invoice dispute", f.assistant.started.Title)
	assert.Equal(t, f.origin.AgentDefinitionID, f.assistant.started.AgentDefinitionID)

	require.NotEmpty(t, f.conversations.listed)
	history := f.conversations.listed[0]
	require.NotNil(t, history.AfterSequence)
	require.NotNil(t, history.BeforeSequence)
	assert.Equal(t, 5, *history.AfterSequence)
	assert.Equal(t, 14, *history.BeforeSequence)
}

func TestHandoff_AChapterThatCannotBeReadStopsTheHandoff(t *testing.T) {
	t.Parallel()

	f := newHandoffFixture()
	f.svc.chapters = &fakeChapterSpans{
		err: errortypes.NewBusinessError("This chapter has nothing saved in it yet."),
	}

	_, err := f.handoffChapter(t, pulid.MustNew("achp_"), f.target.ID)

	assert.True(t, errortypes.IsBusinessError(err))
	assert.Nil(t, f.assistant.started)
	assert.Empty(t, f.conversations.appended)
	assert.Nil(t, f.completer.req)
}

func TestHandoff_AChapterNeedsChaptersToBeAvailable(t *testing.T) {
	t.Parallel()

	f := newHandoffFixture()

	_, err := f.handoffChapter(t, pulid.MustNew("achp_"), f.target.ID)

	assert.True(t, errortypes.IsBusinessError(err))
	assert.Nil(t, f.assistant.started)
}

func TestHandoff_TheWholeConversationStillCannotGoToItsOwnAgent(t *testing.T) {
	t.Parallel()

	f := newHandoffFixture()
	f.svc.chapters = &fakeChapterSpans{span: spanOf("Acme", 0, nil)}

	_, err := f.handoffChapter(t, "", f.origin.AgentDefinitionID)

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "agentDefinitionId", validation.Field)
	assert.Nil(t, f.assistant.started)
}
