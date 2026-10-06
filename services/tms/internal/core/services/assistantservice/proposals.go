package assistantservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentshadow"
	"github.com/emoss08/trenova/internal/core/services/fieldsensitivity"
	"github.com/emoss08/trenova/internal/core/services/proposalrecorder"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/zap"
)

type chatProposalStore interface {
	ListByThread(
		ctx context.Context,
		req repositories.ListAgentProposalsByThreadRequest,
	) ([]*agent.AgentProposal, error)
	ListByRun(
		ctx context.Context,
		req repositories.ListAgentProposalsByRunRequest,
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
	TurnID     pulid.ID
	Definition *agentdefinition.Definition
	Thread     *conversation.Thread
	Actor      *services.RequestActor
	// Saved are the messages as persisted, used to tie each proposal to the
	// assistant turn that asked for it.
	Saved   []conversation.Message
	Actions []services.PendingAction
	Model   string
	Input   string
	// Failed records the run as having ended before the turn finished.
	Failed bool
	// Artifacts, when set, views each outbound message as a draft and the
	// plan as a checklist beside the conversation.
	Artifacts *artifactRecorder
	// Taint is the outside content the turn that proposed read.
	Taint       *agent.RunTaint
	Fingerprint *agent.Fingerprint
	Delegations []services.DelegatedRun
}

type chatRun struct {
	definition     *agentdefinition.Definition
	delegateCallID string
	failed         bool
	model          string
	input          string
	fingerprint    *agent.Fingerprint
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
		Taint:            params.Taint,
		Delegated:        delegated,
		OpenEmptyRuns:    true,
		CallOrder:        callOrder(params.Saved),
		SourceMessageIDs: sourceMessageIndex(params.Saved),
		Evidence: func(_ services.PendingAction, sourceMessageID pulid.ID) []agent.EvidenceRef {
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
	persisted := make([]services.AssistantProposal, 0, len(params.Actions))
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
		hold := holdFor(verdict)
		for _, proposal := range group.proposals {
			out := toAssistantProposal(proposal, hold)
			out.AgentID = group.definition.ID
			out.AgentName = group.definition.Name
			persisted = append(persisted, out)
		}
		all = append(all, group.proposals...)
	}

	params.Artifacts.fromProposals(all, recorded.Plan)

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

func tenantOf(actor *services.RequestActor) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  actor.OrganizationID,
		BuID:   actor.BusinessUnitID,
		UserID: actor.UserID,
	}
}

