package assistantchapterservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Chapters      assistantports.ChapterRepository
	Conversations assistantports.ConversationRepository
	Turns         assistantports.AssistantTurnRepository
	Controls      tenantports.AgentControlRepository
	Scheduler     assistantports.ChapterScheduler `optional:"true"`
	Realtime      realtimeports.RealtimeService   `optional:"true"`
	Logger        *zap.Logger
}

type Service struct {
	chapters      assistantports.ChapterRepository
	conversations assistantports.ConversationRepository
	turns         assistantports.AssistantTurnRepository
	controls      tenantports.AgentControlRepository
	scheduler     assistantports.ChapterScheduler
	realtime      realtimeports.RealtimeService
	l             *zap.Logger
}

func New(p Params) assistantports.AssistantChapterService {
	return &Service{
		chapters:      p.Chapters,
		conversations: p.Conversations,
		turns:         p.Turns,
		controls:      p.Controls,
		scheduler:     p.Scheduler,
		realtime:      p.Realtime,
		l:             p.Logger.Named("service.assistant-chapter"),
	}
}

func (s *Service) List(
	ctx context.Context,
	req assistantports.ThreadChaptersRequest,
) ([]*conversation.Chapter, error) {
	if err := s.owned(ctx, req.ThreadID, req.UserID, req.TenantInfo); err != nil {
		return nil, err
	}

	return s.chapters.List(ctx, assistantports.ListChaptersRequest{
		ThreadID:   req.ThreadID,
		TenantInfo: req.TenantInfo,
	})
}

