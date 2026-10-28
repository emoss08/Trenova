package assistantchapterservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
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
	req *assistantports.ChapterTurnRequest,
) (*assistantports.ChapterPlan, error) {
	plan := &assistantports.ChapterPlan{Turn: *req}

	turn, err := s.turns.GetByID(ctx, assistantports.GetAssistantTurnRequest{
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

	existing, err := s.chapters.ForTurn(ctx, assistantports.ChapterForTurnRequest{
		ThreadID:   req.ThreadID,
		TurnID:     req.TurnID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	switch req.Phase {
	case assistantports.ChapterPhaseOpened:
		return s.planOpened(ctx, plan, turn, existing, auto)
	case assistantports.ChapterPhaseAnswered:
		return s.planAnswered(ctx, plan, turn, existing, auto)
	case assistantports.ChapterPhaseRequested:
		return s.planRequested(ctx, plan, existing, auto)
	default:
		return nil, fmt.Errorf("unknown chapter phase %q", req.Phase)
	}
}

func (s *Service) planOpened(
	ctx context.Context,
	plan *assistantports.ChapterPlan,
	turn *conversation.AssistantTurn,
	existing *conversation.Chapter,
	auto bool,
) (*assistantports.ChapterPlan, error) {
	question := strings.TrimSpace(turn.Input)
	switch {
	case existing != nil:
		plan.Skip = skipAlreadyOpen
	case turn.Origin != conversation.AssistantTurnOriginPerson &&
		turn.Origin != conversation.AssistantTurnOriginScheduled:
		plan.Skip = skipContinues
	case question == "":
		plan.Skip = skipNoQuestion
	}
	if plan.Skipped() {
		return plan, nil
	}

	chapters, err := s.chapters.List(ctx, assistantports.ListChaptersRequest{
		ThreadID:   plan.Turn.ThreadID,
		TenantInfo: plan.Turn.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if len(chapters) >= conversation.MaxChaptersPerThread {
		plan.Skip = skipLimit

		return plan, nil
	}

	plan.Evidence = []string{question}
	plan.NameEarlier = earlierToName(chapters)
	if turn.Origin == conversation.AssistantTurnOriginScheduled {
		plan.Opens = true
		plan.Fallback = conversation.ChapterTitleFrom(question)
		plan.FallbackSource = conversation.ChapterTitleSchedule
		plan.ModelSource = conversation.ChapterTitleSchedule
		if auto {
			plan.Request = nameRequest(&plan.Turn, question, "")
		}

		return plan, nil
	}

	if len(chapters) == 0 {
		plan.Opens = true
		plan.Fallback = conversation.ChapterTitleFrom(question)
		plan.FallbackSource = conversation.ChapterTitleQuestion

		return plan, nil
	}

	if !auto {
		plan.Skip = skipAutoOff

		return plan, nil
	}

	recent, err := s.conversations.ListMessages(ctx, assistantports.ListMessagesRequest{
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
	plan.ModelSource = conversation.ChapterTitleModel
	plan.Evidence = append(plan.Evidence, contents(recent)...)
	plan.Request = decideRequest(&plan.Turn, chapters[len(chapters)-1].Title, recent, question)

	return plan, nil
}

func (s *Service) planAnswered(
	ctx context.Context,
	plan *assistantports.ChapterPlan,
	turn *conversation.AssistantTurn,
	existing *conversation.Chapter,
	auto bool,
) (*assistantports.ChapterPlan, error) {
	if existing == nil {
		plan.Skip = skipNoChapter

		return plan, nil
	}
	plan.ChapterID = existing.ID

	if existing.StartSequence == nil ||
		turn.Status == conversation.AssistantTurnStatusRefused {
		plan.Remove = existing.TitleSource != conversation.ChapterTitlePerson
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
	plan *assistantports.ChapterPlan,
	existing *conversation.Chapter,
	auto bool,
) (*assistantports.ChapterPlan, error) {
	switch {
	case existing == nil:
		plan.Skip = skipNoChapter
	case !auto || existing.TitleSource != conversation.ChapterTitleQuestion:
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
	plan *assistantports.ChapterPlan,
	existing *conversation.Chapter,
) (*assistantports.ChapterPlan, error) {
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

	plan.ModelSource = conversation.ChapterTitleModel
	plan.Evidence = []string{question, reply}
	plan.Request = nameRequest(&plan.Turn, question, reply)

	return plan, nil
}

func worthRenaming(chapter *conversation.Chapter, question string) bool {
	return chapter.TitleSource == conversation.ChapterTitleModel &&
		len(strings.Fields(question)) < shortQuestionWord
}

func earlierToName(chapters []*conversation.Chapter) pulid.ID {
	if len(chapters) == 0 {
		return ""
	}
	if last := chapters[len(chapters)-1]; last.TitleSource == conversation.ChapterTitleQuestion {
		return last.TurnID
	}

	return ""
}

func (s *Service) Apply(
	ctx context.Context,
	plan *assistantports.ChapterPlan,
	result *llmports.StructuredCompletionResult,
) (*assistantports.ChapterOutcome, error) {
	outcome := &assistantports.ChapterOutcome{ChapterID: plan.ChapterID}
	if plan.Skipped() {
		return outcome, nil
	}
	turn := &plan.Turn

	if plan.Remove {
		removed, err := s.chapters.Delete(ctx, assistantports.DeleteChapterRequest{
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
		if _, err := s.chapters.Retitle(ctx, assistantports.RetitleChapterRequest{
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
	plan *assistantports.ChapterPlan,
	title string,
	source conversation.ChapterTitleSource,
	outcome *assistantports.ChapterOutcome,
) (*assistantports.ChapterOutcome, error) {
	turn := &plan.Turn
	chapter, err := s.create(ctx, &conversation.Chapter{
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
	plan *assistantports.ChapterPlan,
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

	answer.Title = conversation.CleanChapterTitle(answer.Title)
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
