package assistantservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// DelegateDeclinedError is an agent that may not be handed a task, with why,
// in words the agent that asked can pass on to the person.
type DelegateDeclinedError struct {
	Reason string
}

func (e *DelegateDeclinedError) Error() string { return e.Reason }

func declined(reason string) error { return &DelegateDeclinedError{Reason: reason} }

// OpenDelegateRequest is a task the agent a person is talking to hands to
// another agent, from inside that person's turn.
type OpenDelegateRequest struct {
	// Parent is the agent that handed the task over, as its turn began.
	Parent *agentdefinition.Definition
	// Actor is the person the turn runs as. The delegate runs as them too.
	Actor     *services.RequestActor
	ThreadID  pulid.ID
	StepOwner services.RunStepOwner
	Call      agentruntime.DelegateCall
}

// DelegateOpening is the delegate's turn, ready to drive.
type DelegateOpening struct {
	Request *services.RunRequest
	Turn    agentruntime.TurnState
}

// OpenDelegate checks that the task may be handed to the agent it names and
// opens that agent's turn on it.
//
// Every check is made again here, against the records as they are now rather
// than as the turn began: the parent must still list the delegate, the
// delegate must still exist in the tenant, be enabled and be an agent people
// talk to, the person must still be allowed to use the assistant, and the
// delegate's own budget must not be spent. A refusal is a
// *DelegateDeclinedError.
//
// The turn is opened as the delegate: its tools narrowed to what the person
// may use, its tiers, its ceiling, its budget, its model. It is marked as
// working for the parent, so it never holds delegate_task, and it is not
// unattended, so a write that is the person's own still runs as theirs.
func (s *Service) OpenDelegate(
	ctx context.Context,
	req *OpenDelegateRequest,
) (*DelegateOpening, error) {
	if req.Parent == nil || req.Actor == nil {
		return nil, declined("There is no conversation to hand this task from.")
	}
	tenant := req.Actor.TenantInfo()
	name := req.Call.Delegate.Name

	parent, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.Parent.ID,
		TenantInfo: tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, declined("The agent handing this task over no longer exists.")
		}

		return nil, fmt.Errorf("read the agent handing over the task: %w", err)
	}

	delegate, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.Call.Delegate.ID,
		TenantInfo: tenant,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, declined(name + " no longer exists.")
		}

		return nil, fmt.Errorf("read %s: %w", name, err)
	}

	if refusal := parent.DelegateRefusal(delegate); refusal != "" {
		return nil, declined(refusal)
	}
	if !agentruntime.MayUseAssistant(ctx, s.permissions, req.Actor, s.logger) {
		return nil, declined("The person is not allowed to use other agents, so " +
			delegate.Name + " cannot be asked.")
	}
	if err = s.assertWithinBudget(ctx, delegate); err != nil {
		if errortypes.IsBusinessError(err) {
			return nil, declined(err.Error())
		}

		return nil, err
	}

	runReq := &services.RunRequest{
		Definition: delegate,
		Actor:      req.Actor,
		Context:    s.delegateContext(ctx, parent, delegate, req.Actor),
		Input:      req.Call.Task,
		ThreadID:   req.ThreadID,
		// Earlier proposals keep the delegate from proposing again a write
		// that is already waiting on the person.
		Proposals: s.proposalOutcomes(ctx, &conversation.Thread{ID: req.ThreadID}, tenant),
		// A document it publishes is kept beside the same conversation.
		Publishes: s.artifacts != nil,
		StepOwner: req.StepOwner,
		Delegation: &services.Delegation{
			ParentAgentID:   parent.ID,
			ParentAgentName: parent.Name,
			CallID:          req.Call.Call.ID,
			StepScope:       req.Call.StepScope,
		},
	}

	return &DelegateOpening{
		Request: runReq,
		Turn:    s.runtime.OpenTurn(ctx, runReq).State(),
	}, nil
}

// delegateContext is the delegate's prompt context: its own, built as for a
// conversation, and marked as working for the parent. A context that cannot
// be built leaves the delegate with less to go on, not without its task.
func (s *Service) delegateContext(
	ctx context.Context,
	parent, delegate *agentdefinition.Definition,
	actor *services.RequestActor,
) agentdefinition.RuntimeContext {
	bare := agentdefinition.RuntimeContext{
		Trigger:     agent.RunTriggerChat,
		DelegatedBy: parent.Name,
	}
	if s.contexts == nil {
		return bare
	}

	built, err := s.contexts.Build(ctx, &services.RuntimeContextRequest{
		Definition:  delegate,
		Actor:       actor,
		Trigger:     agent.RunTriggerChat,
		DelegatedBy: parent.Name,
	})
	if err != nil {
		s.logger.Warn("the context of an agent handed a task could not be built",
			zap.String("agent", delegate.ID.String()),
			zap.Error(err),
		)

		return bare
	}

	return built
}

// IsDelegateDeclined reports a task an agent may not be handed.
func IsDelegateDeclined(err error) (*DelegateDeclinedError, bool) {
	var refusal *DelegateDeclinedError
	if errors.As(err, &refusal) {
		return refusal, true
	}

	return nil, false
}

// persistDelegatedProposals records the writes each agent the turn handed a
// task to proposed or made, as that agent's own: a run of its definition in
// this conversation, so a decision on one is a decision on it, and its trust
// is what the decision teaches. The follow-up after a decision still comes to
// this conversation, whose agent is the one the person is talking to.
func (s *Service) persistDelegatedProposals(
	ctx context.Context,
	params persistProposalsParams,
	delegations []services.DelegatedRun,
) ([]services.AssistantProposal, error) {
	proposals := make([]services.AssistantProposal, 0, len(delegations))
	var failures []error
	for idx := range delegations {
		delegation := &delegations[idx]
		if delegation.Definition == nil || len(delegation.Actions) == 0 {
			continue
		}

		recorded, err := s.persistProposals(ctx, persistProposalsParams{
			Failed:     params.Failed || delegation.Failed,
			Definition: delegation.Definition,
			Thread:     params.Thread,
			Actor:      params.Actor,
			Saved:      params.Saved,
			Actions:    delegation.Actions,
			Model:      delegation.Model,
			Input:      delegation.Input,
			Artifacts:  params.Artifacts,
		})
		if err != nil {
			failures = append(failures, fmt.Errorf("record %s's proposals: %w",
				delegation.Definition.Name, err))
			continue
		}
		proposals = append(proposals, recorded...)
	}

	return proposals, errors.Join(failures...)
}
