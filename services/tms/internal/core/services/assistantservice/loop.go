package assistantservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

// TurnRequest is one question as the guard and the runtime read it.
type TurnRequest struct {
	Definition *agentdefinition.Definition
	Actor      *serviceports.RequestActor
	History    []conversation.Message
	Input      string
	Page       *agentdefinition.PageContext
	// PreferredProviderID is the reader's chosen model for this conversation.
	PreferredProviderID pulid.ID
	// ThreadID is the conversation, for attributing what the turn cost.
	ThreadID pulid.ID
	// Proposals is what became of the writes earlier turns proposed.
	Proposals []serviceports.ProposalOutcome
	// Subject is the record the conversation is about, when it was opened
	// from one.
	Subject *agentdefinition.RuntimeSubject
	// Attachments and Mentions are what the person handed over with the
	// message: files, and records named from the composer.
	Attachments []agentdefinition.RuntimeAttachment
	Mentions    []agentdefinition.RuntimeMention
}

type TurnResult struct {
	Reply    string
	Decision agentguard.Decision
	Messages []conversation.Message
	Actions  []serviceports.PendingAction
	Model    string
	Provider pulid.ID
}

// admit asks the scope guard about a question and, when it may be answered,
// builds the run request that answers it. A refused question gets no run
// request: the point of guarding first is that the expensive call never
// happens.
func (s *Service) admit(
	ctx context.Context,
	req *TurnRequest,
) (agentguard.Decision, *serviceports.RunRequest) {
	decision := s.guard.Evaluate(ctx, agentguard.EvaluateRequest{
		TenantInfo: req.Actor.TenantInfo(),
		Input:      req.Input,
		Recent:     recentTurns(req.History),
	})
	if !decision.Allowed {
		return decision, nil
	}

	runtimeContext := s.buildContext(ctx, req)
	runtimeContext.PendingProposals = pendingProposals(req.Proposals)

	return decision, &serviceports.RunRequest{
		Definition:          req.Definition,
		Actor:               req.Actor,
		Context:             runtimeContext,
		History:             req.History,
		Input:               req.Input,
		PreferredProviderID: req.PreferredProviderID,
		// A model the person picked is the model they get; an administrator's
		// default on the agent is only where the order starts.
		PinProvider: !req.PreferredProviderID.IsNil(),
		ThreadID:    req.ThreadID,
		Proposals:   req.Proposals,
	}
}

// turnResultOf is what a turn came to, in the shape the conversation keeps:
// the refusal, the interrupted turn closed with a note, or the answer.
func turnResultOf(
	input string,
	decision agentguard.Decision,
	run *serviceports.RunResult,
	err error,
) *TurnResult {
	if !decision.Allowed {
		return &TurnResult{
			Reply:    decision.Message,
			Decision: decision,
			Messages: []conversation.Message{
				scopedMessage(conversation.RoleUser, input, decision, true),
				scopedMessage(conversation.RoleAssistant, decision.Message, decision, true),
			},
		}
	}

	if err != nil {
		return interruptedTurn(input, decision, run, err)
	}

	result := &TurnResult{
		Reply:    run.Reply,
		Decision: decision,
		Messages: run.Messages,
		Actions:  run.Actions,
		Model:    run.Model,
		Provider: run.ProviderID,
	}

	if len(result.Messages) > 0 {
		asked := result.Messages[0].CreatedAt
		result.Messages[0] = scopedMessage(conversation.RoleUser, input, decision, false)
		result.Messages[0].CreatedAt = asked
	}

	if run.OutputRefused {
		result.Decision = outputDecision(run, result.Messages)
	}

	return result
}

func (s *Service) buildContext(
	ctx context.Context,
	req *TurnRequest,
) agentdefinition.RuntimeContext {
	bare := agentdefinition.RuntimeContext{
		Trigger:     agent.RunTriggerChat,
		Page:        req.Page,
		Subject:     req.Subject,
		Attachments: req.Attachments,
		Mentions:    req.Mentions,
	}
	if s.contexts == nil {
		return bare
	}

	runtimeContext, err := s.contexts.Build(ctx, &serviceports.RuntimeContextRequest{
		Definition:  req.Definition,
		Actor:       req.Actor,
		Trigger:     agent.RunTriggerChat,
		Subject:     req.Subject,
		Page:        req.Page,
		Attachments: req.Attachments,
		Mentions:    req.Mentions,
	})
	if err != nil {
		s.logger.Warn("assistant context could not be built", zap.Error(err))

		return bare
	}

	return runtimeContext
}

