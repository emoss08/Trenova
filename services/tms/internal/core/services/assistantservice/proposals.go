package assistantservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// chatRunOpener and chatProposalStore are the slices of the agent run and
// proposal repositories a conversation actually needs.
//
// A chat turn opens a run and records proposals against it. It has no business
// advancing a run's status or resolving a proposal — those belong to the
// decision service — so the dependency says only what this service may do.
type chatRunOpener interface {
	Create(ctx context.Context, entity *agent.AgentRun) (*agent.AgentRun, error)
}

type chatProposalStore interface {
	Create(ctx context.Context, entity *agent.AgentProposal) (*agent.AgentProposal, error)
	ListByThread(
		ctx context.Context,
		req repositories.ListAgentProposalsByThreadRequest,
	) ([]*agent.AgentProposal, error)
}

// Evidence on a chat proposal points at the conversation, because that is where
// the reasoning is. An approver who wants more than the rationale line can open
// the thread and read the exchange that led to it, including the tool results the
// model saw.
const (
	evidenceTypeThread  = "AssistantThread"
	evidenceTypeMessage = "AssistantMessage"
)

// chatPromptVersion identifies the prompt shape a chat run used. Agent runs
// require one so a later change in how agents are prompted can be told apart
// from a change in the model.
const chatPromptVersion = "assistant-chat/v1"

// persistProposalsParams groups what turning a turn's pending actions into
// durable proposals needs.
type persistProposalsParams struct {
	Definition *agentdefinition.Definition
	Thread     *conversation.Thread
	Actor      *services.RequestActor
	// Saved are the messages as persisted, used to tie each proposal to the
	// assistant turn that asked for it.
	Saved   []conversation.Message
	Actions []PendingAction
	Model   string
	Input   string
}

// persistProposals records the turn's proposed writes so they can be approved.
//
// Before this, a proposal lived only in the HTTP response: refreshing the page
// lost it, and there was nothing for the approval endpoint to act on. Each turn
// that proposes something opens one agent run, which is what the existing
// proposal, decision and audit machinery is built around — a chat turn is just
// another thing an agent did, and it belongs in the same ledger as the billing
// agent's work rather than in a parallel one.
func (s *Service) persistProposals(
	ctx context.Context,
	params persistProposalsParams,
) ([]services.AssistantProposal, error) {
	if len(params.Actions) == 0 {
		return nil, nil
	}

	run, err := s.openChatRun(ctx, params)
	if err != nil {
		return nil, err
	}

	sourceByToolCall := sourceMessageIndex(params.Saved)
	persisted := make([]services.AssistantProposal, 0, len(params.Actions))

	for _, action := range params.Actions {
		proposal := &agent.AgentProposal{
			OrganizationID:  params.Actor.OrganizationID,
			BusinessUnitID:  params.Actor.BusinessUnitID,
			RunID:           run.ID,
			ToolName:        action.ToolName,
			ToolParams:      nonNilParams(action.Arguments),
			Rationale:       action.Rationale,
			AutonomyTier:    proposalTier(action.Tier),
			Status:          agent.ProposalStatusPending,
			SourceMessageID: sourceByToolCall[action.ToolCallID],
			Evidence:        chatEvidence(params.Thread, sourceByToolCall[action.ToolCallID]),
		}

		multiErr := errortypes.NewMultiError()
		proposal.Validate(multiErr)
		if multiErr.HasErrors() {
			return nil, multiErr
		}

		created, cErr := s.proposals.Create(ctx, proposal)
		if cErr != nil {
			return nil, cErr
		}

		persisted = append(persisted, toAssistantProposal(created))
	}

	return persisted, nil
}

// openChatRun creates the run a conversation turn's proposals hang from.
//
// The run is completed immediately rather than left open: unlike the billing
// agent, a chat turn has already finished thinking by the time its proposals
// exist, and the proposals carry their own pending status. Leaving the run
// awaiting a decision would make every answered thread look like unfinished
// work on the agent runs screen.
func (s *Service) openChatRun(
	ctx context.Context,
	params persistProposalsParams,
) (*agent.AgentRun, error) {
	now := timeutils.NowUnix()
	run := &agent.AgentRun{
		OrganizationID:   params.Actor.OrganizationID,
		BusinessUnitID:   params.Actor.BusinessUnitID,
		AgentType:        agent.TypeAssistantChat,
		SubjectType:      agent.SubjectAssistantThread,
		SubjectID:        params.Thread.ID,
		Status:           agent.RunStatusAwaitingDecision,
		ModelIdentifier:  params.Model,
		PromptVersion:    chatPromptVersion,
		InputContextHash: hashChatContext(params.Definition, params.Input),
		StartedAt:        now,
	}

	multiErr := errortypes.NewMultiError()
	run.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.runs.Create(ctx, run)
}

