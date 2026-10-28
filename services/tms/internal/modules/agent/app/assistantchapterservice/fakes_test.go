package assistantchapterservice

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type fakeChapters struct {
	chapters []*conversation.Chapter
	turnOf   map[pulid.ID]pulid.ID
	created  []*conversation.Chapter
	retitled []assistantports.RetitleChapterRequest
	deleted  []assistantports.DeleteChapterRequest
}

func (f *fakeChapters) List(
	context.Context,
	assistantports.ListChaptersRequest,
) ([]*conversation.Chapter, error) {
	return slices.Clone(f.chapters), nil
}

func (f *fakeChapters) Get(
	_ context.Context,
	req assistantports.GetChapterRequest,
) (*conversation.Chapter, error) {
	for _, chapter := range f.chapters {
		if chapter.ID == req.ID {
			return chapter, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Chapter not found")
}

func (f *fakeChapters) ForTurn(
	_ context.Context,
	req assistantports.ChapterForTurnRequest,
) (*conversation.Chapter, error) {
	for _, chapter := range f.chapters {
		if chapter.TurnID == req.TurnID {
			return chapter, nil
		}
	}

	return nil, nil
}

func (f *fakeChapters) Count(context.Context, assistantports.ListChaptersRequest) (int, error) {
	return len(f.chapters), nil
}

func (f *fakeChapters) Create(
	_ context.Context,
	chapter *conversation.Chapter,
) (*conversation.Chapter, error) {
	if chapter.ID.IsNil() {
		chapter.ID = pulid.MustNew("achp_")
	}
	f.chapters = append(f.chapters, chapter)
	f.created = append(f.created, chapter)

	return chapter, nil
}

func (f *fakeChapters) Retitle(
	_ context.Context,
	req assistantports.RetitleChapterRequest,
) (*conversation.Chapter, error) {
	f.retitled = append(f.retitled, req)
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
	req assistantports.DeleteChapterRequest,
) (bool, error) {
	f.deleted = append(f.deleted, req)
	idx := slices.IndexFunc(f.chapters, func(c *conversation.Chapter) bool {
		return c.ID == req.ID &&
			(!req.KeepPerson || c.TitleSource != conversation.ChapterTitlePerson)
	})
	if idx < 0 {
		return false, nil
	}
	f.chapters = slices.Delete(f.chapters, idx, idx+1)

	return true, nil
}

func (f *fakeChapters) TurnOfMessage(
	_ context.Context,
	req assistantports.TurnOfMessageRequest,
) (pulid.ID, error) {
	turnID, ok := f.turnOf[req.MessageID]
	if !ok {
		return "", errortypes.NewNotFoundError("Message not found in this conversation")
	}

	return turnID, nil
}

type fakeConversations struct {
	assistantports.ConversationRepository

	thread *conversation.Thread
	recent []conversation.Message
	byTurn map[pulid.ID][]conversation.Message
	listed []assistantports.ListMessagesRequest
}

func (f *fakeConversations) GetThread(
	_ context.Context,
	req assistantports.GetThreadRequest,
) (*conversation.Thread, error) {
	if req.ID != f.thread.ID || req.UserID != f.thread.UserID {
		return nil, errortypes.NewNotFoundError("Conversation not found")
	}

	return f.thread, nil
}

func (f *fakeConversations) ListMessages(
	_ context.Context,
	req assistantports.ListMessagesRequest,
) ([]conversation.Message, error) {
	f.listed = append(f.listed, req)
	if req.TurnID.IsNotNil() {
		return f.byTurn[req.TurnID], nil
	}

	return f.recent, nil
}

type fakeTurns struct {
	assistantports.AssistantTurnRepository

	turns map[pulid.ID]*conversation.AssistantTurn
}

func (f *fakeTurns) GetByID(
	_ context.Context,
	req assistantports.GetAssistantTurnRequest,
) (*conversation.AssistantTurn, error) {
	turn, ok := f.turns[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Turn not found")
	}

	return turn, nil
}

type fakeControls struct {
	tenantports.AgentControlRepository

	control *tenant.AgentControl
}

func (f *fakeControls) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	return f.control, nil
}

type fakeScheduler struct {
	cues []*assistantports.ChapterTurnRequest
}

func (f *fakeScheduler) Cue(_ context.Context, req *assistantports.ChapterTurnRequest) {
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
	svc           assistantports.AssistantChapterService
	chapters      *fakeChapters
	conversations *fakeConversations
	turns         *fakeTurns
	control       *tenant.AgentControl
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
		turns:     &fakeTurns{turns: map[pulid.ID]*conversation.AssistantTurn{}},
		control:   &tenant.AgentControl{},
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
	origin conversation.AssistantTurnOrigin,
	status conversation.AssistantTurnStatus,
	input string,
) *conversation.AssistantTurn {
	turn := &conversation.AssistantTurn{
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
	source conversation.ChapterTitleSource,
	start *int64,
) *conversation.Chapter {
	chapter := &conversation.Chapter{
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
	phase assistantports.ChapterPhase,
) *assistantports.ChapterTurnRequest {
	return &assistantports.ChapterTurnRequest{
		TenantInfo: f.tenant,
		ThreadID:   f.thread.ID,
		UserID:     f.tenant.UserID,
		TurnID:     turnID,
		Phase:      phase,
	}
}

func sequence(value int64) *int64 { return &value }
