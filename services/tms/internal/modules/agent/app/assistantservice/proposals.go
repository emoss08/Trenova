package assistantservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/app/proposalrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevidence"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentfingerprint"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantproposalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const chatPromptVersion = "assistant-chat/v2"

// persistProposalsParams groups what turning a turn's pending actions into
// durable proposals needs.
type persistProposalsParams struct {
	TurnID     pulid.ID
	Definition *agentdefinition.Definition
	Thread     *conversation.Thread
	Actor      *permissionports.RequestActor
	// Saved are the messages as persisted, used to tie each proposal to the
	// assistant turn that asked for it.
	Saved     []conversation.Message
	Actions   []agentruntimeports.PendingAction
	PlanTitle string
	Model     string
	Input     string
	// Failed records the run as having ended before the turn finished.
	Failed bool
	// Artifacts, when set, views each outbound message as a draft and the
	// plan as a checklist beside the conversation.
	Artifacts *artifactrecorder.Recorder
	// Taint is the outside content the turn that proposed read.
	Taint       *agent.RunTaint
	Fingerprint *agentfingerprint.Fingerprint
	Delegations []agentruntimeports.DelegatedRun
}

type chatRun struct {
	definition     *agentdefinition.Definition
	delegateCallID string
	failed         bool
	model          string
	input          string
	fingerprint    *agentfingerprint.Fingerprint
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
) ([]assistantproposalports.AssistantProposal, error) {
	delegated := make([]proposalrecorder.DelegatedActions, 0, len(params.Delegations))
	for idx := range params.Delegations {
		delegation := &params.Delegations[idx]
		if delegation.Definition == nil {
			continue
		}
		delegated = append(delegated, proposalrecorder.DelegatedActions{
			Definition: delegation.Definition,
			Open: params.openRun(chatRun{
				definition:     delegation.Definition,
				delegateCallID: delegation.CallID,
				failed:         params.Failed || delegation.Failed,
				model:          delegation.Model,
				input:          delegation.Input,
			}),
			Actions: delegation.Actions,
			Taint:   delegation.Taint,
		})
	}
	if len(params.Actions) == 0 && len(delegated) == 0 {
		return nil, nil
	}

	recorded, err := s.recorder.Record(ctx, &proposalrecorder.RecordRequest{
		Actor:      params.Actor,
		Definition: params.Definition,
		Open: params.openRun(chatRun{
			definition:  params.Definition,
			failed:      params.Failed,
			model:       params.Model,
			input:       params.Input,
			fingerprint: params.Fingerprint,
		}),
		Actions:          params.Actions,
		PlanTitle:        params.PlanTitle,
		Taint:            params.Taint,
		Delegated:        delegated,
		OpenEmptyRuns:    true,
		CallOrder:        callOrder(params.Saved),
		SourceMessageIDs: sourceMessageIndex(params.Saved),
		Evidence: func(_ agentruntimeports.PendingAction, sourceMessageID pulid.ID) []agentevidence.EvidenceRef {
			return chatEvidence(params.Thread, sourceMessageID)
		},
	})
	if err != nil {
		return nil, err
	}

	groups := make([]proposalGroup, 0, len(recorded.Delegated)+1)
	groups = append(groups, proposalGroup{
		definition: params.Definition,
		proposals:  recorded.Proposals,
	})
	for idx := range recorded.Delegated {
		groups = append(groups, proposalGroup{
			definition: recorded.Delegated[idx].Definition,
			proposals:  recorded.Delegated[idx].Proposals,
		})
	}

	all := make([]*agent.AgentProposal, 0, len(params.Actions))
	persisted := make([]assistantproposalports.AssistantProposal, 0, len(params.Actions))
	for _, group := range groups {
		if len(group.proposals) == 0 {
			continue
		}
		// The definition that just proposed is in hand, so the hold is
		// decided from it rather than read back through the run. A card that
		// appears with buttons and loses them on the next refresh would be
		// worse than one that arrives on hold.
		verdict, verdictErr := s.shadow.ForDefinition(ctx, tenantOf(params.Actor),
			group.definition)
		if verdictErr != nil {
			return nil, verdictErr
		}
		hold := verdict.Hold()
		for _, proposal := range group.proposals {
			out := proposalstores.ToAssistantProposal(proposal, hold)
			out.AgentID = group.definition.ID
			out.AgentName = group.definition.Name
			persisted = append(persisted, out)
		}
		all = append(all, group.proposals...)
	}

	params.Artifacts.FromProposals(all, recorded.Plan)

	return persisted, nil
}

