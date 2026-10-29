//go:build integration

package assistantchapterrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/assistantchapterrepository"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/conversationrepository"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationchapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	ctx           context.Context
	other         context.Context
	otherTenant   pagination.TenantInfo
	db            *keel.DB
	repo          assistantports.ChapterRepository
	conversations assistantports.ConversationRepository
	tenant        pagination.TenantInfo
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "CH")
	conn := postgres.NewTestConnection(db)

	return &harness{
		ctx:         data.Ctx(ctx),
		other:       other.Ctx(ctx),
		otherTenant: other.Info(),
		db:          db,
		repo: assistantchapterrepository.New(assistantchapterrepository.Params{
			DB:     conn,
			Logger: zap.NewNop(),
		}),
		conversations: conversationrepository.New(conversationrepository.Params{
			DB:     conn,
			Logger: zap.NewNop(),
		}),
		tenant: data.Info(),
	}
}

func (h *harness) thread(t *testing.T) *conversation.Thread {
	t.Helper()

	created, err := h.conversations.CreateThread(h.ctx, &conversation.Thread{
		OrganizationID:    h.tenant.OrgID,
		BusinessUnitID:    h.tenant.BuID,
		UserID:            h.tenant.UserID,
		AgentDefinitionID: pulid.MustNew("agd_"),
		Status:            conversation.ThreadStatusActive,
		Origin:            conversation.ThreadOriginDesk,
	})
	require.NoError(t, err)

	return created
}