// hashChatContext fingerprints what the model was asked, so two runs can be
// compared without storing the message again next to the conversation that
// already holds it.
func hashChatContext(definition *agentdefinition.Definition, input string) string {
	sum := sha256.Sum256(
		fmt.Appendf(nil, "%s\x00%d\x00%s", definition.ID.String(), definition.Version, input),
	)

	return hex.EncodeToString(sum[:])
}

// sourceMessageIndex maps each tool call id to the saved assistant message that
// asked for it. A turn can produce several assistant messages across loop
// iterations, so the call id rather than position is what ties them together.
func sourceMessageIndex(saved []conversation.Message) map[string]pulid.ID {
	index := make(map[string]pulid.ID, len(saved))
	for _, message := range saved {
		if message.Role != conversation.RoleAssistant {
			continue
		}
		for _, call := range message.ToolCalls {
			index[call.ID] = message.ID
		}
	}

	return index
}

// proposalTier fills in a tier the dispatcher did not set.
//
// The fallback is the most restrictive tier there is, so a missing value can only
// ever ask for more human involvement rather than less. Dropping the proposal
// instead would lose the one record that tells an approver something was asked
// for, and defaulting the other way would be a way for an unset field to grant
// autonomy nobody configured.
func proposalTier(tier agent.AutonomyTier) agent.AutonomyTier {
	if tier == "" {
		return agent.TierPropose
	}

	return tier
}

// chatEvidence cites the conversation a proposal came out of.
//
// The thread is always known. The message is cited only when the tool call could
// be matched to a saved turn: pointing at an id that is not there would be worse
// than citing one thing accurately.
func chatEvidence(thread *conversation.Thread, messageID pulid.ID) []agent.EvidenceRef {
	evidence := make([]agent.EvidenceRef, 0, 2)
	evidence = append(evidence, agent.EvidenceRef{
		Type: evidenceTypeThread,
		ID:   thread.ID.String(),
		Note: threadEvidenceNote(thread),
	})

	if !messageID.IsNil() {
		evidence = append(evidence, agent.EvidenceRef{
			Type: evidenceTypeMessage,
			ID:   messageID.String(),
			Note: "The assistant turn that asked for this action",
		})
	}

	return evidence
}

func threadEvidenceNote(thread *conversation.Thread) string {
	if title := strings.TrimSpace(thread.Title); title != "" {
		return fmt.Sprintf("Proposed during the conversation %q", title)
	}

	return "Proposed during an assistant conversation"
}

// nonNilParams keeps a proposal's parameters a JSON object rather than null. A
// tool taking no arguments is legitimate, and `null` would fail the column's
// not-null constraint for no reason.
func nonNilParams(params map[string]any) map[string]any {
	if params == nil {
		return map[string]any{}
	}

	return params
}

// logProposalPersistFailure reports proposals that could not be saved.
//
// The conversation itself is already persisted at this point, so the turn is not
// failed: the person got their answer, and the reply already told them the action
// has not run — which is still true. What they lose is the ability to approve it,
// so the result carries a flag and the client says so rather than showing an
// approval affordance for a proposal that does not exist.
func (s *Service) logProposalPersistFailure(thread *conversation.Thread, err error) {
	s.logger.Error("failed to persist assistant proposals",
		zap.String("thread", thread.ID.String()),
		zap.Error(err),
	)
}

// ListThreadProposals returns every proposal raised in a conversation, with the
// outcome of any that were approved.
//
// Reading the thread first is the authorization check, the same one that guards
// its messages: a thread is scoped to its owner, so someone else's thread is not
// found rather than returned.
func (s *Service) ListThreadProposals(
	ctx context.Context,
	req repositories.GetThreadRequest,
) ([]services.AssistantProposal, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}

	stored, err := s.proposals.ListByThread(
		ctx,
		repositories.ListAgentProposalsByThreadRequest{
			ThreadID:   req.ID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	proposals := make([]services.AssistantProposal, 0, len(stored))
	for _, proposal := range stored {
		proposals = append(proposals, toAssistantProposal(proposal))
	}

	return proposals, nil
}

func toAssistantProposal(proposal *agent.AgentProposal) services.AssistantProposal {
	return services.AssistantProposal{
		ID:              proposal.ID,
		RunID:           proposal.RunID,
		ToolName:        proposal.ToolName,
		Arguments:       proposal.ToolParams,
		Rationale:       proposal.Rationale,
		AutonomyTier:    proposal.AutonomyTier,
		Status:          proposal.Status,
		SourceMessageID: proposal.SourceMessageID,
		ExecutedAt:      proposal.ExecutedAt,
		ExecutionError:  proposal.ExecutionError,
	}
}