func (s *Service) Open(
	ctx context.Context,
	req assistantports.OpenChapterRequest,
) (*conversation.Chapter, error) {
	if err := s.owned(ctx, req.ThreadID, req.UserID, req.TenantInfo); err != nil {
		return nil, err
	}

	turnID, err := s.chapters.TurnOfMessage(ctx, assistantports.TurnOfMessageRequest{
		ThreadID:   req.ThreadID,
		MessageID:  req.MessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if turnID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"A chapter starts where a question was asked, and this message was not part of one.",
		)
	}

	if existing, fErr := s.chapters.ForTurn(ctx, assistantports.ChapterForTurnRequest{
		ThreadID:   req.ThreadID,
		TurnID:     turnID,
		TenantInfo: req.TenantInfo,
	}); fErr != nil {
		return nil, fErr
	} else if existing != nil {
		return existing, nil
	}

	count, err := s.chapters.Count(ctx, assistantports.ListChaptersRequest{
		ThreadID:   req.ThreadID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if count >= conversation.MaxChaptersPerThread {
		return nil, errortypes.NewBusinessError(
			"This conversation has as many chapters as it can hold. Merge some before starting another.",
		)
	}

	question, _, err := s.turnText(ctx, req.ThreadID, turnID, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	chapter, err := s.create(ctx, &conversation.Chapter{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ThreadID:       req.ThreadID,
		TurnID:         turnID,
		Title:          conversation.ChapterTitleFrom(question),
		TitleSource:    conversation.ChapterTitleQuestion,
	})
	if err != nil {
		return nil, err
	}

	s.nameQuestionChapters(ctx, req, chapter)

	return chapter, nil
}

func (s *Service) nameQuestionChapters(
	ctx context.Context,
	req assistantports.OpenChapterRequest,
	opened *conversation.Chapter,
) {
	turn := &assistantports.ChapterTurnRequest{
		TenantInfo: req.TenantInfo,
		ThreadID:   req.ThreadID,
		UserID:     req.UserID,
	}
	s.cueNaming(ctx, turn, opened.TurnID)

	chapters, err := s.chapters.List(ctx, assistantports.ListChaptersRequest{
		ThreadID:   req.ThreadID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		s.l.Warn("could not read the chapters to name beside a new one",
			zap.String("thread", req.ThreadID.String()), zap.Error(err))

		return
	}
	for _, chapter := range chapters {
		if chapter.ID != opened.ID && chapter.TitleSource == conversation.ChapterTitleQuestion {
			s.cueNaming(ctx, turn, chapter.TurnID)
		}
	}
}

func (s *Service) cueNaming(
	ctx context.Context,
	turn *assistantports.ChapterTurnRequest,
	turnID pulid.ID,
) {
	if s.scheduler == nil || turnID.IsNil() {
		return
	}
	s.scheduler.Cue(ctx, &assistantports.ChapterTurnRequest{
		TenantInfo: turn.TenantInfo,
		ThreadID:   turn.ThreadID,
		UserID:     turn.UserID,
		TurnID:     turnID,
		Phase:      assistantports.ChapterPhaseRequested,
	})
}

func (s *Service) Rename(
	ctx context.Context,
	req assistantports.RenameChapterRequest,
) (*conversation.Chapter, error) {
	if err := s.owned(ctx, req.ThreadID, req.UserID, req.TenantInfo); err != nil {
		return nil, err
	}

	title := strings.Join(strings.Fields(req.Title), " ")
	me := errortypes.NewMultiError()
	conversation.ValidateChapterTitle("title", title, me)
	if me.HasErrors() {
		return nil, me
	}

	chapter, err := s.chapters.Retitle(ctx, assistantports.RetitleChapterRequest{
		ID:          req.ID,
		ThreadID:    req.ThreadID,
		TenantInfo:  req.TenantInfo,
		Title:       title,
		TitleSource: conversation.ChapterTitlePerson,
		Version:     req.Version,
	})
	if err != nil {
		return nil, err
	}
	s.announce(ctx, req.TenantInfo, req.UserID, req.ThreadID)

	return chapter, nil
}

func (s *Service) Remove(ctx context.Context, req assistantports.RemoveChapterRequest) error {
	if err := s.owned(ctx, req.ThreadID, req.UserID, req.TenantInfo); err != nil {
		return err
	}

	removed, err := s.chapters.Delete(ctx, assistantports.DeleteChapterRequest{
		ID:         req.ID,
		ThreadID:   req.ThreadID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	if !removed {
		return errortypes.NewNotFoundError("Chapter not found in this conversation")
	}
	s.announce(ctx, req.TenantInfo, req.UserID, req.ThreadID)

	return nil
}

func (s *Service) Span(
	ctx context.Context,
	req assistantports.ChapterSpanRequest,
) (*assistantports.ChapterSpan, error) {
	chapters, err := s.List(ctx, assistantports.ThreadChaptersRequest{
		ThreadID:   req.ThreadID,
		UserID:     req.UserID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return SpanOf(chapters, req.ID)
}

func SpanOf(chapters []*conversation.Chapter, id pulid.ID) (*assistantports.ChapterSpan, error) {
	idx := slices.IndexFunc(chapters, func(c *conversation.Chapter) bool { return c.ID == id })
	if idx < 0 {
		return nil, errortypes.NewNotFoundError("Chapter not found in this conversation")
	}
	chapter := chapters[idx]
	if chapter.StartSequence == nil {
		return nil, errortypes.NewBusinessError(
			"This chapter has nothing saved in it yet. Wait for its reply to finish.",
		)
	}

	span := &assistantports.ChapterSpan{Chapter: chapter, From: int(*chapter.StartSequence)}
	for _, next := range chapters[idx+1:] {
		if next.StartSequence != nil && int(*next.StartSequence) > span.From {
			through := int(*next.StartSequence) - 1
			span.Through = &through

			break
		}
	}

	return span, nil
}

func (s *Service) owned(
	ctx context.Context,
	threadID, userID pulid.ID,
	tenant pagination.TenantInfo,
) error {
	if _, err := s.conversations.GetThread(ctx, assistantports.GetThreadRequest{
		ID:         threadID,
		UserID:     userID,
		TenantInfo: tenant,
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) create(
	ctx context.Context,
	chapter *conversation.Chapter,
) (*conversation.Chapter, error) {
	me := errortypes.NewMultiError()
	chapter.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	saved, err := s.chapters.Create(ctx, chapter)
	if err != nil {
		return nil, fmt.Errorf("save the chapter: %w", err)
	}

	return saved, nil
}

func (s *Service) turnText(
	ctx context.Context,
	threadID, turnID pulid.ID,
	tenant pagination.TenantInfo,
) (question, reply string, err error) {
	messages, err := s.conversations.ListMessages(ctx, assistantports.ListMessagesRequest{
		ThreadID:     threadID,
		TenantInfo:   tenant,
		TurnID:       turnID,
		ExcludeKinds: conversation.ModelHiddenKinds(),
	})
	if err != nil {
		return "", "", fmt.Errorf("read the turn's messages: %w", err)
	}

	for idx := range messages {
		message := &messages[idx]
		switch {
		case message.Role == conversation.RoleUser && question == "":
			question = message.Content
		case message.Role == conversation.RoleAssistant && strings.TrimSpace(message.Content) != "":
			reply = message.Content
		}
	}

	return question, reply, nil
}

func (s *Service) announce(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID, threadID pulid.ID,
) {
	if s.realtime == nil {
		return
	}
	if err := s.realtime.PublishResourceInvalidation(
		context.WithoutCancel(ctx),
		&realtimeports.PublishResourceInvalidationRequest{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			AudienceUserID: userID,
			Resource:       assistantports.ChaptersResource,
			Action:         "updated",
			RecordID:       threadID,
			Entity:         map[string]string{"threadId": threadID.String()},
		},
	); err != nil {
		s.l.Debug("could not announce a change to a conversation's chapters",
			zap.String("thread", threadID.String()), zap.Error(err))
	}
}
