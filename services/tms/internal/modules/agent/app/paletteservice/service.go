package paletteservice

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/turnlimits"
	"github.com/emoss08/trenova/internal/modules/agent/app/paletteanswer"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/palettequestion"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/palettejobs/palettecontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// resultWait bounds reading a question's outcome: it has ended, or the
	// reader would not be asking.
	resultWait = 10 * time.Second
	streamPath = "/api/v1/assistant/ask/"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Answers       *paletteanswer.Service
	Assistant     assistantports.AssistantService
	Conversations conversationports.ConversationRepository
	Streams       assistantstreamports.TurnStreamReader
	Questions     paletteports.QuestionRepository
	Workflows     kernelports.WorkflowStarter
	Results       kernelports.WorkflowResults
	AgentControls tenantports.AgentControlRepository
	Quota         platformplanports.QuotaGuard
}

// Service runs command palette questions: each is a workflow on the chat
// queue that streams its answer, named by the person who asked so nobody else
// can read or stop it.
type Service struct {
	l             *zap.Logger
	answers       *paletteanswer.Service
	assistant     assistantports.AssistantService
	conversations conversationports.ConversationRepository
	streams       assistantstreamports.TurnStreamReader
	questions     paletteports.QuestionRepository
	workflows     kernelports.WorkflowStarter
	results       kernelports.WorkflowResults
	limits        *turnlimits.Limits
}

var _ paletteports.PaletteService = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) paletteports.PaletteService {
	return &Service{
		l:             p.Logger.Named("service.palette"),
		answers:       p.Answers,
		assistant:     p.Assistant,
		conversations: p.Conversations,
		streams:       p.Streams,
		questions:     p.Questions,
		workflows:     p.Workflows,
		results:       p.Results,
		limits: &turnlimits.Limits{
			Conversations:    p.Conversations,
			AgentControls:    p.AgentControls,
			Quota:            p.Quota,
			PaletteQuestions: p.Questions,
		},
	}
}

func (s *Service) Ask(
	ctx context.Context,
	req *paletteports.AskRequest,
	actor *permissionports.RequestActor,
) (*paletteports.Ask, error) {
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError("Only a person can ask Nova from the palette")
	}
	question, err := validQuestion(&req.Question)
	if err != nil {
		return nil, err
	}

	tenant := req.TenantInfo
	tenant.UserID = actor.UserID
	available, err := s.answers.Available(ctx, tenant)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, errortypes.NewBusinessError(
			"No AI provider answers command palette questions. " +
				"An administrator can assign one in AI Control.",
		)
	}

	if err = s.limits.AssertMayAsk(ctx, actor.UserID, tenant); err != nil {
		return nil, err
	}
	if err = s.assertRoom(ctx, tenant); err != nil {
		return nil, err
	}

	record := &palettequestion.Question{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		UserID:         tenant.UserID,
		Status:         palettequestion.StatusAsking,
	}
	if err = s.questions.Create(ctx, record); err != nil {
		return nil, err
	}

	ref := paletteports.AskRef{AskID: record.ID, TenantInfo: tenant}
	if _, err = s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:            palettecontract.WorkflowID(ref),
		TaskQueue:     temporaltype.TaskQueueAgentChat.String(),
		StaticSummary: "Nova answers a command palette question",
		Priority: temporal.Priority{
			PriorityKey: agentflowcontract.PriorityInteractive,
			FairnessKey: tenant.OrgID.String(),
		},
	}, palettecontract.AnswerWorkflowName, &palettecontract.AnswerInput{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		UserID:         tenant.UserID,
		AskID:          ref.AskID,
		Question:       *question,
	}); err != nil {
		s.finish(ctx, ref, palettequestion.StatusFailed)

		return nil, err
	}

	return &paletteports.Ask{
		ID:        ref.AskID,
		StreamURL: streamPath + ref.AskID.String() + "/stream/",
	}, nil
}

// assertRoom refuses a question while the person already has as many being
// answered as one person may: the palette shows one answer at a time.
func (s *Service) assertRoom(ctx context.Context, tenant pagination.TenantInfo) error {
	asking, err := s.questions.CountAsking(
		ctx,
		tenant,
		timeutils.NowUnix()-palettequestion.AskingWindow,
	)
	if err != nil {
		return err
	}
	if asking >= palettequestion.MaxAsking {
		return errortypes.NewBusinessError(
			"Nova is still answering your last question. Wait for it, or stop it first.",
		)
	}

	return nil
}

