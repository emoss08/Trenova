package assistantservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

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
}

type TurnResult struct {
	Reply    string
	Decision agentguard.Decision
	Messages []conversation.Message
	Actions  []serviceports.PendingAction
	Model    string
	Provider pulid.ID
}

func (s *Service) Run(ctx context.Context, req *TurnRequest) (*TurnResult, error) {
	return s.RunObserved(ctx, req, nil)
}

func (s *Service) RunObserved(
	ctx context.Context,
	req *TurnRequest,
	emit serviceports.AssistantStreamEmitter,
) (*TurnResult, error) {
	if emit == nil {
		emit = func(serviceports.StreamEvent) {}
	}

	decision := s.guard.Evaluate(ctx, agentguard.EvaluateRequest{
		TenantInfo: req.Actor.TenantInfo(),
		Input:      req.Input,
		Recent:     recentTurns(req.History),
	})
	if !decision.Allowed {
		emit(refusedEvent(decision))

		return &TurnResult{
			Reply:    decision.Message,
			Decision: decision,
			Messages: []conversation.Message{
				scopedMessage(conversation.RoleUser, req.Input, decision, true),
				scopedMessage(conversation.RoleAssistant, decision.Message, decision, true),
			},
		}, nil
	}

	emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventAccepted,
		Data: serviceports.AssistantAcceptedEvent{
			Content:       req.Input,
			ScopeStage:    string(decision.Stage),
			ScopeCategory: string(decision.Category),
		},
	})

	runtimeContext := s.buildContext(ctx, req)

	runtimeContext.PendingProposals = pendingProposals(req.Proposals)

	run, err := s.runtime.Run(ctx, &serviceports.RunRequest{
		Definition:          req.Definition,
		Actor:               req.Actor,
		Context:             runtimeContext,
		History:             req.History,
		Input:               req.Input,
		Emit:                emit,
		PreferredProviderID: req.PreferredProviderID,
		// A model the person picked is the model they get; an administrator's
		// default on the agent is only where the order starts.
		PinProvider: !req.PreferredProviderID.IsNil(),
		ThreadID:    req.ThreadID,
		Proposals:   req.Proposals,
	})
	if err != nil {
		return interruptedTurn(req, decision, run, err), err
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
		result.Messages[0] = scopedMessage(conversation.RoleUser, req.Input, decision, false)
	}

	if run.OutputRefused {
		result.Decision = outputDecision(run, result.Messages)
	}

	return result, nil
}

func (s *Service) buildContext(
	ctx context.Context,
	req *TurnRequest,
) agentdefinition.RuntimeContext {
	if s.contexts == nil {
		return agentdefinition.RuntimeContext{Trigger: agent.RunTriggerChat, Page: req.Page}
	}

	runtimeContext, err := s.contexts.Build(ctx, &serviceports.RuntimeContextRequest{
		Definition: req.Definition,
		Actor:      req.Actor,
		Trigger:    agent.RunTriggerChat,
		Page:       req.Page,
	})
	if err != nil {
		s.logger.Warn("assistant context could not be built", zap.Error(err))

		return agentdefinition.RuntimeContext{Trigger: agent.RunTriggerChat, Page: req.Page}
	}

	return runtimeContext
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
		return interruptedNotice
	default:
		return failedBeforeStartNotice
	}
}

// interruptedTurn is what a failed run leaves behind: everything that ran,
// closed with a note. Nil when nothing ran at all, since a turn with no user
// message in it is not a turn.
//
// Keeping it is not optional. A tool that executed before the failure changed
// something, and the thread is the only place a person can see that it did;
// discarding the turn erased the write from the record while leaving it in the
// database.
func interruptedTurn(
	req *TurnRequest,
	decision agentguard.Decision,
	run *serviceports.RunResult,
	err error,
) *TurnResult {
	if run == nil || len(run.Messages) == 0 {
		return nil
	}

	// The first message is the question; anything after it is what ran.
	notice := closingNotice(err, len(run.Messages) > 1)

	messages := make([]conversation.Message, 0, len(run.Messages)+1)
	messages = append(messages, run.Messages...)
	messages[0] = scopedMessage(conversation.RoleUser, req.Input, decision, false)
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

func outputDecision(run *serviceports.RunResult, messages []conversation.Message) agentguard.Decision {
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