// describeSubject reads the record a thread was opened from, so the model
// starts with it in front of it. A subject that cannot be read is named by
// its type and id rather than failing the turn: the conversation is still
// worth having.
//
// Access is checked on every turn, not only when the thread was opened: a
// person who lost access to the record keeps the conversation but no longer
// has the record read into it.
func (s *Service) describeSubject(
	ctx context.Context,
	thread *conversation.Thread,
	actor *serviceports.RequestActor,
	tenant pagination.TenantInfo,
) *agentdefinition.RuntimeSubject {
	if !thread.HasSubject() {
		return nil
	}

	bare := &agentdefinition.RuntimeSubject{
		Type:  thread.SubjectType,
		ID:    thread.SubjectID.String(),
		Label: stringutils.HumanizeSnakeCase(string(thread.SubjectType)),
	}
	if s.subjects == nil {
		return bare
	}
	allowed, err := s.mayReadSubject(ctx, actor, thread.SubjectType)
	if err != nil || !allowed {
		s.logger.Info("thread subject withheld from the turn",
			zap.String("thread", thread.ID.String()),
			zap.String("subjectType", string(thread.SubjectType)),
			zap.Bool("denied", err == nil),
			zap.Error(err),
		)

		return bare
	}

	subject, err := s.subjects.Describe(ctx, tenant, thread.SubjectType, thread.SubjectID)
	if err != nil {
		s.logger.Warn("thread subject could not be described",
			zap.String("thread", thread.ID.String()),
			zap.String("subjectType", string(thread.SubjectType)),
			zap.Error(err),
		)

		return bare
	}
	if subject == nil {
		return bare
	}

	return subject
}

// interruptedNotice closes a turn the model did not finish. It is written into
// the thread rather than left to an error banner because the thread is what is
// read back tomorrow, and a lookup followed by silence reads as an answer that
// was never given rather than one that was cut off.
const interruptedNotice = "_This reply was interrupted before it finished. " +
	"What is shown above is what had happened by then. Ask again to continue._"

// stoppedNotice is the same, for a turn the person ended themselves.
const stoppedNotice = "_Stopped here. What is shown above is what had happened by then._"

// The same two notes for a turn that ended before the model said or did
// anything. "What is shown above" under a question with nothing above it but
// the question read as a reply that had been lost.
const (
	failedBeforeStartNotice  = "_This reply failed before it started. Ask again to continue._"
	stoppedBeforeStartNotice = "_Stopped before a reply started. Ask again to continue._"
)

// closingNotice picks the note for how the turn ended and whether anything
// had happened by then.
func closingNotice(err error, ranAnything bool) string {
	stopped := errors.Is(err, context.Canceled)
	switch {
	case stopped && ranAnything:
		return stoppedNotice
	case stopped:
		return stoppedBeforeStartNotice
	case ranAnything:
		return withReason(interruptedNotice, err)
	default:
		return withReason(failedBeforeStartNotice, err)
	}
}

// withReason adds why the turn failed, when the failure is one the person
// can do something about. A request the provider refused is not fixed by
// asking again; a provider that could not be reached usually is. The notice
// is italic markdown, so the reason goes inside the closing underscore.
func withReason(notice string, err error) string {
	reason := failureReason(err)
	if reason == "" {
		return notice
	}

	return strings.TrimSuffix(notice, "_") + " " + reason + "_"
}

// failureReason names the kind of failure in a sentence, without the
// provider's own message: that message is for the administrator and is
// kept with the usage record in AI Control.
func failureReason(err error) string {
	var failure serviceports.ProviderFailure
	if errors.As(err, &failure) {
		if failure.ProviderRetryable() {
			return fmt.Sprintf(
				"The model provider was unavailable (status %d).", failure.ProviderStatus(),
			)
		}

		return fmt.Sprintf(
			"The model provider rejected the request (status %d); an administrator "+
				"can see its reason under AI Control.",
			failure.ProviderStatus(),
		)
	}
	if errors.Is(err, serviceports.ErrProvidersResting) {
		return "Every model provider is paused after repeated failures."
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "The model provider did not answer in time."
	}

	return ""
}

