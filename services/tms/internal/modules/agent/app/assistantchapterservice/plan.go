package assistantchapterservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationchapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantchapterports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantturnports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	skipNoTurn        = "the turn is gone"
	skipContinues     = "the turn continues its chapter"
	skipAlreadyOpen   = "the turn already opens a chapter"
	skipNoQuestion    = "the turn asks nothing"
	skipLimit         = "the conversation has as many chapters as it may"
	skipAutoOff       = "the organization turned automatic chapters off"
	skipNoChapter     = "the turn opens no chapter"
	skipNamed         = "the chapter keeps its title"
	shortQuestionWord = 5
)

func (s *Service) Plan(
	ctx context.Context,
	req *assistantchapterports.ChapterTurnRequest,
) (*assistantchapterports.ChapterPlan, error) {
	plan := &assistantchapterports.ChapterPlan{Turn: *req}

	turn, err := s.turns.GetByID(ctx, assistantturnports.GetAssistantTurnRequest{
		ID:         req.TurnID,
		TenantInfo: req.TenantInfo,
		UserID:     req.UserID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			plan.Skip = skipNoTurn

			return plan, nil
		}

		return nil, fmt.Errorf("read the turn: %w", err)
	}

	control, err := s.controls.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, fmt.Errorf("read the organization's agent control: %w", err)
	}
	auto := !control.AutoChaptersOff

	existing, err := s.chapters.ForTurn(ctx, assistantchapterports.ChapterForTurnRequest{
		ThreadID:   req.ThreadID,
		TurnID:     req.TurnID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	switch req.Phase {
	case assistantchapterports.ChapterPhaseOpened:
		return s.planOpened(ctx, plan, turn, existing, auto)
	case assistantchapterports.ChapterPhaseAnswered:
		return s.planAnswered(ctx, plan, turn, existing, auto)
	case assistantchapterports.ChapterPhaseRequested:
		return s.planRequested(ctx, plan, existing, auto)
	default:
		return nil, fmt.Errorf("unknown chapter phase %q", req.Phase)
	}
}

func (s *Service) planOpened(
	ctx context.Context,
	plan *assistantchapterports.ChapterPlan,
	turn *conversationturn.AssistantTurn,
	existing *conversationchapter.Chapter,
	auto bool,
) (*assistantchapterports.ChapterPlan, error) {
	question := strings.TrimSpace(turn.Input)
	switch {
	case existing != nil:
		plan.Skip = skipAlreadyOpen
	case turn.Origin != conversationturn.AssistantTurnOriginPerson &&
		turn.Origin != conversationturn.AssistantTurnOriginScheduled:
		plan.Skip = skipContinues
	case question == "":
		plan.Skip = skipNoQuestion
	}
	if plan.Skipped() {
		return plan, nil
	}

	chapters, err := s.chapters.List(ctx, assistantchapterports.ListChaptersRequest{
		ThreadID:   plan.Turn.ThreadID,
		TenantInfo: plan.Turn.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if len(chapters) >= conversationchapter.MaxChaptersPerThread {
		plan.Skip = skipLimit

		return plan, nil
	}

	plan.Evidence = []string{question}
	plan.NameEarlier = earlierToName(chapters)
	if turn.Origin == conversationturn.AssistantTurnOriginScheduled {
		plan.Opens = true
		plan.Fallback = conversationchapter.ChapterTitleFrom(question)
		plan.FallbackSource = conversationchapter.ChapterTitleSchedule
		plan.ModelSource = conversationchapter.ChapterTitleSchedule
		if auto {
			plan.Request = nameRequest(&plan.Turn, question, "")
		}

		return plan, nil
	}

	if len(chapters) == 0 {
		plan.Opens = true
		plan.Fallback = conversationchapter.ChapterTitleFrom(question)
		plan.FallbackSource = conversationchapter.ChapterTitleQuestion

		return plan, nil
	}

	if !auto {
		plan.Skip = skipAutoOff

		return plan, nil
	}

	recent, err := s.conversations.ListMessages(ctx, conversationports.ListMessagesRequest{
		ThreadID:        plan.Turn.ThreadID,
		TenantInfo:      plan.Turn.TenantInfo,
		Limit:           recentMessages,
		ExcludeKinds:    conversation.ModelHiddenKinds(),
		SinceCompaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("read how the conversation went: %w", err)
	}

	plan.Decides = true
	plan.ModelSource = conversationchapter.ChapterTitleModel
	plan.Evidence = append(plan.Evidence, contents(recent)...)
	plan.Request = decideRequest(&plan.Turn, chapters[len(chapters)-1].Title, recent, question)

	return plan, nil
}

func (s *Service) planAnswered(
	ctx context.Context,
	plan *assistantchapterports.ChapterPlan,
	turn *conversationturn.AssistantTurn,
	existing *conversationchapter.Chapter,
	auto bool,
) (*assistantchapterports.ChapterPlan, error) {
	if existing == nil {
		plan.Skip = skipNoChapter

		return plan, nil
	}
	plan.ChapterID = existing.ID

	if existing.StartSequence == nil ||
		turn.Status == conversation.AssistantTurnStatusRefused {
		plan.Remove = existing.TitleSource != conversationchapter.ChapterTitlePerson
		if !plan.Remove {
			plan.Skip = skipNamed
		}

		return plan, nil
	}

	if !auto || turn.Status != conversation.AssistantTurnStatusCompleted ||
		!worthRenaming(existing, turn.Input) {
		plan.Skip = skipNamed

		return plan, nil
	}

	return s.planRename(ctx, plan, existing)
}

func (s *Service) planRequested(
	ctx context.Context,
	plan *assistantchapterports.ChapterPlan,
	existing *conversationchapter.Chapter,
	auto bool,
) (*assistantchapterports.ChapterPlan, error) {
	switch {
	case existing == nil:
		plan.Skip = skipNoChapter
	case !auto || existing.TitleSource != conversationchapter.ChapterTitleQuestion:
		plan.Skip = skipNamed
	}
	if plan.Skipped() {
		return plan, nil
	}
	plan.ChapterID = existing.ID

	return s.planRename(ctx, plan, existing)
}

func (s *Service) planRename(
	ctx context.Context,
	plan *assistantchapterports.ChapterPlan,
	existing *conversationchapter.Chapter,
) (*assistantchapterports.ChapterPlan, error) {
	question, reply, err := s.turnText(
		ctx,
		plan.Turn.ThreadID,
		existing.TurnID,
		plan.Turn.TenantInfo,
	)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(question) == "" {
		plan.Skip = skipNoQuestion

		return plan, nil
	}

	plan.ModelSource = conversationchapter.ChapterTitleModel
	plan.Evidence = []string{question, reply}
	plan.Request = nameRequest(&plan.Turn, question, reply)

	return plan, nil
}

func worthRenaming(chapter *conversationchapter.Chapter, question string) bool {
	return chapter.TitleSource == conversationchapter.ChapterTitleModel &&
		len(strings.Fields(question)) < shortQuestionWord
}

func earlierToName(chapters []*conversationchapter.Chapter) pulid.ID {
	if len(chapters) == 0 {
		return ""
	}
	if last := chapters[len(chapters)-1]; last.TitleSource == conversationchapter.ChapterTitleQuestion {
		return last.TurnID
	}

	return ""
}

func (s *Service) Apply(
	ctx context.Context,
	plan *assistantchapterports.ChapterPlan,
	result *llmports.StructuredCompletionResult,
) (*assistantchapterports.ChapterOutcome, error) {
	outcome := &assistantchapterports.ChapterOutcome{ChapterID: plan.ChapterID}
	if plan.Skipped() {
		return outcome, nil
	}
	turn := &plan.Turn

	if plan.Remove {
		removed, err := s.chapters.Delete(ctx, assistantchapterports.DeleteChapterRequest{
			ID:         plan.ChapterID,
			ThreadID:   turn.ThreadID,
			TenantInfo: turn.TenantInfo,
			KeepPerson: true,
		})
		if err != nil {
			return nil, err
		}
		outcome.Removed = removed
		if removed {
			s.announce(ctx, turn.TenantInfo, turn.UserID, turn.ThreadID)
		}

		return outcome, nil
	}

	answer := s.read(plan, result)

	switch {
	case plan.Decides:
		if answer.NewChapter == nil || !*answer.NewChapter || answer.Title == "" {
			return outcome, nil
		}

		return s.open(ctx, plan, answer.Title, plan.ModelSource, outcome)
	case plan.Opens:
		if answer.Title == "" {
			outcome.Fallback = plan.Request != nil

			return s.open(ctx, plan, plan.Fallback, plan.FallbackSource, outcome)
		}

		return s.open(ctx, plan, answer.Title, plan.ModelSource, outcome)
	case answer.Title != "":
		if _, err := s.chapters.Retitle(ctx, assistantchapterports.RetitleChapterRequest{
			ID:          plan.ChapterID,
			ThreadID:    turn.ThreadID,
			TenantInfo:  turn.TenantInfo,
			Title:       answer.Title,
			TitleSource: plan.ModelSource,
		}); err != nil {
			if errortypes.IsNotFoundError(err) {
				return outcome, nil
			}

			return nil, err
		}
		outcome.Retitled = true
		s.announce(ctx, turn.TenantInfo, turn.UserID, turn.ThreadID)
	}

	return outcome, nil
}

func (s *Service) open(
	ctx context.Context,
	plan *assistantchapterports.ChapterPlan,
	title string,
	source conversationchapter.ChapterTitleSource,
	outcome *assistantchapterports.ChapterOutcome,
) (*assistantchapterports.ChapterOutcome, error) {
	turn := &plan.Turn
	chapter, err := s.create(ctx, &conversationchapter.Chapter{
		OrganizationID: turn.TenantInfo.OrgID,
		BusinessUnitID: turn.TenantInfo.BuID,
		ThreadID:       turn.ThreadID,
		TurnID:         turn.TurnID,
		Title:          title,
		TitleSource:    source,
	})
	if err != nil {
		var fk *errortypes.Error
		if errors.As(err, &fk) && fk.Code == errortypes.ErrInvalidReference {
			return outcome, nil
		}

		return nil, err
	}
	outcome.ChapterID = chapter.ID
	outcome.Opened = true
	s.announce(ctx, turn.TenantInfo, turn.UserID, turn.ThreadID)
	s.cueNaming(ctx, turn, plan.NameEarlier)

	return outcome, nil
}

func (s *Service) read(
	plan *assistantchapterports.ChapterPlan,
	result *llmports.StructuredCompletionResult,
) chapterAnswer {
	if result == nil || strings.TrimSpace(result.Text) == "" {
		return chapterAnswer{}
	}

	var answer chapterAnswer
	if err := sonic.UnmarshalString(result.Text, &answer); err != nil {
		s.l.Warn("the chapter model's answer could not be read",
			zap.String("thread", plan.Turn.ThreadID.String()),
			zap.String("model", result.ModelIdentifier),
			zap.Error(err))

		return chapterAnswer{}
	}

	answer.Title = conversationchapter.CleanChapterTitle(answer.Title)
	if answer.Title != "" && !supported(answer.Title, plan.Evidence) {
		s.l.Info("the chapter model named a number the conversation does not hold",
			zap.String("thread", plan.Turn.ThreadID.String()),
			zap.String("model", result.ModelIdentifier))
		answer.Title = ""
	}

	return answer
}

func supported(title string, evidence []string) bool {
	return numberguard.CheckNumbers(title, numberguard.SupportedFromText(evidence...)).OK
}

func contents(messages []conversation.Message) []string {
	out := make([]string, 0, len(messages))
	for idx := range messages {
		if text := strings.TrimSpace(messages[idx].Content); text != "" {
			out = append(out, text)
		}
	}

	return out
}