// holdFor turns a shadow verdict into what the card shows. Nothing is held
// by a live verdict.
func holdFor(verdict agentshadow.Verdict) *services.ProposalHold {
	if !verdict.Shadow() {
		return nil
	}

	return &services.ProposalHold{
		Reason:    string(verdict.Cause),
		AgentName: verdict.AgentName,
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

	// Only a pending proposal can be held; a decided one is history whatever
	// the switches say now, and reading its run would be work for nothing.
	runIDs := make([]pulid.ID, 0, len(stored))
	for _, proposal := range stored {
		if proposal.Status == agent.ProposalStatusPending {
			runIDs = append(runIDs, proposal.RunID)
		}
	}
	verdicts, err := s.shadow.ForRuns(ctx, req.TenantInfo, runIDs)
	if err != nil {
		return nil, err
	}

	decided := s.decisionsFor(ctx, stored, req.TenantInfo)
	proposers := s.proposersOf(ctx, req.TenantInfo, runsOf(stored))

	proposals := make([]services.AssistantProposal, 0, len(stored))
	for _, proposal := range stored {
		out := toAssistantProposal(proposal, holdFor(verdicts[proposal.RunID]))
		out.Fields = s.editableFields(proposal)
		if decision := decided[proposal.ID]; decision != nil {
			decidedAt := decision.CreatedAt
			out.Modifications = modificationsOf(decision)
			out.DecidedAt = &decidedAt
			out.DecidedByUserID = decision.DecidedByUserID
			out.DecisionNote = decision.Note
		}
		if by, ok := proposers[proposal.RunID]; ok {
			out.AgentID, out.AgentName = by.id, by.name
		}
		proposals = append(proposals, out)
	}

	reader := req.TenantInfo
	if reader.UserID.IsNil() {
		reader.UserID = req.UserID
	}
	s.labelChoices(ctx, reader, proposals)

	return proposals, nil
}

// labelChoices names the records each pending proposal offers its reader to
// untick, in one read across the thread, for the resources the reader may
// read. The rest, and every record when the read fails, keep their ids, which
// the proposal's parameters already show.
func (s *Service) labelChoices(
	ctx context.Context,
	reader pagination.TenantInfo,
	proposals []services.AssistantProposal,
) {
	if s.labeler == nil {
		return
	}

	reads := fieldsensitivity.NewReadAccess(s.permissions, services.UserActor(reader))
	refs := make(map[permission.Resource][]pulid.ID)
	for i := range proposals {
		for j := range proposals[i].Fields {
			field := &proposals[i].Fields[j]
			if field.Kind == toolschema.KindRecordSubset &&
				reads.MayRead(ctx, permission.Resource(field.Resource)) {
				services.SubsetChoiceRefs(refs, field)
			}
		}
	}
	if len(refs) == 0 {
		return
	}

	labels, err := s.labeler.Labels(ctx, reader, refs)
	if err != nil {
		s.logger.Warn("could not read the labels of the records a proposal offers",
			zap.Error(err))

		return
	}

	for i := range proposals {
		for j := range proposals[i].Fields {
			services.LabelSubsetChoices(&proposals[i].Fields[j], labels)
		}
	}
}

// runsOf is the run behind each proposal.
func runsOf(stored []*agent.AgentProposal) []pulid.ID {
	ids := make([]pulid.ID, 0, len(stored))
	for _, proposal := range stored {
		if proposal != nil {
			ids = append(ids, proposal.RunID)
		}
	}

	return ids
}

// editableFields is what a person may change on a pending proposal, from
// the tool's own schema, with the parameter naming its record read-only. A
// decided proposal has nothing left to edit and a tool the registry no longer
// has cannot be edited into running.
func (s *Service) editableFields(proposal *agent.AgentProposal) []toolschema.Field {
	if s.tools == nil || proposal.Status != agent.ProposalStatusPending {
		return []toolschema.Field{}
	}

	tool, ok := s.tools.Get(proposal.ToolName)
	if !ok {
		return []toolschema.Field{}
	}

	return services.ProposalFields(tool, proposal.ToolParams)
}

// decisionsFor reads the decision behind each decided proposal, keyed by
// proposal: what the approver changed, who decided it, when, and what they
// told the agent. A read that fails degrades to showing none: the
// proposal's status is the record and the card still says how it ended.
func (s *Service) decisionsFor(
	ctx context.Context,
	stored []*agent.AgentProposal,
	tenant pagination.TenantInfo,
) map[pulid.ID]*agent.AgentDecision {
	if s.decisions == nil {
		return nil
	}

	ids := make([]pulid.ID, 0, len(stored))
	for _, proposal := range stored {
		if proposal != nil && proposal.Status != agent.ProposalStatusPending {
			ids = append(ids, proposal.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	decisions, err := s.decisions.ListByProposals(
		ctx,
		repositories.ListAgentDecisionsByProposalsRequest{
			ProposalIDs: ids,
			TenantInfo:  tenant,
		},
	)
	if err != nil {
		s.logger.Warn("could not read the decisions behind the thread's proposals", zap.Error(err))

		return nil
	}

	out := make(map[pulid.ID]*agent.AgentDecision, len(decisions))
	for _, decision := range decisions {
		if decision == nil || decision.ProposalID == nil {
			continue
		}
		if current, ok := out[*decision.ProposalID]; ok && current.CreatedAt > decision.CreatedAt {
			continue
		}
		out[*decision.ProposalID] = decision
	}

	return out
}

func modificationsOf(decision *agent.AgentDecision) map[string]any {
	if decision == nil || decision.Decision != agent.DecisionModified ||
		len(decision.Modifications) == 0 {
		return nil
	}

	return decision.Modifications
}

type chatPlanStore interface {
	ListByThread(
		ctx context.Context,
		req repositories.ListAgentPlansByThreadRequest,
	) ([]*agent.AgentPlan, error)
}

// planStoreOrNil keeps a typed nil out of the interface, so the absence of a
// plan store reads as nil rather than as a store that panics.
func planStoreOrNil(plans repositories.AgentPlanRepository) chatPlanStore {
	if plans == nil {
		return nil
	}

	return plans
}

// ListThreadPlans reads the plans raised in a conversation, with the same
// hold a pending proposal carries when a shadow switch keeps it from being
// decided.
func (s *Service) ListThreadPlans(
	ctx context.Context,
	req repositories.GetThreadRequest,
) ([]services.AssistantPlan, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.plans == nil {
		return []services.AssistantPlan{}, nil
	}

	stored, err := s.plans.ListByThread(ctx, repositories.ListAgentPlansByThreadRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	runIDs := make([]pulid.ID, 0, len(stored))
	for _, plan := range stored {
		if plan.Status.Decidable() {
			runIDs = append(runIDs, plan.RunID)
		}
	}
	stepRuns, err := s.planStepRuns(ctx, &req, len(runIDs) > 0)
	if err != nil {
		return nil, err
	}
	for _, plan := range stored {
		if plan.Status.Decidable() {
			runIDs = append(runIDs, stepRuns[plan.ID]...)
		}
	}
	verdicts, err := s.shadow.ForRuns(ctx, req.TenantInfo, sliceutils.Dedupe(runIDs))
	if err != nil {
		return nil, err
	}

	planRuns := make([]pulid.ID, 0, len(stored))
	for _, plan := range stored {
		planRuns = append(planRuns, plan.RunID)
	}
	proposers := s.proposersOf(ctx, req.TenantInfo, planRuns)

	plans := make([]services.AssistantPlan, 0, len(stored))
	for _, plan := range stored {
		out := toAssistantPlan(plan, holdFor(planVerdict(verdicts, plan, stepRuns[plan.ID])))
		if by, ok := proposers[plan.RunID]; ok {
			out.AgentID, out.AgentName = by.id, by.name
		}
		plans = append(plans, out)
	}

	return plans, nil
}

func (s *Service) planStepRuns(
	ctx context.Context,
	req *repositories.GetThreadRequest,
	needed bool,
) (map[pulid.ID][]pulid.ID, error) {
	runs := make(map[pulid.ID][]pulid.ID)
	if !needed || s.proposals == nil {
		return runs, nil
	}

	stored, err := s.proposals.ListByThread(ctx, repositories.ListAgentProposalsByThreadRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	for _, proposal := range stored {
		if proposal == nil || proposal.PlanID == nil ||
			slices.Contains(runs[*proposal.PlanID], proposal.RunID) {
			continue
		}
		runs[*proposal.PlanID] = append(runs[*proposal.PlanID], proposal.RunID)
	}

	return runs, nil
}

func planVerdict(
	verdicts map[pulid.ID]agentshadow.Verdict,
	plan *agent.AgentPlan,
	stepRuns []pulid.ID,
) agentshadow.Verdict {
	if !plan.Status.Decidable() {
		return agentshadow.Verdict{}
	}
	if verdict := verdicts[plan.RunID]; verdict.Shadow() {
		return verdict
	}
	for _, runID := range stepRuns {
		if verdict := verdicts[runID]; verdict.Shadow() {
			return verdict
		}
	}

	return agentshadow.Verdict{}
}

func toAssistantPlan(plan *agent.AgentPlan, hold *services.ProposalHold) services.AssistantPlan {
	decidedBy := pulid.Nil
	if plan.DecidedByUserID != nil {
		decidedBy = *plan.DecidedByUserID
	}

	return services.AssistantPlan{
		ID:              plan.ID,
		RunID:           plan.RunID,
		Title:           plan.Title,
		Summary:         plan.Summary,
		Status:          plan.Status,
		StepCount:       plan.StepCount,
		CompletedSteps:  plan.CompletedSteps,
		FailedStep:      plan.FailedStep,
		FailureError:    plan.FailureError,
		DecidedAt:       plan.DecidedAt,
		DecidedByUserID: decidedBy,
		ExpiresAt:       plan.ExpiresAt,
		Hold:            hold,
		CreatedAt:       plan.CreatedAt,
	}
}

func toAssistantProposal(
	proposal *agent.AgentProposal,
	hold *services.ProposalHold,
) services.AssistantProposal {
	out := services.AssistantProposal{
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
		ExecutionResult: proposal.ExecutionResult,
		ExpiresAt:       proposal.ExpiresAt,
		Hold:            hold,
		PlanStep:        proposal.PlanStep,
		SimulatedAt:     proposal.SimulatedAt,
		Simulation:      proposal.Simulation,
		CreatedAt:       proposal.CreatedAt,
	}
	if proposal.PlanID != nil {
		out.PlanID = *proposal.PlanID
	}
	if proposal.Status == agent.ProposalStatusPending && len(proposal.PendingModifications) > 0 {
		out.PendingModifications = proposal.PendingModifications
	}

	return out
}

// proposalOutcomes reads what became of every proposal this conversation
// raised, for the model to see the current state of each. A read that fails
// degrades to the stored tool results rather than failing the turn: a stale
// "awaiting review" is what the model always used to see.
func (s *Service) proposalOutcomes(
	ctx context.Context,
	thread *conversation.Thread,
	tenant pagination.TenantInfo,
) []services.ProposalOutcome {
	if s.proposals == nil {
		return nil
	}

	stored, err := s.proposals.ListByThread(ctx, repositories.ListAgentProposalsByThreadRequest{
		ThreadID:   thread.ID,
		TenantInfo: tenant,
	})
	if err != nil {
		s.logger.Warn("could not read the thread's proposals for the model",
			zap.String("thread", thread.ID.String()), zap.Error(err))

		return nil
	}

	decided := s.decisionsFor(ctx, stored, tenant)

	outcomes := make([]services.ProposalOutcome, 0, len(stored))
	for _, proposal := range stored {
		if proposal == nil {
			continue
		}
		planID := pulid.Nil
		if proposal.PlanID != nil {
			planID = *proposal.PlanID
		}
		outcomes = append(outcomes, services.ProposalOutcome{
			ProposalID:      proposal.ID,
			PlanID:          planID,
			SourceMessageID: proposal.SourceMessageID,
			ToolName:        proposal.ToolName,
			ToolParams:      proposal.ToolParams,
			Rationale:       proposal.Rationale,
			Status:          proposal.Status,
			AutonomyTier:    proposal.AutonomyTier,
			ExecutionError:  proposal.ExecutionError,
			ExecutedAt:      proposal.ExecutedAt,
			ExecutionResult: proposal.ExecutionResult,
			Modifications:   modificationsOf(decided[proposal.ID]),
		})
	}

	return outcomes
}