// interruptedTurn is what a failed run leaves behind: everything that ran,
// closed with a note. A run that came back with nothing, or did not come
// back at all, still leaves the question and the note, so the thread shows
// what the person asked and that nothing answered it.
//
// Keeping it is not optional. A tool that executed before the failure changed
// something, and the thread is the only place a person can see that it did;
// discarding the turn erased the write from the record while leaving it in the
// database.
func interruptedTurn(
	input string,
	decision agentguard.Decision,
	run *serviceports.RunResult,
	err error,
) *TurnResult {
	if run == nil {
		run = &serviceports.RunResult{}
	}

	// The first message is the question; anything after it is what ran.
	notice := closingNotice(err, len(run.Messages) > 1)

	messages := make([]conversation.Message, 0, len(run.Messages)+2)
	question := scopedMessage(conversation.RoleUser, input, decision, false)
	if len(run.Messages) > 0 {
		question.CreatedAt = run.Messages[0].CreatedAt
	}
	messages = append(messages, question)
	if len(run.Messages) > 1 {
		messages = append(messages, run.Messages[1:]...)
	}
	messages = append(messages, conversation.Message{
		Role:       conversation.RoleAssistant,
		Content:    notice,
		Model:      run.Model,
		ProviderID: run.ProviderID,
	})

	return &TurnResult{
		Reply:    notice,
		Decision: decision,
		Messages: messages,
		Actions:  run.Actions,
		Model:    run.Model,
		Provider: run.ProviderID,
	}
}

func scopedMessage(
	role conversation.Role,
	content string,
	decision agentguard.Decision,
	refused bool,
) conversation.Message {
	return conversation.Message{
		Role:          role,
		Content:       content,
		ScopeStage:    string(decision.Stage),
		ScopeCategory: string(decision.Category),
		ScopeReason:   string(decision.Reason),
		Refused:       refused,
	}
}

func outputDecision(
	run *serviceports.RunResult,
	messages []conversation.Message,
) agentguard.Decision {
	decision := agentguard.Decision{
		Allowed:     false,
		Stage:       agentguard.StageOutput,
		Message:     run.Reply,
		MatchedRule: run.OutputRule,
	}
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if message.Role == conversation.RoleAssistant && message.Refused {
			decision.Category = agentguard.Category(message.ScopeCategory)
			decision.Reason = agentguard.Reason(message.ScopeReason)

			break
		}
	}

	return decision
}

func refusedEvent(decision agentguard.Decision) serviceports.StreamEvent {
	return serviceports.StreamEvent{
		Event: serviceports.AssistantEventRefused,
		Data: serviceports.AssistantRefusedEvent{
			Message:  decision.Message,
			Stage:    string(decision.Stage),
			Category: string(decision.Category),
			Reason:   string(decision.Reason),
		},
	}
}

// recentTurns renders the thread's prose for the scope guard.
//
// Only what was said: a tool call and its result are a payload and a machine
// answer, and whatever they established about the subject is already in the
// reply the assistant wrote from them. Including them would spend the
// classifier's context on JSON and tell it less.
func recentTurns(history []conversation.Message) []agentguard.Turn {
	turns := make([]agentguard.Turn, 0, len(history))
	for _, message := range history {
		switch message.Role {
		case conversation.RoleUser:
			turns = append(turns, agentguard.Turn{Role: "user", Content: message.Content})
		case conversation.RoleAssistant:
			turns = append(turns, agentguard.Turn{Role: "assistant", Content: message.Content})
		case conversation.RoleTool:
		}
	}

	return turns
}

// pendingProposals names the undecided proposals for the prompt.
func pendingProposals(outcomes []serviceports.ProposalOutcome) []agentdefinition.PendingProposal {
	pending := make([]agentdefinition.PendingProposal, 0, len(outcomes))
	for _, outcome := range outcomes {
		if outcome.Pending() {
			pending = append(pending, agentdefinition.PendingProposal{
				ToolName:  outcome.ToolName,
				Rationale: outcome.Rationale,
			})
		}
	}

	return pending
}
