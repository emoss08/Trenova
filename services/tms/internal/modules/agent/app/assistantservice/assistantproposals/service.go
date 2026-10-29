package assistantproposals

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/attribution"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposaloutcomes"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/zap"
)

// SaveProposalEdits keeps the values a person changed on one of the
// conversation's pending proposals, such as the wording of a drafted message,
// so the edit is still there after a reload and goes with the approval.
//
// Reading the thread first is the authorization check, as for listing its
// proposals, and the proposal must be one the thread raised: a proposal from
// someone else's conversation is not found. Only what differs from the
// agent's own values is kept, and an empty set clears what was saved.
func (s *Service) SaveProposalEdits(
	ctx context.Context,
	req conversationports.GetThreadRequest,
	proposalID pulid.ID,
	modifications map[string]any,
) (*assistantproposalports.ProposalEdits, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.proposals == nil || s.proposalEdits == nil {
		return nil, errortypes.NewNotFoundError("Proposal not found in this conversation")
	}

	stored, err := s.proposals.ListByThread(
		ctx,
		agentproposalports.ListAgentProposalsByThreadRequest{
			ThreadID:   req.ID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	var proposal *agent.AgentProposal
	for _, candidate := range stored {
		if candidate != nil && candidate.ID == proposalID {
			proposal = candidate
			break
		}
	}
	if proposal == nil {
		return nil, errortypes.NewNotFoundError("Proposal not found in this conversation")
	}
	if err = editable(proposal); err != nil {
		return nil, err
	}

	changed := toolschema.Changed(proposal.ToolParams, modifications)
	if err = s.checkEditedFields(proposal, changed); err != nil {
		return nil, err
	}
	if len(changed) == 0 {
		changed = nil
	}

	saved, err := s.proposalEdits.SetPendingModifications(
		ctx,
		agentproposalports.SetPendingModificationsRequest{
			ID:            proposal.ID,
			TenantInfo:    req.TenantInfo,
			Modifications: changed,
		},
	)
	if err != nil {
		return nil, err
	}

	return &assistantproposalports.ProposalEdits{
		ProposalID:           saved.ID,
		PendingModifications: saved.PendingModifications,
	}, nil
}

// checkEditedFields refuses a change to a parameter a person may not edit:
// one the tool does not take, or the one naming the record it acts on. The
// approval checks the values again; this keeps what is saved to what the
// approval could carry.
func (s *Service) checkEditedFields(proposal *agent.AgentProposal, changed map[string]any) error {
	if s.tools == nil || len(changed) == 0 {
		return nil
	}

	editableNames := make(map[string]struct{})
	for _, field := range s.editableFields(proposal) {
		if !field.ReadOnly {
			editableNames[field.Name] = struct{}{}
		}
	}

	me := errortypes.NewMultiError()
	for name := range changed {
		if _, ok := editableNames[name]; !ok {
			me.Add(name, errortypes.ErrInvalid, "This value cannot be changed before approving")
		}
	}
	if me.HasErrors() {
		return me
	}

	return nil
}

// ListThreadProposals returns every proposal raised in a conversation, with the
// outcome of any that were approved.
//
// Reading the thread first is the authorization check, the same one that guards
// its messages: a thread is scoped to its owner, so someone else's thread is not
// found rather than returned.
func (s *Service) ListThreadProposals(
	ctx context.Context,
	req conversationports.GetThreadRequest,
) ([]assistantproposalports.AssistantProposal, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}

	stored, err := s.proposals.ListByThread(
		ctx,
		agentproposalports.ListAgentProposalsByThreadRequest{
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

	decided := s.outcomes().DecisionsFor(ctx, stored, req.TenantInfo)
	proposers := s.attribution().ProposersOf(ctx, req.TenantInfo, runsOf(stored))

	proposals := make([]assistantproposalports.AssistantProposal, 0, len(stored))
	for _, proposal := range stored {
		out := proposalstores.ToAssistantProposal(proposal, verdicts[proposal.RunID].Hold())
		out.Fields = s.editableFields(proposal)
		if decision := decided[proposal.ID]; decision != nil {
			decidedAt := decision.CreatedAt
			out.Modifications = proposalstores.ModificationsOf(decision)
			out.DecidedAt = &decidedAt
			out.DecidedByUserID = decision.DecidedByUserID
			out.DecisionNote = decision.Note
		}
		if by, ok := proposers[proposal.RunID]; ok {
			out.AgentID, out.AgentName = by.Id, by.Name
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
	proposals []assistantproposalports.AssistantProposal,
) {
	if s.labeler == nil {
		return
	}

	reads := fieldsensitivity.NewReadAccess(s.permissions, permissionports.UserActor(reader))
	refs := make(map[permission.Resource][]pulid.ID)
	for i := range proposals {
		for j := range proposals[i].Fields {
			field := &proposals[i].Fields[j]
			if field.Kind == toolschema.KindRecordSubset &&
				reads.MayRead(ctx, permission.Resource(field.Resource)) {
				agentruntimeports.SubsetChoiceRefs(refs, field)
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
			agentruntimeports.LabelSubsetChoices(&proposals[i].Fields[j], labels)
		}
	}
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

	return agenttoolports.ProposalFields(tool, proposal.ToolParams)
}

// ListThreadPlans reads the plans raised in a conversation, with the same
// hold a pending proposal carries when a shadow switch keeps it from being
// decided.
func (s *Service) ListThreadPlans(
	ctx context.Context,
	req conversationports.GetThreadRequest,
) ([]assistantproposalports.AssistantPlan, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.plans == nil {
		return []assistantproposalports.AssistantPlan{}, nil
	}

	stored, err := s.plans.ListByThread(ctx, agentplanports.ListAgentPlansByThreadRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	steps, err := s.threadPlanSteps(ctx, &req, len(stored) > 0)
	if err != nil {
		return nil, err
	}

	runIDs := make([]pulid.ID, 0, len(stored))
	deciders := make([]pulid.ID, 0, len(stored))
	for _, plan := range stored {
		if plan.Status.Decidable() {
			runIDs = append(runIDs, plan.RunID)
			runIDs = append(runIDs, stepRuns(steps[plan.ID])...)
		}
		if plan.DecidedByUserID != nil && plan.DecidedByUserID.IsNotNil() {
			deciders = append(deciders, *plan.DecidedByUserID)
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
	proposers := s.attribution().ProposersOf(ctx, req.TenantInfo, planRuns)
	names := s.attribution().UserNames(ctx, req.TenantInfo, sliceutils.Dedupe(deciders))

	plans := make([]assistantproposalports.AssistantPlan, 0, len(stored))
	for _, plan := range stored {
		planSteps := steps[plan.ID]
		out := toAssistantPlan(plan, planVerdict(verdicts, plan, stepRuns(planSteps)).Hold())
		out.RunMode = agentplan.PlanRunModeOf(planSteps)
		out.DecidedByName = names[out.DecidedByUserID]
		if by, ok := proposers[plan.RunID]; ok {
			out.AgentID, out.AgentName = by.Id, by.Name
		}
		plans = append(plans, out)
	}

	return plans, nil
}

func (s *Service) threadPlanSteps(
	ctx context.Context,
	req *conversationports.GetThreadRequest,
	needed bool,
) (map[pulid.ID][]*agent.AgentProposal, error) {
	steps := make(map[pulid.ID][]*agent.AgentProposal)
	if !needed || s.proposals == nil {
		return steps, nil
	}

	stored, err := s.proposals.ListByThread(
		ctx,
		agentproposalports.ListAgentProposalsByThreadRequest{
			ThreadID:   req.ID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	for _, proposal := range stored {
		if proposal == nil || proposal.PlanID == nil {
			continue
		}
		steps[*proposal.PlanID] = append(steps[*proposal.PlanID], proposal)
	}
	for planID := range steps {
		slices.SortStableFunc(steps[planID], func(a, b *agent.AgentProposal) int {
			return cmp.Compare(a.PlanStep, b.PlanStep)
		})
	}

	return steps, nil
}

type Service struct {
	logger        *zap.Logger
	conversations conversationports.ConversationRepository
	definitions   agentdefinitionports.AgentDefinitionRepository
	proposals     proposalstores.ProposalStore
	proposalEdits proposalEditStore
	plans         proposalstores.PlanStore
	shadow        *agentshadow.Resolver
	decisions     agentdecisionports.AgentDecisionRepository
	labeler       agentruntimeports.RecordLabeler
	permissions   permissionports.PermissionEngine
	tools         agenttoolports.AgentToolRegistry
	runs          agentruntimeports.AgentRunRepository
	users         tenantports.UserRepository
}

func (s *Service) attribution() *attribution.Attributor {
	return &attribution.Attributor{
		Logger:      s.logger,
		Definitions: s.definitions,
		Runs:        s.runs,
		Users:       s.users,
	}
}

func (s *Service) outcomes() *proposaloutcomes.Reader {
	return &proposaloutcomes.Reader{
		Logger:    s.logger,
		Proposals: s.proposals,
		Decisions: s.decisions,
	}
}
