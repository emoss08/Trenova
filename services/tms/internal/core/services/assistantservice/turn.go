package assistantservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// TurnPlan is a question made ready to answer: checked, guarded, and turned
// into a runtime turn, all as data.
//
// A turn is answered in three steps that may run on different workers:
// PrepareTurn reads what the question needs and builds this, the runtime
// drives the turn, and FinishTurn saves what it came to. Everything the last
// step needs from the first travels here, so nothing is read twice and
// nothing read in one step can disagree with what another step saw.
type TurnPlan struct {
	ThreadID   pulid.ID                    `json:"threadId"`
	Definition *agentdefinition.Definition `json:"definition"`
	// Input is what the model answers: the question, or on a decision
	// follow-up the note describing the decision.
	Input    string              `json:"input"`
	FollowUp bool                `json:"followUp"`
	Decision agentguard.Decision `json:"decision"`
	// Page, Attachments and Mentions are what the person handed over with
	// the question. They are kept on the question when it is saved.
	Page        *agent.PageContext               `json:"page,omitempty"`
	Attachments []conversation.MessageAttachment `json:"attachments,omitempty"`
	Mentions    []agent.EntityRef                `json:"mentions,omitempty"`
	// Timezone is the organization's, which is what a tool reads a bare date
	// in.
	Timezone string `json:"timezone,omitempty"`
	// PreferredProviderID is the model the conversation asked for.
	PreferredProviderID pulid.ID                   `json:"preferredProviderId,omitzero"`
	Proposals           []services.ProposalOutcome `json:"proposals,omitempty"`
	// Turn is the runtime turn, empty when the guard refused the question.
	Turn agentruntime.TurnState `json:"turn"`
}

// Refused reports a question the guard declined. There is nothing to run.
func (p *TurnPlan) Refused() bool { return !p.Decision.Allowed }

// Opening is the first thing the reader is told: that the question was taken,
// or why it was not.
func (p *TurnPlan) Opening() services.StreamEvent {
	return openingEvent(p.Input, p.Decision)
}

func openingEvent(input string, decision agentguard.Decision) services.StreamEvent {
	if !decision.Allowed {
		return refusedEvent(decision)
	}

	return services.StreamEvent{
		Event: services.AssistantEventAccepted,
		Data: services.AssistantAcceptedEvent{
			Content:       input,
			ScopeStage:    string(decision.Stage),
			ScopeCategory: string(decision.Category),
		},
	}
}

// RunRequest is the plan as the runtime reads it, less the services a caller
// supplies for itself.
func (p *TurnPlan) RunRequest(actor *services.RequestActor) *services.RunRequest {
	return &services.RunRequest{
		Definition:          p.Definition,
		Actor:               actor,
		Context:             agentdefinition.RuntimeContext{Timezone: p.Timezone},
		Input:               p.Input,
		PreferredProviderID: p.PreferredProviderID,
		// A model the person picked is the model they get; an administrator's
		// default on the agent is only where the order starts.
		PinProvider: !p.PreferredProviderID.IsNil(),
		ThreadID:    p.ThreadID,
		Proposals:   p.Proposals,
	}
}

func (p *TurnPlan) turnContext() turnContext {
	return turnContext{page: p.Page, attachments: p.Attachments, mentions: p.Mentions}
}

// PrepareTurn checks a question and makes it ready to answer.
//
// It reads everything the turn needs and decides whether it may run at all,
// which is every refusal the person can do something about: an empty message,
// a disabled agent, a spent budget, a full conversation, a file that is not
// theirs. The scope guard runs here too, so a question the assistant will not
// answer never reaches the model.
func (s *Service) PrepareTurn(
	ctx context.Context,
	req *services.SendMessageRequest,
	actor *services.RequestActor,
) (*TurnPlan, error) {
	plan, _, err := s.prepareTurn(ctx, req, actor)

	return plan, err
}