func (s *Service) finish(
	ctx context.Context,
	ref paletteports.AskRef,
	status palettequestion.Status,
) {
	if err := s.questions.Finish(ctx, ref, status, timeutils.NowUnix()); err != nil {
		s.l.Warn("could not record how a palette question ended",
			zap.String("askId", ref.AskID.String()), zap.Error(err))
	}
}

func validQuestion(question *paletteports.Question) (*paletteports.Question, error) {
	out := &paletteports.Question{
		Content:  strings.TrimSpace(question.Content),
		Page:     question.Page.Normalized(),
		Mentions: agentpagecontext.NormalizeEntityRefs(question.Mentions),
	}

	multiErr := errortypes.NewMultiError()
	if out.Content == "" {
		multiErr.Add("content", errortypes.ErrRequired, "Question cannot be empty")
	}
	if out.Page != nil {
		out.Page.Validate("context", multiErr)
	}
	agentpagecontext.ValidateEntityRefs("mentions", out.Mentions, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return out, nil
}

// Relay streams a question's events to its reader. A question whose workflow
// closed before the reader caught up ends with what it came to.
func (s *Service) Relay(ctx context.Context, req *paletteports.RelayRequest) error {
	ended := false
	err := s.streams.Read(ctx, &assistantstreamports.ReadTurnStreamRequest{
		Ref: assistantstreamports.TurnStreamRef{
			TenantInfo: req.Ref.TenantInfo,
			TurnID:     req.Ref.AskID,
			WorkflowID: palettecontract.WorkflowID(req.Ref),
		},
		Cursor: req.Cursor,
		OnFrame: func(frame assistantstreamports.TurnStreamFrame) error {
			ended = ended || frame.Terminal()

			return req.OnFrame(frame)
		},
	})
	if err != nil || ended || ctx.Err() != nil {
		return err
	}

	return req.OnFrame(s.ending(ctx, req.Ref))
}

func (s *Service) ending(
	ctx context.Context,
	ref paletteports.AskRef,
) assistantstreamports.TurnStreamFrame {
	answer, err := s.outcome(ctx, ref)
	if err == nil && answer.Failure == "" && answer.Answer != nil {
		data, _ := sonic.Marshal(paletteports.AnsweredEvent{AskID: ref.AskID})

		return assistantstreamports.TurnStreamFrame{
			Event: assistantstreamports.AssistantEventDone,
			Data:  data,
		}
	}

	message := "Nova couldn't answer that just now. Try again, or open it in the Desk."
	if err == nil && answer.Failure != "" {
		message = answer.Failure
	}
	data, _ := sonic.Marshal(map[string]string{"message": message})

	return assistantstreamports.TurnStreamFrame{
		Event: assistantstreamports.AssistantEventError,
		Data:  data,
	}
}

// Stop takes a question back. It is marked stopped at once, so asking the
// next one is not refused while the worker is still being told.
func (s *Service) Stop(ctx context.Context, ref paletteports.AskRef) error {
	if _, err := s.questions.Get(ctx, ref); err != nil {
		return err
	}
	s.finish(ctx, ref, palettequestion.StatusStopped)

	return s.workflows.CancelWorkflow(ctx, palettecontract.WorkflowID(ref), "")
}

func (s *Service) outcome(
	ctx context.Context,
	ref paletteports.AskRef,
) (*paletteports.Answer, error) {
	waitCtx, cancel := context.WithTimeout(ctx, resultWait)
	defer cancel()

	var answer paletteports.Answer
	if err := s.results.WorkflowResult(waitCtx, palettecontract.WorkflowID(ref), &answer); err != nil {
		return nil, errortypes.NewNotFoundError("That question is no longer available").
			WithInternal(err)
	}

	return &answer, nil
}

// Keep carries a question into the Desk: a conversation with the general
// assistant that opens with the question, the reads Nova made and its answer,
// so the person goes on from there. A question that was not answered opens
// the conversation with nothing in it, and the client asks it there. A
// question is carried once; asking again opens the same conversation.
func (s *Service) Keep(
	ctx context.Context,
	ref paletteports.AskRef,
	actor *permissionports.RequestActor,
) (*paletteports.Kept, error) {
	record, err := s.questions.Get(ctx, ref)
	if err != nil {
		return nil, err
	}
	if record.KeptThreadID.IsNotNil() {
		return s.kept(ctx, ref, record.KeptThreadID)
	}

	claimed, err := s.questions.ClaimKeep(ctx, ref, timeutils.NowUnix())
	if err != nil {
		return nil, err
	}
	if !claimed {
		if record, err = s.questions.Get(ctx, ref); err == nil && record.KeptThreadID.IsNotNil() {
			return s.kept(ctx, ref, record.KeptThreadID)
		}

		return nil, errortypes.NewBusinessError("This answer is already being opened in the Desk.")
	}

	kept, err := s.carry(ctx, ref, actor)
	if err != nil {
		if releaseErr := s.questions.ReleaseKeep(ctx, ref); releaseErr != nil {
			s.l.Warn("could not release a palette question after carrying it failed",
				zap.String("askId", ref.AskID.String()), zap.Error(releaseErr))
		}

		return nil, err
	}

	return kept, nil
}

func (s *Service) kept(
	ctx context.Context,
	ref paletteports.AskRef,
	threadID pulid.ID,
) (*paletteports.Kept, error) {
	thread, err := s.assistant.GetThread(ctx, &conversationports.GetThreadRequest{
		ID:         threadID,
		UserID:     ref.TenantInfo.UserID,
		TenantInfo: ref.TenantInfo,
	})
	if errortypes.IsNotFoundError(err) {
		return nil, errortypes.NewBusinessError(
			"This answer was opened in a Desk conversation that has since been deleted.",
		)
	}
	if err != nil {
		return nil, err
	}

	return &paletteports.Kept{Thread: thread}, nil
}

func (s *Service) carry(
	ctx context.Context,
	ref paletteports.AskRef,
	actor *permissionports.RequestActor,
) (*paletteports.Kept, error) {
	answer, err := s.outcome(ctx, ref)
	if err != nil {
		return nil, err
	}

	thread, err := s.assistant.StartGeneralThread(ctx, &assistantports.GeneralThreadRequest{
		Title:      answer.Question.Content,
		TenantInfo: ref.TenantInfo,
	}, actor)
	if err != nil {
		return nil, err
	}

	kept := &paletteports.Kept{Thread: thread}
	if answer.Answer == nil || answer.Failure != "" {
		kept.Unanswered = answer.Question.Content
	} else if _, err = s.conversations.AppendTurn(ctx, &conversationports.AppendTurnRequest{
		ThreadID:   thread.ID,
		TenantInfo: ref.TenantInfo,
		Messages:   seed(answer),
	}); err != nil {
		s.discard(ctx, ref, thread.ID)

		return nil, err
	}

	if err = s.questions.MarkKept(ctx, ref, thread.ID); err != nil {
		s.discard(ctx, ref, thread.ID)

		return nil, err
	}

	return kept, nil
}

// discard removes a conversation a failed carry left behind, so trying again
// does not leave two.
func (s *Service) discard(ctx context.Context, ref paletteports.AskRef, threadID pulid.ID) {
	if err := s.assistant.DeleteThread(ctx, &conversationports.GetThreadRequest{
		ID:         threadID,
		UserID:     ref.TenantInfo.UserID,
		TenantInfo: ref.TenantInfo,
	}); err != nil {
		s.l.Warn("could not remove a conversation a failed carry left behind",
			zap.String("threadId", threadID.String()), zap.Error(err))
	}
}

// seed is how a carried question opens its Desk conversation: the question
// and Nova's answer. The palette's own reads are not replayed, since the Desk
// holds none of those tools; the records they found ride on the question as
// mentions, which the Desk reads again with its own.
func seed(answer *paletteports.Answer) []conversation.Message {
	question := answer.Question

	return []conversation.Message{
		{
			Role:        conversation.RoleUser,
			Content:     question.Content,
			Mentions:    carriedMentions(question.Mentions, answer.Answer.Records),
			PageContext: question.Page,
		},
		{
			Role:       conversation.RoleAssistant,
			Content:    answer.Answer.Text,
			Model:      answer.Answer.Model,
			ProviderID: answer.Answer.ProviderID,
			Truncated:  answer.Answer.Truncated,
		},
	}
}

// carriedMentions are the records the question named, then the ones Nova
// read, each once and no more than a message may mention.
func carriedMentions(named, read []agentpagecontext.EntityRef) []agentpagecontext.EntityRef {
	refs := agentpagecontext.NormalizeEntityRefs(slices.Concat(named, read))
	seen := make(map[string]struct{}, len(refs))
	out := make([]agentpagecontext.EntityRef, 0, min(len(refs), agentpagecontext.MaxMentions))
	for _, ref := range refs {
		key := ref.Type + ":" + ref.ID
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
		if len(out) == agentpagecontext.MaxMentions {
			break
		}
	}

	return out
}