func (h *harness) turn(
	t *testing.T,
	thread *conversation.Thread,
	createdAt int64,
) *conversationturn.AssistantTurn {
	t.Helper()

	turn := &conversationturn.AssistantTurn{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		ThreadID:       thread.ID,
		UserID:         h.tenant.UserID,
		Origin:         conversationturn.AssistantTurnOriginPerson,
		Status:         conversation.AssistantTurnStatusCompleted,
		Input:          "question",
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	require.NoError(t, conversationdb.AssistantTurns.Create(h.ctx, h.db, turn))

	return turn
}

func (h *harness) say(
	t *testing.T,
	thread *conversation.Thread,
	turnID pulid.ID,
	contents ...string,
) []conversation.Message {
	t.Helper()

	messages := make([]conversation.Message, 0, len(contents))
	for idx, content := range contents {
		role := conversation.RoleUser
		if idx%2 == 1 {
			role = conversation.RoleAssistant
		}
		messages = append(messages, conversation.Message{Role: role, Content: content})
	}
	saved, err := h.conversations.AppendTurn(h.ctx, assistantports.AppendTurnRequest{
		ThreadID:   thread.ID,
		TurnID:     turnID,
		TenantInfo: h.tenant,
		Messages:   messages,
	})
	require.NoError(t, err)

	return saved
}

func (h *harness) chapter(
	t *testing.T,
	turn *conversationturn.AssistantTurn,
	title string,
	source conversationchapter.ChapterTitleSource,
) *conversationchapter.Chapter {
	t.Helper()

	created, err := h.repo.Create(h.ctx, &conversationchapter.Chapter{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		ThreadID:       turn.ThreadID,
		TurnID:         turn.ID,
		Title:          title,
		TitleSource:    source,
	})
	require.NoError(t, err)

	return created
}

func (h *harness) list(
	t *testing.T,
	ctx context.Context,
	threadID pulid.ID,
) []*conversationchapter.Chapter {
	t.Helper()

	chapters, err := h.repo.List(ctx, assistantports.ListChaptersRequest{
		ThreadID:   threadID,
		TenantInfo: h.tenant,
	})
	require.NoError(t, err)

	return chapters
}

func chapterIDs(chapters []*conversationchapter.Chapter) []pulid.ID {
	ids := make([]pulid.ID, 0, len(chapters))
	for _, chapter := range chapters {
		ids = append(ids, chapter.ID)
	}

	return ids
}

/*
A turn opens at most one chapter. The chapter workflow and a person can both
try to open one for the same question, and a retried activity tries again;
whichever lands first is the chapter, and the others read it back.
*/
func TestCreate_OpensOneChapterPerTurn(t *testing.T) {
	h := newHarness(t)
	thread := h.thread(t)
	turn := h.turn(t, thread, 1_000)
	h.say(t, thread, turn.ID, "Which loads are late?", "Three are late.")

	first := h.chapter(t, turn, "Late loads", conversationchapter.ChapterTitleModel)
	again := h.chapter(t, turn, "Something else", conversationchapter.ChapterTitleQuestion)

	assert.Equal(t, first.ID, again.ID)
	assert.Equal(t, "Late loads", again.Title)
	assert.Equal(t, conversationchapter.ChapterTitleModel, again.TitleSource)

	count, err := h.repo.Count(h.ctx, assistantports.ListChaptersRequest{
		ThreadID:   thread.ID,
		TenantInfo: h.tenant,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestList_OrdersChaptersByTheirTurnsAndSaysWhereEachStarts(t *testing.T) {
	h := newHarness(t)
	thread := h.thread(t)
	firstTurn := h.turn(t, thread, 1_000)
	secondTurn := h.turn(t, thread, 2_000)
	unanswered := h.turn(t, thread, 3_000)
	h.say(t, thread, firstTurn.ID, "Which loads are late?", "Three are late.")
	h.say(t, thread, "", "A card written outside any turn")
	h.say(t, thread, secondTurn.ID, "Open the Acme dispute", "It is open.")

	third := h.chapter(t, unanswered, "Detention policy", conversationchapter.ChapterTitleQuestion)
	second := h.chapter(
		t,
		secondTurn,
		"Acme invoice dispute",
		conversationchapter.ChapterTitleModel,
	)
	first := h.chapter(t, firstTurn, "Late loads", conversationchapter.ChapterTitlePerson)

	chapters := h.list(t, h.ctx, thread.ID)

	require.Equal(t, []pulid.ID{first.ID, second.ID, third.ID}, chapterIDs(chapters),
		"chapters read in the order their questions were asked, not the order they were saved")
	require.NotNil(t, chapters[0].StartSequence)
	assert.Equal(t, int64(0), *chapters[0].StartSequence)
	require.NotNil(t, chapters[1].StartSequence)
	assert.Equal(t, int64(3), *chapters[1].StartSequence,
		"a chapter starts at its turn's first message, after the card between them")
	assert.Nil(t, chapters[2].StartSequence, "a turn that saved nothing yet has no start")
	assert.Equal(t, []int64{1_000, 2_000, 3_000},
		[]int64{chapters[0].StartedAt, chapters[1].StartedAt, chapters[2].StartedAt})
	assert.Equal(t, "Late loads", chapters[0].Title)
	assert.Equal(t, conversationchapter.ChapterTitlePerson, chapters[0].TitleSource)

	got, err := h.repo.Get(h.ctx, assistantports.GetChapterRequest{
		ID:         second.ID,
		ThreadID:   thread.ID,
		TenantInfo: h.tenant,
	})
	require.NoError(t, err)
	require.NotNil(t, got.StartSequence)
	assert.Equal(t, int64(3), *got.StartSequence)

	forTurn, err := h.repo.ForTurn(h.ctx, assistantports.ChapterForTurnRequest{
		ThreadID:   thread.ID,
		TurnID:     secondTurn.ID,
		TenantInfo: h.tenant,
	})
	require.NoError(t, err)
	require.NotNil(t, forTurn)
	assert.Equal(t, second.ID, forTurn.ID)

	none, err := h.repo.ForTurn(h.ctx, assistantports.ChapterForTurnRequest{
		ThreadID:   thread.ID,
		TurnID:     h.turn(t, thread, 4_000).ID,
		TenantInfo: h.tenant,
	})
	require.NoError(t, err)
	assert.Nil(t, none, "a turn that opened no chapter reads as none, not an error")
}

func TestList_ReadsOnlyTheConversationsOwnChapters(t *testing.T) {
	h := newHarness(t)
	thread := h.thread(t)
	elsewhere := h.thread(t)
	h.chapter(t, h.turn(t, thread, 1_000), "Late loads", conversationchapter.ChapterTitleModel)
	h.chapter(t, h.turn(t, elsewhere, 1_000), "Acme", conversationchapter.ChapterTitleModel)

	chapters := h.list(t, h.ctx, thread.ID)

	require.Len(t, chapters, 1)
	assert.Equal(t, "Late loads", chapters[0].Title)
}

func TestRetitle_AModelNeverReplacesAPersonsTitle(t *testing.T) {
	h := newHarness(t)
	thread := h.thread(t)
	named := h.chapter(
		t,
		h.turn(t, thread, 1_000),
		"My loads",
		conversationchapter.ChapterTitlePerson,
	)
	asked := h.chapter(
		t,
		h.turn(t, thread, 2_000),
		"Which loads are late?",
		conversationchapter.ChapterTitleQuestion,
	)

	kept, err := h.repo.Retitle(h.ctx, assistantports.RetitleChapterRequest{
		ID:          named.ID,
		ThreadID:    thread.ID,
		TenantInfo:  h.tenant,
		Title:       "Late loads",
		TitleSource: conversationchapter.ChapterTitleModel,
	})
	require.NoError(t, err)
	assert.Equal(t, "My loads", kept.Title)
	assert.Equal(t, conversationchapter.ChapterTitlePerson, kept.TitleSource)
	assert.Equal(t, named.Version, kept.Version)

	renamed, err := h.repo.Retitle(h.ctx, assistantports.RetitleChapterRequest{
		ID:          asked.ID,
		ThreadID:    thread.ID,
		TenantInfo:  h.tenant,
		Title:       "Late loads",
		TitleSource: conversationchapter.ChapterTitleModel,
	})
	require.NoError(t, err)
	assert.Equal(t, "Late loads", renamed.Title)
	assert.Equal(t, conversationchapter.ChapterTitleModel, renamed.TitleSource)
	assert.Greater(
		t,
		renamed.Version,
		asked.Version,
		"a rename is a change a person's edit must see",
	)
}

func TestRetitle_APersonRenamesTheVersionTheyRead(t *testing.T) {
	h := newHarness(t)
	thread := h.thread(t)
	chapter := h.chapter(
		t,
		h.turn(t, thread, 1_000),
		"Late loads",
		conversationchapter.ChapterTitleModel,
	)

	read := chapter.Version
	renamed, err := h.repo.Retitle(h.ctx, assistantports.RetitleChapterRequest{
		ID:          chapter.ID,
		ThreadID:    thread.ID,
		TenantInfo:  h.tenant,
		Title:       "Dallas lane",
		TitleSource: conversationchapter.ChapterTitlePerson,
		Version:     &read,
	})
	require.NoError(t, err)
	assert.Equal(t, "Dallas lane", renamed.Title)
	assert.Equal(t, conversationchapter.ChapterTitlePerson, renamed.TitleSource)

	_, err = h.repo.Retitle(h.ctx, assistantports.RetitleChapterRequest{
		ID:          chapter.ID,
		ThreadID:    thread.ID,
		TenantInfo:  h.tenant,
		Title:       "Houston lane",
		TitleSource: conversationchapter.ChapterTitlePerson,
		Version:     &read,
	})

	var stale *errortypes.Error
	require.ErrorAs(t, err, &stale)
	assert.Equal(t, "version", stale.Field)
	assert.Equal(t, errortypes.ErrVersionMismatch, stale.Code)

	current := h.list(t, h.ctx, thread.ID)
	require.Len(t, current, 1)
	assert.Equal(t, "Dallas lane", current[0].Title, "a stale rename changes nothing")
}

func TestRetitle_AMissingChapterIsNotFound(t *testing.T) {
	h := newHarness(t)
	thread := h.thread(t)

	_, err := h.repo.Retitle(h.ctx, assistantports.RetitleChapterRequest{
		ID:          pulid.MustNew("achp_"),
		ThreadID:    thread.ID,
		TenantInfo:  h.tenant,
		Title:       "Late loads",
		TitleSource: conversationchapter.ChapterTitleModel,
	})

	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestDelete_KeepPersonLeavesAChapterThePersonNamed(t *testing.T) {
	h := newHarness(t)
	thread := h.thread(t)
	named := h.chapter(
		t,
		h.turn(t, thread, 1_000),
		"My loads",
		conversationchapter.ChapterTitlePerson,
	)
	asked := h.chapter(
		t,
		h.turn(t, thread, 2_000),
		"Late loads",
		conversationchapter.ChapterTitleQuestion,
	)

	removed, err := h.repo.Delete(h.ctx, assistantports.DeleteChapterRequest{
		ID:         named.ID,
		ThreadID:   thread.ID,
		TenantInfo: h.tenant,
		KeepPerson: true,
	})
	require.NoError(t, err)
	assert.False(t, removed)

	removed, err = h.repo.Delete(h.ctx, assistantports.DeleteChapterRequest{
		ID:         asked.ID,
		ThreadID:   thread.ID,
		TenantInfo: h.tenant,
		KeepPerson: true,
	})
	require.NoError(t, err)
	assert.True(t, removed)
	assert.Equal(t, []pulid.ID{named.ID}, chapterIDs(h.list(t, h.ctx, thread.ID)))

	removed, err = h.repo.Delete(h.ctx, assistantports.DeleteChapterRequest{
		ID:         named.ID,
		ThreadID:   thread.ID,
		TenantInfo: h.tenant,
	})
	require.NoError(t, err)
	assert.True(t, removed, "the person may remove a chapter they named")
	assert.Empty(t, h.list(t, h.ctx, thread.ID))

	removed, err = h.repo.Delete(h.ctx, assistantports.DeleteChapterRequest{
		ID:         named.ID,
		ThreadID:   thread.ID,
		TenantInfo: h.tenant,
	})
	require.NoError(t, err)
	assert.False(t, removed, "a chapter already gone is reported as not removed")
}

func TestTurnOfMessage_NamesTheTurnThatSavedIt(t *testing.T) {
	h := newHarness(t)
	thread := h.thread(t)
	turn := h.turn(t, thread, 1_000)
	asked := h.say(t, thread, turn.ID, "Which loads are late?", "Three are late.")
	card := h.say(t, thread, "", "A card written outside any turn")

	for _, message := range asked {
		turnID, err := h.repo.TurnOfMessage(h.ctx, assistantports.TurnOfMessageRequest{
			ThreadID:   thread.ID,
			MessageID:  message.ID,
			TenantInfo: h.tenant,
		})
		require.NoError(t, err)
		assert.Equal(t, turn.ID, turnID)
	}

	turnID, err := h.repo.TurnOfMessage(h.ctx, assistantports.TurnOfMessageRequest{
		ThreadID:   thread.ID,
		MessageID:  card[0].ID,
		TenantInfo: h.tenant,
	})
	require.NoError(t, err)
	assert.True(t, turnID.IsNil(), "a card outside a turn has none")

	_, err = h.repo.TurnOfMessage(h.ctx, assistantports.TurnOfMessageRequest{
		ThreadID:   h.thread(t).ID,
		MessageID:  asked[0].ID,
		TenantInfo: h.tenant,
	})
	assert.True(
		t,
		errortypes.IsNotFoundError(err),
		"a message of another conversation is not found",
	)
}

func TestChapters_AnotherTenantSeesNone(t *testing.T) {
	h := newHarness(t)
	thread := h.thread(t)
	turn := h.turn(t, thread, 1_000)
	asked := h.say(t, thread, turn.ID, "Which loads are late?", "Three are late.")
	chapter := h.chapter(t, turn, "Late loads", conversationchapter.ChapterTitleModel)

	chapters, err := h.repo.List(h.other, assistantports.ListChaptersRequest{
		ThreadID:   thread.ID,
		TenantInfo: h.otherTenant,
	})
	require.NoError(t, err)
	assert.Empty(t, chapters)

	count, err := h.repo.Count(h.other, assistantports.ListChaptersRequest{
		ThreadID:   thread.ID,
		TenantInfo: h.otherTenant,
	})
	require.NoError(t, err)
	assert.Zero(t, count)

	_, err = h.repo.Get(h.other, assistantports.GetChapterRequest{
		ID:         chapter.ID,
		ThreadID:   thread.ID,
		TenantInfo: h.otherTenant,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	_, err = h.repo.TurnOfMessage(h.other, assistantports.TurnOfMessageRequest{
		ThreadID:   thread.ID,
		MessageID:  asked[0].ID,
		TenantInfo: h.otherTenant,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	removed, err := h.repo.Delete(h.other, assistantports.DeleteChapterRequest{
		ID:         chapter.ID,
		ThreadID:   thread.ID,
		TenantInfo: h.otherTenant,
	})
	require.NoError(t, err)
	assert.False(t, removed)

	_, err = h.repo.Retitle(h.other, assistantports.RetitleChapterRequest{
		ID:          chapter.ID,
		ThreadID:    thread.ID,
		TenantInfo:  h.otherTenant,
		Title:       "Taken",
		TitleSource: conversationchapter.ChapterTitlePerson,
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	mine := h.list(t, h.ctx, thread.ID)
	require.Len(t, mine, 1)
	assert.Equal(t, "Late loads", mine[0].Title, "the other tenant changed nothing")
}
