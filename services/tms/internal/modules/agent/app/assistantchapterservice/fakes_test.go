package assistantchapterservice

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationchapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantchapterports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantturnports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type fakeChapters struct {
	chapters []*conversationchapter.Chapter
	turnOf   map[pulid.ID]pulid.ID
	created  []*conversationchapter.Chapter
	retitled []assistantchapterports.RetitleChapterRequest
	deleted  []assistantchapterports.DeleteChapterRequest
}

func (f *fakeChapters) List(
	context.Context,
	assistantchapterports.ListChaptersRequest,
) ([]*conversationchapter.Chapter, error) {
	return slices.Clone(f.chapters), nil
}

func (f *fakeChapters) Get(
	_ context.Context,
	req *assistantchapterports.GetChapterRequest,
) (*conversationchapter.Chapter, error) {
	for _, chapter := range f.chapters {
		if chapter.ID == req.ID {
			return chapter, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Chapter not found")
}

func (f *fakeChapters) ForTurn(
	_ context.Context,
	req *assistantchapterports.ChapterForTurnRequest,
) (*conversationchapter.Chapter, error) {
	for _, chapter := range f.chapters {
		if chapter.TurnID == req.TurnID {
			return chapter, nil
		}
	}

	return nil, nil
}

func (f *fakeChapters) Count(
	context.Context,
	assistantchapterports.ListChaptersRequest,
) (int, error) {
	return len(f.chapters), nil
}

func (f *fakeChapters) Create(
	_ context.Context,
	chapter *conversationchapter.Chapter,
) (*conversationchapter.Chapter, error) {
	if chapter.ID.IsNil() {
		chapter.ID = pulid.MustNew("achp_")
	}
	f.chapters = append(f.chapters, chapter)
	f.created = append(f.created, chapter)

	return chapter, nil
}

func (f *fakeChapters) Retitle(
	_ context.Context,
	req *assistantchapterports.RetitleChapterRequest,
) (*conversationchapter.Chapter, error) {
	f.retitled = append(f.retitled, *req)
	for _, chapter := range f.chapters {
		if chapter.ID == req.ID {
			chapter.Title = req.Title
			chapter.TitleSource = req.TitleSource

			return chapter, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Chapter not found")
}

func (f *fakeChapters) Delete(
	_ context.Context,
	req *assistantchapterports.DeleteChapterRequest,
) (bool, error) {
	f.deleted = append(f.deleted, *req)
	idx := slices.IndexFunc(f.chapters, func(c *conversationchapter.Chapter) bool {
		return c.ID == req.ID &&
			(!req.KeepPerson || c.TitleSource != conversationchapter.ChapterTitlePerson)
	})
	if idx < 0 {
		return false, nil
	}
	f.chapters = slices.Delete(f.chapters, idx, idx+1)

	return true, nil
}

func (f *fakeChapters) TurnOfMessage(
	_ context.Context,
	req *assistantchapterports.TurnOfMessageRequest,
) (pulid.ID, error) {
	turnID, ok := f.turnOf[req.MessageID]
	if !ok {
		return "", errortypes.NewNotFoundError("Message not found in this conversation")
	}

	return turnID, nil
}

type fakeConversations struct {
	conversationports.ConversationRepository

	thread *conversation.Thread
	recent []conversation.Message
	byTurn map[pulid.ID][]conversation.Message
	listed []conversationports.ListMessagesRequest
}

func (f *fakeConversations) GetThread(
	_ context.Context,
	req *conversationports.GetThreadRequest,
) (*conversation.Thread, error) {
	if req.ID != f.thread.ID || req.UserID != f.thread.UserID {
		return nil, errortypes.NewNotFoundError("Conversation not found")
	}

	return f.thread, nil
}

func (f *fakeConversations) ListMessages(
	_ context.Context,
	req *conversationports.ListMessagesRequest,
) ([]conversation.Message, error) {
	f.listed = append(f.listed, *req)
	if req.TurnID.IsNotNil() {
		return f.byTurn[req.TurnID], nil
	}

	return f.recent, nil
}

type fakeTurns struct {
	assistantturnports.AssistantTurnRepository

	turns map[pulid.ID]*conversationturn.AssistantTurn
}

func (f *fakeTurns) GetByID(
	_ context.Context,
	req *assistantturnports.GetAssistantTurnRequest,
) (*conversationturn.AssistantTurn, error) {
	turn, ok := f.turns[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Turn not found")
	}

	return turn, nil
}

type fakeControls struct {
	tenantports.AgentControlRepository

	control *agentcontrol.AgentControl
}

func (f *fakeControls) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*agentcontrol.AgentControl, error) {
	return f.control, nil
}

type fakeScheduler struct {
	cues []*assistantchapterports.ChapterTurnRequest
}

func (f *fakeScheduler) Cue(_ context.Context, req *assistantchapterports.ChapterTurnRequest) {
	f.cues = append(f.cues, req)
}

type fakeRealtime struct {
	published []*realtimeports.PublishResourceInvalidationRequest
}

func (f *fakeRealtime) PublishResourceInvalidation(
	_ context.Context,
	req *realtimeports.PublishResourceInvalidationRequest,
) error {
	f.published = append(f.published, req)

	return nil
}

type fixture struct {
	svc           assistantchapterports.AssistantChapterService
	chapters      *fakeChapters
	conversations *fakeConversations
	turns         *fakeTurns
	control       *agentcontrol.AgentControl
	scheduler     *fakeScheduler
	realtime      *fakeRealtime
	tenant        pagination.TenantInfo
	thread        *conversation.Thread
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	info := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	thread := &conversation.Thread{
		ID:             pulid.MustNew("athr_"),
		OrganizationID: info.OrgID,
		BusinessUnitID: info.BuID,
		UserID:         info.UserID,
	}
	f := &fixture{
		chapters: &fakeChapters{turnOf: map[pulid.ID]pulid.ID{}},
		conversations: &fakeConversations{
			thread: thread,
			byTurn: map[pulid.ID][]conversation.Message{},
		},
		turns:     &fakeTurns{turns: map[pulid.ID]*conversationturn.AssistantTurn{}},
		control:   &agentcontrol.AgentControl{},
		scheduler: &fakeScheduler{},
		realtime:  &fakeRealtime{},
		tenant:    info,
		thread:    thread,
	}
	f.svc = New(Params{
		Chapters:      f.chapters,
		Conversations: f.conversations,
		Turns:         f.turns,
		Controls:      &fakeControls{control: f.control},
		Scheduler:     f.scheduler,
		Realtime:      f.realtime,
		Logger:        zap.NewNop(),
	})

	return f
}

func (f *fixture) turn(
	origin conversationturn.AssistantTurnOrigin,
	status conversation.AssistantTurnStatus,
	input string,
) *conversationturn.AssistantTurn {
	turn := &conversationturn.AssistantTurn{
		ID:             pulid.MustNew("atrn_"),
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		ThreadID:       f.thread.ID,
		UserID:         f.tenant.UserID,
		Origin:         origin,
		Status:         status,
		Input:          input,
	}
	f.turns.turns[turn.ID] = turn

	return turn
}

func (f *fixture) chapter(
	turnID pulid.ID,
	title string,
	source conversationchapter.ChapterTitleSource,
	start *int64,
) *conversationchapter.Chapter {
	chapter := &conversationchapter.Chapter{
		ID:             pulid.MustNew("achp_"),
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		ThreadID:       f.thread.ID,
		TurnID:         turnID,
		Title:          title,
		TitleSource:    source,
		StartSequence:  start,
	}
	f.chapters.chapters = append(f.chapters.chapters, chapter)

	return chapter
}

func (f *fixture) request(
	turnID pulid.ID,
	phase assistantchapterports.ChapterPhase,
) *assistantchapterports.ChapterTurnRequest {
	return &assistantchapterports.ChapterTurnRequest{
		TenantInfo: f.tenant,
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		TurnID:     turnID,
		Phase:      phase,
	}
}

func sequence(value int64) *int64 { return &value }
