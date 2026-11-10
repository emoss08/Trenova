package deskactivityservice

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/deskrecords"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentplanports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskactivityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskattentionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/textcase"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const (
	defaultEntries   = 20
	maxEntries       = 50
	readPerEntry     = 10
	maxRead          = 500
	recordsPerEntry  = 5
	maxLookbackHours = 24 * 31
	ranLookbackDays  = 30
)

type Params struct {
	fx.In

	Proposals agentproposalports.AgentProposalRepository
	Decisions agentdecisionports.AgentDecisionRepository
	Plans     agentplanports.AgentPlanRepository
	Runs      agentruntimeports.AgentRunRepository
	Labels    *deskrecords.Labeler
}

type Service struct {
	proposals agentproposalports.AgentProposalRepository
	decisions agentdecisionports.AgentDecisionRepository
	plans     agentplanports.AgentPlanRepository
	runs      agentruntimeports.AgentRunRepository
	labels    *deskrecords.Labeler
	now       func() int64
}

func New(p Params) deskactivityports.DeskActivityService {
	return &Service{
		proposals: p.Proposals,
		decisions: p.Decisions,
		plans:     p.Plans,
		runs:      p.Runs,
		labels:    p.Labels,
		now:       timeutils.NowUnix,
	}
}

func (s *Service) Activity(
	ctx context.Context,
	req *deskactivityports.ActivityRequest,
	actor *permissionports.RequestActor,
) (*deskactivityports.Activity, error) {
	earliest := s.now() - maxLookbackHours*3600
	if req.Since < earliest {
		return nil, errortypes.NewValidationError(
			"since",
			errortypes.ErrInvalid,
			"Activity reaches back at most 31 days",
		)
	}
	first := req.First
	if first <= 0 {
		first = defaultEntries
	}
	first = min(first, maxEntries)
	limit := min(first*readPerEntry, maxRead)

	proposals, err := s.proposals.ListSettled(ctx, &agentproposalports.ListSettledProposalsRequest{
		TenantInfo: req.TenantInfo,
		Since:      req.Since,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	entries := fold(proposals)
	out := &deskactivityports.Activity{HasMore: len(proposals) == limit || len(entries) > first}
	if len(entries) > first {
		entries = entries[:first]
	}
	out.Entries = entries
	if len(entries) == 0 {
		out.Entries = []*deskactivityports.Entry{}
		return out, nil
	}

	if err = s.attachAgents(ctx, req.TenantInfo, entries); err != nil {
		return nil, err
	}
	if err = s.attachDecisions(ctx, req.TenantInfo, entries); err != nil {
		return nil, err
	}

	records := make([]*deskattentionports.Record, 0, len(entries)*recordsPerEntry)
	for _, entry := range entries {
		records = append(records, entry.Records...)
	}
	if err = s.labels.Label(ctx, req.TenantInfo, actor, records); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Service) attachAgents(
	ctx context.Context,
	tenant pagination.TenantInfo,
	entries []*deskactivityports.Entry,
) error {
	runIDs := make([]pulid.ID, 0, len(entries))
	for _, entry := range entries {
		runIDs = append(runIDs, entry.RunID)
	}
	runs, err := s.runs.ListByIDs(ctx, &agentruntimeports.ListAgentRunsByIDsRequest{
		IDs:        distinct(runIDs),
		TenantInfo: tenant,
	})
	if err != nil {
		return err
	}
	byRun := make(map[pulid.ID]pulid.ID, len(runs))
	for _, run := range runs {
		if run != nil {
			byRun[run.ID] = run.AgentDefinitionID
		}
	}
	for _, entry := range entries {
		entry.AgentDefinitionID = byRun[entry.RunID]
	}

	return nil
}

func (s *Service) attachDecisions(
	ctx context.Context,
	tenant pagination.TenantInfo,
	entries []*deskactivityports.Entry,
) error {
	proposalIDs := make([]pulid.ID, 0, len(entries))
	planIDs := make([]pulid.ID, 0, len(entries))
	for _, entry := range entries {
		if entry.Outcome == deskactivityports.OutcomeOnItsOwn ||
			entry.Outcome == deskactivityports.OutcomePreviewed {
			continue
		}
		if entry.PlanID.IsNotNil() {
			continue
		}
		proposalIDs = append(proposalIDs, entry.ProposalIDs...)
	}
	for _, entry := range entries {
		if entry.PlanID.IsNotNil() {
			planIDs = append(planIDs, entry.PlanID)
		}
	}

	deciders := map[pulid.ID]pulid.ID{}
	planTitles := map[pulid.ID]string{}
	if len(proposalIDs) > 0 {
		decisions, err := s.decisions.ListByProposals(
			ctx,
			agentdecisionports.ListAgentDecisionsByProposalsRequest{
				ProposalIDs: distinct(proposalIDs),
				TenantInfo:  tenant,
			},
		)
		if err != nil {
			return err
		}
		for _, decision := range decisions {
			if decision != nil && decision.ProposalID != nil {
				deciders[*decision.ProposalID] = decision.DecidedByUserID
			}
		}
	}
	if len(planIDs) > 0 {
		plans, err := s.plans.ListByIDs(ctx, agentplanports.ListAgentPlansByIDsRequest{
			IDs:        distinct(planIDs),
			TenantInfo: tenant,
		})
		if err != nil {
			return err
		}
		for _, plan := range plans {
			if plan == nil {
				continue
			}
			planTitles[plan.ID] = plan.Title
			if plan.DecidedByUserID != nil {
				deciders[plan.ID] = *plan.DecidedByUserID
			}
		}
	}

	for _, entry := range entries {
		if entry.PlanID.IsNotNil() {
			if title := planTitles[entry.PlanID]; title != "" {
				entry.Title = title
			}
			if entry.Outcome != deskactivityports.OutcomeOnItsOwn &&
				entry.Outcome != deskactivityports.OutcomePreviewed {
				entry.DecidedByUserID = deciders[entry.PlanID]
			}
			continue
		}
		for _, id := range entry.ProposalIDs {
			if decider := deciders[id]; decider.IsNotNil() {
				entry.DecidedByUserID = decider
				break
			}
		}
	}

	return nil
}

func (s *Service) Entry(
	ctx context.Context,
	req *deskactivityports.EntryRequest,
	actor *permissionports.RequestActor,
) (*deskactivityports.EntryDetail, error) {
	tenant := req.TenantInfo
	proposal, err := s.proposals.GetByID(ctx, agentproposalports.GetAgentProposalByIDRequest{
		ID:         req.ID,
		TenantInfo: &tenant,
	})
	if err != nil {
		return nil, err
	}
	if proposal.ExecutedAt == nil && proposal.SimulatedAt == nil {
		return nil, errortypes.NewNotFoundError("This change has not settled")
	}

	siblings, err := s.proposals.ListByRun(ctx, agentproposalports.ListAgentProposalsByRunRequest{
		RunID:      proposal.RunID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}
	settled := make([]*agent.AgentProposal, 0, len(siblings))
	for _, sibling := range siblings {
		if sibling != nil && (sibling.ExecutedAt != nil || sibling.SimulatedAt != nil) {
			settled = append(settled, sibling)
		}
	}
	slices.SortStableFunc(settled, func(a, b *agent.AgentProposal) int {
		if c := cmp.Compare(settledAt(b), settledAt(a)); c != 0 {
			return c
		}
		return cmp.Compare(b.ID.String(), a.ID.String())
	})

	entry, members := entryHolding(settled, proposal.ID)
	if entry == nil {
		return nil, errortypes.NewNotFoundError("This change is not in the ledger")
	}
	if err = s.attachAgents(ctx, tenant, []*deskactivityports.Entry{entry}); err != nil {
		return nil, err
	}
	if err = s.attachDecisions(ctx, tenant, []*deskactivityports.Entry{entry}); err != nil {
		return nil, err
	}

	changes := make([]*deskactivityports.Change, 0, len(members))
	records := make([]*deskattentionports.Record, 0, len(members)+len(entry.Records))
	records = append(records, entry.Records...)
	for _, member := range members {
		change := &deskactivityports.Change{
			ProposalID: member.ID,
			At:         settledAt(member),
			Outcome:    outcomeOf(member),
			Error:      member.ExecutionError,
		}
		if kind, ok := recordkind.OfResource(permission.Resource(member.TargetResource)); ok &&
			member.TargetID.IsNotNil() {
			change.Record = &deskattentionports.Record{Kind: kind, ID: member.TargetID}
			records = append(records, change.Record)
		}
		changes = append(changes, change)
	}
	if err = s.labels.Label(ctx, tenant, actor, records); err != nil {
		return nil, err
	}

	ranSince := s.now() - ranLookbackDays*24*3600
	ranBefore := 0
	if entry.AgentDefinitionID.IsNotNil() {
		ranBefore, err = s.proposals.CountExecutedTool(ctx, &agentproposalports.CountExecutedToolRequest{
			TenantInfo:   tenant,
			DefinitionID: entry.AgentDefinitionID,
			ToolName:     proposal.ToolName,
			Since:        ranSince,
		})
		if err != nil {
			return nil, err
		}
	}

	return &deskactivityports.EntryDetail{
		Entry:     entry,
		ToolName:  toolPhrase(proposal.ToolName),
		Changes:   changes,
		RanBefore: ranBefore,
		RanSince:  ranSince,
	}, nil
}

func entryHolding(
	settled []*agent.AgentProposal,
	id pulid.ID,
) (*deskactivityports.Entry, []*agent.AgentProposal) {
	entries := fold(settled)
	offset := 0
	for _, entry := range entries {
		members := settled[offset : offset+entry.Count]
		offset += entry.Count
		if slices.Contains(entry.ProposalIDs, id) {
			return entry, members
		}
	}

	return nil, nil
}

func outcomeOf(p *agent.AgentProposal) deskactivityports.Outcome {
	switch {
	case p.SimulatedAt != nil:
		return deskactivityports.OutcomePreviewed
	case p.ExecutionError != "" || p.Status == agent.ProposalStatusExecutionFailed:
		return deskactivityports.OutcomeFailed
	case p.AutonomyTier == agent.TierAutoExecute:
		return deskactivityports.OutcomeOnItsOwn
	default:
		return deskactivityports.OutcomeApproved
	}
}

func settledAt(p *agent.AgentProposal) int64 {
	switch {
	case p.ExecutedAt != nil:
		return *p.ExecutedAt
	case p.SimulatedAt != nil:
		return *p.SimulatedAt
	}

	return p.UpdatedAt
}

func planOf(p *agent.AgentProposal) pulid.ID {
	if p.PlanID == nil {
		return pulid.Nil
	}

	return *p.PlanID
}

func foldKey(p *agent.AgentProposal) string {
	if plan := planOf(p); plan.IsNotNil() {
		return "plan|" + plan.String() + "|" + string(outcomeOf(p))
	}

	return strings.Join([]string{
		p.RunID.String(),
		planOf(p).String(),
		p.ToolName,
		string(outcomeOf(p)),
	}, "|")
}

func fold(proposals []*agent.AgentProposal) []*deskactivityports.Entry {
	entries := make([]*deskactivityports.Entry, 0, len(proposals))
	var current *deskactivityports.Entry
	var currentKey string
	var currentRuns []*agent.AgentProposal
	flush := func() {
		if current == nil {
			return
		}
		current.Title = titleOf(currentRuns, current.Outcome)
		entries = append(entries, current)
	}
	for _, p := range proposals {
		if p == nil {
			continue
		}
		key := foldKey(p)
		if current == nil || key != currentKey {
			flush()
			current = &deskactivityports.Entry{
				ID:      p.ID,
				Outcome: outcomeOf(p),
				At:      settledAt(p),
				RunID:   p.RunID,
				PlanID:  planOf(p),
				Detail:  detailOf(p),
			}
			currentKey = key
			currentRuns = nil
		}
		current.Count++
		current.ProposalIDs = append(current.ProposalIDs, p.ID)
		currentRuns = append(currentRuns, p)
		addRecord(current, p)
	}
	flush()

	return entries
}

func addRecord(entry *deskactivityports.Entry, p *agent.AgentProposal) {
	if p.TargetID.IsNil() {
		return
	}
	kind, ok := recordkind.OfResource(permission.Resource(p.TargetResource))
	if !ok {
		return
	}
	for _, record := range entry.Records {
		if record.ID == p.TargetID {
			return
		}
	}
	if len(entry.Records) == recordsPerEntry {
		entry.RecordsMore++
		return
	}
	entry.Records = append(entry.Records, &deskattentionports.Record{Kind: kind, ID: p.TargetID})
}

func detailOf(p *agent.AgentProposal) string {
	if p.ExecutionError != "" {
		return p.ExecutionError
	}

	return strings.TrimSpace(p.Rationale)
}

func toolPhrase(toolName string) string {
	return textcase.HumanizeSnakeCase(agenttoolports.ReadingOrder(toolName))
}

func titleOf(proposals []*agent.AgentProposal, outcome deskactivityports.Outcome) string {
	if len(proposals) == 0 {
		return ""
	}
	first := proposals[0]
	result := first.ExecutionResult
	sameResult := result != nil && result.Action != "" && result.Kind != ""
	for _, p := range proposals[1:] {
		if p.ExecutionResult == nil || p.ExecutionResult.Action != result.Action ||
			p.ExecutionResult.Kind != result.Kind {
			sameResult = false
			break
		}
	}

	if outcome == deskactivityports.OutcomeOnItsOwn || outcome == deskactivityports.OutcomeApproved {
		if sameResult {
			action := textcase.CapitalizeFirst(result.Action)
			if len(proposals) == 1 {
				return strings.TrimSpace(action + " " + result.Kind + " " + result.Name)
			}
			return action + " " + strconv.Itoa(len(proposals)) + " " + inflect.PluralNoun(result.Kind)
		}
	}

	phrase := textcase.CapitalizeFirst(toolPhrase(first.ToolName))
	if len(proposals) == 1 {
		return phrase
	}

	return phrase + ", " + strconv.Itoa(len(proposals)) + " times"
}

func distinct(ids []pulid.ID) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(ids))
	out := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		if id.IsNil() {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}

	return out
}