// prepareTurn is PrepareTurn, also handing back the run request the turn was
// opened from, for a caller that drives the turn in process.
func (s *Service) prepareTurn(
	ctx context.Context,
	req *services.SendMessageRequest,
	actor *services.RequestActor,
) (*TurnPlan, *services.RunRequest, error) {
	content := strings.TrimSpace(req.Content)
	followUp := req.FollowUpProposalID.IsNotNil() || req.FollowUpPlanID.IsNotNil()
	multiErr := errortypes.NewMultiError()
	switch {
	case content == "" && !followUp:
		multiErr.Add("content", errortypes.ErrRequired, "Message cannot be empty")
	case content != "" && followUp:
		multiErr.Add("content", errortypes.ErrInvalid,
			"A decision follow-up carries no message; the decision is its input")
	case req.FollowUpProposalID.IsNotNil() && req.FollowUpPlanID.IsNotNil():
		multiErr.Add("followUpPlanId", errortypes.ErrInvalid,
			"A follow-up answers one decision: a proposal or a plan, not both")
	}
	page := req.Page.Normalized()
	if page != nil {
		page.Validate("context", multiErr)
	}
	mentions := validateMentions(req.Mentions, multiErr)
	if multiErr.HasErrors() {
		return nil, nil, multiErr
	}

	thread, err := s.conversations.GetThread(ctx, repositories.GetThreadRequest{
		ID:         req.ThreadID,
		UserID:     actor.UserID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}

	attachments, runtimeAttachments, err := s.resolveAttachments(
		ctx, thread, req.AttachmentDocumentIDs, actor, req.TenantInfo,
	)
	if err != nil {
		return nil, nil, err
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         thread.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}

	if !definition.Enabled {
		return nil, nil, errortypes.NewBusinessError(
			"Agent {0} is disabled and cannot be used", definition.Name,
		)
	}
	if err = assertChatAgent(definition); err != nil {
		return nil, nil, err
	}

	if err = s.assertWithinBudget(ctx, definition); err != nil {
		return nil, nil, err
	}

	if err = s.assertRoom(ctx, thread, req.TenantInfo); err != nil {
		return nil, nil, err
	}

	// Another agent's steps on a task this one handed it are the thread's to
	// show, not the model's to read again: it only ever saw its own call and
	// the answer that came back.
	history, err := s.conversations.ListMessages(ctx, repositories.ListMessagesRequest{
		ThreadID:     thread.ID,
		TenantInfo:   req.TenantInfo,
		Limit:        historyLimit,
		ExcludeKinds: conversation.ModelHiddenKinds(),
	})
	if err != nil {
		return nil, nil, err
	}

	if followUp {
		content, err = s.decisionNote(ctx, decisionNoteParams{
			thread:     thread,
			proposalID: req.FollowUpProposalID,
			planID:     req.FollowUpPlanID,
			history:    history,
			tenant:     req.TenantInfo,
		})
		if err != nil {
			return nil, nil, err
		}
	}

	s.keepProviderChoice(ctx, thread, req, actor)

	turnReq := &TurnRequest{
		Definition:          definition,
		Actor:               actor,
		History:             history,
		Input:               content,
		Page:                page,
		PreferredProviderID: thread.PreferredProviderID,
		ThreadID:            thread.ID,
		Proposals:           s.proposalOutcomes(ctx, thread, req.TenantInfo),
		Subject:             s.describeSubject(ctx, thread, actor, req.TenantInfo),
		Attachments:         runtimeAttachments,
		Mentions:            mentions,
	}

	decision, runReq := s.admit(ctx, turnReq)
	plan := &TurnPlan{
		ThreadID:            thread.ID,
		Definition:          definition,
		Input:               content,
		FollowUp:            followUp,
		Decision:            decision,
		Page:                page,
		Attachments:         attachments,
		Mentions:            mentions,
		PreferredProviderID: thread.PreferredProviderID,
		Proposals:           turnReq.Proposals,
	}
	if runReq != nil {
		// A conversation keeps what the turn publishes beside it, in the
		// activity that runs the publish rather than on this request.
		runReq.Publishes = s.artifacts != nil
		plan.Timezone = runReq.Context.Timezone
		plan.Turn = s.runtime.OpenTurn(ctx, runReq).State()
	}

	return plan, runReq, nil
}

// keepProviderChoice resolves the reader's choice of model against what they
// may actually pick, then keeps it on the thread so the picker still shows it
// after a reload. A choice that no longer resolves is cleared rather than
// carried, so the stored preference and the model that answers cannot drift
// apart. A send that says nothing about the model keeps the thread's saved
// choice; only a choice, including the choice of "automatic", replaces it.
func (s *Service) keepProviderChoice(
	ctx context.Context,
	thread *conversation.Thread,
	req *services.SendMessageRequest,
	actor *services.RequestActor,
) {
	if !req.ProviderChosen {
		return
	}

	chosen := s.resolvePreference(ctx, req.PreferredProviderID, *actor)
	if chosen == thread.PreferredProviderID {
		return
	}

	thread.PreferredProviderID = chosen
	if _, err := s.conversations.UpdateThread(ctx, thread); err != nil {
		s.logger.Warn("could not save the chosen model on the thread",
			zap.String("thread", thread.ID.String()),
			zap.Error(err),
		)
	}
}

// FinishTurnRequest is what a turn came to, for saving.
type FinishTurnRequest struct {
	Plan       *TurnPlan
	Actor      *services.RequestActor
	TenantInfo pagination.TenantInfo
	// Run is what the runtime did. It is nil for a refused question, and may
	// be nil for a turn that failed before the runtime handed anything back.
	Run *services.RunResult
	// Failure is why the turn ended before it finished, nil when it did
	// finish. A stop is context.Canceled.
	Failure error
	// Artifacts are what the turn's tool calls produced, already kept.
	Artifacts []*assistantartifact.Artifact
}

// FinishTurn saves what a turn came to: the messages, any write it proposed,
// and the ties between the artifacts it made and the messages that made them.
//
// A turn that failed is saved too, closed with a note. A tool that executed
// before the failure changed something, and the conversation is the only place
// a person can see that it did.
//
// It runs whether or not anybody is still watching. What it announces goes to
// emit, which may be nil.
func (s *Service) FinishTurn(
	ctx context.Context,
	req *FinishTurnRequest,
	emit services.AssistantStreamEmitter,
) (*services.SendMessageResult, error) {
	plan := req.Plan

	thread, err := s.conversations.GetThread(ctx, repositories.GetThreadRequest{
		ID:         plan.ThreadID,
		UserID:     req.Actor.UserID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	artifacts := s.newArtifactRecorder(ctx, thread, req.TenantInfo, req.Actor, emit)
	artifacts.adopt(req.Artifacts)

	turn := turnResultOf(plan.Input, plan.Decision, req.Run, req.Failure)
	attachTurnContext(turn.Messages, plan.turnContext())
	if plan.FollowUp {
		markDecisionNote(turn.Messages)
	}

	saved, err := s.conversations.AppendTurn(ctx, repositories.AppendTurnRequest{
		ThreadID:   thread.ID,
		TenantInfo: req.TenantInfo,
		Messages:   turn.Messages,
	})
	if err != nil {
		return nil, err
	}
	artifacts.attachMessages(sourceMessageIndex(saved))
	s.runtime.MarkToolEffects(saved)
	s.nameDelegatedSteps(ctx, req.TenantInfo, saved)

	if req.Failure == nil && !plan.FollowUp {
		s.titleIfUnnamed(ctx, thread, plan.Input)
	}

	result := &services.SendMessageResult{
		Thread:   thread,
		Messages: saved,
		Reply:    turn.Reply,
		Refused:  !turn.Decision.Allowed,
	}

	// A turn that failed or was stopped is recorded too. A write it made
	// before it ended happened, and without its proposal it would leave no
	// audit row, count against no cap, and earn no trust.
	own := persistProposalsParams{
		Failed:     req.Failure != nil,
		Definition: plan.Definition,
		Thread:     thread,
		Actor:      req.Actor,
		Saved:      saved,
		Actions:    turn.Actions,
		Model:      turn.Model,
		Input:      plan.Input,
		Artifacts:  artifacts,
	}
	proposals, err := s.persistProposals(ctx, own)
	if req.Run != nil && len(req.Run.Delegations) > 0 {
		delegated, delegatedErr := s.persistDelegatedProposals(ctx, own, req.Run.Delegations)
		proposals = append(proposals, delegated...)
		err = errors.Join(err, delegatedErr)
	}
	if err != nil {
		s.logProposalPersistFailure(thread, err)
	}

	result.Proposals = proposals
	result.ProposalsUnrecorded = err != nil
	result.Artifacts = artifacts.artifacts()

	return result, nil
}

// ObserveTool keeps what a finished tool call produced for a person to see
// beside the conversation, and says so on emit. It runs where the tool ran,
// the only place its raw result exists. It answers with what the person now
// sees, for the model to be told, and with what it kept, so the turn can tie
// each artifact to its message once the turn is saved.
func (s *Service) ObserveTool(
	ctx context.Context,
	threadID pulid.ID,
	actor *services.RequestActor,
	observation services.ToolObservation,
	emit services.AssistantStreamEmitter,
) (*services.ShownArtifact, []*assistantartifact.Artifact, error) {
	thread := &conversation.Thread{ID: threadID}
	recorder := s.newArtifactRecorder(ctx, thread, actor.TenantInfo(), actor, emit)
	if recorder == nil {
		return nil, nil, nil
	}

	shown, err := recorder.observe(observation)

	return shown, recorder.recorded, err
}