type proposalGroup struct {
	definition *agentdefinition.Definition
	proposals  []*agent.AgentProposal
}

func (p *persistProposalsParams) openRun(run chatRun) *proposalrecorder.OpenRunRequest {
	status := agent.RunStatusCompleted
	if run.failed {
		status = agent.RunStatusFailed
	}

	return &proposalrecorder.OpenRunRequest{
		AgentType:        agent.TypeAssistantChat,
		SubjectType:      agent.SubjectAssistantThread,
		SubjectID:        p.Thread.ID,
		Trigger:          agent.RunTriggerChat,
		Status:           status,
		Model:            run.model,
		PromptVersion:    chatPromptVersion,
		InputContextHash: hashChatContext(run.definition, run.input),
		Fingerprint:      run.fingerprint,
		TraceID:          p.traceID(run.delegateCallID),
		TurnID:           p.TurnID,
		ParentOwnerKind:  p.parentOwnerKind(run.delegateCallID),
		ParentOwnerID:    p.parentOwnerID(run.delegateCallID),
		DelegateCallID:   p.delegateCallID(run.delegateCallID),
	}
}

func (p *persistProposalsParams) traceID(delegateCallID string) string {
	if p.TurnID.IsNil() {
		return ""
	}
	if delegateCallID != "" {
		return aitrace.ForDelegate(p.TurnID, delegateCallID).TraceID.String()
	}

	return aitrace.AnchorFor(aitrace.AnchorAssistantTurn, p.TurnID.String()).TraceID.String()
}

func (p *persistProposalsParams) parentOwnerKind(delegateCallID string) agent.RunOwnerKind {
	if delegateCallID == "" || p.TurnID.IsNil() {
		return ""
	}

	return agent.RunOwnerAssistantTurn
}

func (p *persistProposalsParams) parentOwnerID(delegateCallID string) pulid.ID {
	if delegateCallID == "" {
		return pulid.Nil
	}

	return p.TurnID
}

func (p *persistProposalsParams) delegateCallID(delegateCallID string) string {
	if p.TurnID.IsNil() {
		return ""
	}

	return delegateCallID
}

func tenantOf(actor *permissionports.RequestActor) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  actor.OrganizationID,
		BuID:   actor.BusinessUnitID,
		UserID: actor.UserID,
	}
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

func callOrder(saved []conversation.Message) map[string]int {
	order := make(map[string]int, len(saved))
	for idx := range saved {
		for _, call := range saved[idx].ToolCalls {
			order[call.ID] = len(order)
		}
	}

	return order
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
func chatEvidence(thread *conversation.Thread, messageID pulid.ID) []agentevidence.EvidenceRef {
	evidence := make([]agentevidence.EvidenceRef, 0, 2)
	evidence = append(evidence, agentevidence.EvidenceRef{
		Type: evidenceTypeThread,
		ID:   thread.ID.String(),
		Note: threadEvidenceNote(thread),
	})

	if !messageID.IsNil() {
		evidence = append(evidence, agentevidence.EvidenceRef{
			Type: evidenceTypeMessage,
			ID:   messageID.String(),
			Note: "The assistant turn that asked for this action",
		})
	}

	return evidence
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

// Evidence on a chat proposal points at the conversation, because that is where
// the reasoning is. An approver who wants more than the rationale line can open
// the thread and read the exchange that led to it, including the tool results the
// model saw.
const (
	evidenceTypeThread  = "AssistantThread"
	evidenceTypeMessage = "AssistantMessage"
)

func threadEvidenceNote(thread *conversation.Thread) string {
	if title := strings.TrimSpace(thread.Title); title != "" {
		return fmt.Sprintf("Proposed during the conversation %q", title)
	}

	return "Proposed during an assistant conversation"
}
