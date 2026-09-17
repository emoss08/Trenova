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
	"github.com/emoss08/trenova/internal/core/services/proposalrecorder"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type chatProposalStore interface {
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

const chatPromptVersion = "assistant-chat/v2"

// persistProposalsParams groups what turning a turn's pending actions into
// durable proposals needs.
type persistProposalsParams struct {
	Definition *agentdefinition.Definition
	Thread     *conversation.Thread
	Actor      *services.RequestActor
	// Saved are the messages as persisted, used to tie each proposal to the
	// assistant turn that asked for it.
	Saved   []conversation.Message
	Actions []services.PendingAction
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

	recorded, err := s.recorder.Record(ctx, &proposalrecorder.RecordRequest{
		Actor:      params.Actor,
		Definition: params.Definition,
		Open: &proposalrecorder.OpenRunRequest{
			AgentType:        agent.TypeAssistantChat,
			SubjectType:      agent.SubjectAssistantThread,
			SubjectID:        params.Thread.ID,
			Trigger:          agent.RunTriggerChat,
			Status:           agent.RunStatusCompleted,
			Model:            params.Model,
			PromptVersion:    chatPromptVersion,
			InputContextHash: hashChatContext(params.Definition, params.Input),
		},
		Actions:          params.Actions,
		SourceMessageIDs: sourceMessageIndex(params.Saved),
		Evidence: func(_ services.PendingAction, sourceMessageID pulid.ID) []agent.EvidenceRef {
			return chatEvidence(params.Thread, sourceMessageID)
		},
	})
	if err != nil {
		return nil, err
	}

	persisted := make([]services.AssistantProposal, 0, len(recorded.Proposals))
	for _, proposal := range recorded.Proposals {
		persisted = append(persisted, toAssistantProposal(proposal))
	}

	return persisted, nil
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
		Confidence:      proposal.Confidence.InexactFloat64(),
		ExecutedAt:      proposal.ExecutedAt,
		ExecutionError:  proposal.ExecutionError,
	}
}
