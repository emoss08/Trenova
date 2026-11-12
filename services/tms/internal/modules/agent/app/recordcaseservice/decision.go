package recordcaseservice

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordcaseports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/textcase"
	"go.uber.org/zap"
)

const (
	draftGraceSeconds = 600
	draftLimitSeconds = 1800
	maxDraftNote      = 600
	// justDecidedSeconds is how long a plan that just stopped stays the
	// case's own ahead of another decision waiting on the record, and a plan
	// that just ran stays ahead of older finished ones.
	justDecidedSeconds = 900
	rankStep           = 10
	justFailedRank     = 5
	justRanRank        = 15
)

func (s *Service) decide(
	ctx context.Context,
	req *recordcaseports.CaseRequest,
	built *composed,
) error {
	if built.chosen != nil && built.chosen.Kind == recordcase.OptionKindLeave {
		return nil
	}

	var run pulid.ID
	if built.chosen != nil {
		run = built.chosen.DraftRunID
	}
	found, err := s.repo.Decisions(ctx, &recordcaseports.DecisionsRequest{
		TenantInfo: req.TenantInfo,
		RecordID:   req.ID,
		RunID:      run,
		HandedRuns: handedRuns(built.items),
		Since:      built.now - decidedLookback,
	})
	if err != nil {
		return err
	}

	candidates := make([]*recordcase.Decision, 0, len(found.Plans)+len(found.Proposals))
	for _, plan := range found.Plans {
		if decision := planDecision(plan); decision != nil {
			candidates = append(candidates, decision)
		}
	}
	for _, proposal := range found.Proposals {
		if decision := proposalDecision(proposal, found); decision != nil {
			candidates = append(candidates, decision)
		}
	}
	slices.SortStableFunc(candidates, func(a, b *recordcase.Decision) int {
		if c := cmp.Compare(rank(a, built.now), rank(b, built.now)); c != 0 {
			return c
		}
		if c := cmp.Compare(kindRank(a.Kind), kindRank(b.Kind)); c != 0 {
			return c
		}

		return cmp.Compare(b.ProposedAt, a.ProposedAt)
	})

	candidates = s.live(ctx, req, built, candidates)
	if len(candidates) > 0 {
		picked := candidates[0]
		picked.Others = otherPending(candidates[1:], picked.Title)
		picked.ItemID = itemFor(built.items, picked.ID)
		built.decision = picked
	}
	if built.chosen != nil && built.chosen.Drafting() && built.decision == nil {
		built.drafting, built.draftNote = s.draftState(ctx, req, built)
	}

	return nil
}

// live keeps the decisions a person can act on. A plan or a change made by an
// agent in shadow, or while every agent is paused, is refused approval until
// that is lifted, so it is no part of a record's case: not its plan, and not
// one of the things open about it.
func (s *Service) live(
	ctx context.Context,
	req *recordcaseports.CaseRequest,
	built *composed,
	candidates []*recordcase.Decision,
) []*recordcase.Decision {
	runIDs := make([]pulid.ID, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.RunID.IsNotNil() && !slices.Contains(runIDs, candidate.RunID) {
			runIDs = append(runIDs, candidate.RunID)
		}
	}
	if len(runIDs) == 0 {
		return candidates
	}
	verdicts, err := s.shadow.ForRuns(ctx, req.TenantInfo, runIDs)
	if err != nil {
		s.l.Warn("could not tell which of a case's decisions are live", zap.Error(err))

		return candidates
	}

	held := make(map[string]struct{}, len(candidates))
	kept := make([]*recordcase.Decision, 0, len(candidates))
	for _, candidate := range candidates {
		if verdicts[candidate.RunID].Shadow() {
			held[candidate.ID.String()] = struct{}{}
			continue
		}
		kept = append(kept, candidate)
	}
	if len(held) > 0 {
		built.items = slices.DeleteFunc(built.items, func(item *watchtower.Item) bool {
			_, shadowed := held[item.SourceID]

			return shadowed
		})
	}

	return kept
}

func (s *Service) draftState(
	ctx context.Context,
	req *recordcaseports.CaseRequest,
	built *composed,
) (drafting bool, note string) {
	requested := *built.chosen.DraftRequestedAt
	run, err := s.runRecords.GetByID(ctx, agentruntimeports.GetAgentRunByIDRequest{
		ID:         built.chosen.DraftRunID,
		TenantInfo: &req.TenantInfo,
	})
	if err != nil {
		return requested > built.now-draftGraceSeconds, ""
	}
	switch run.Status {
	case agent.RunStatusCompleted, agent.RunStatusShadowCompleted, agent.RunStatusFailed:
		return false, stringutils.Ellipsize(
			stringutils.FirstNonEmptyTrimmed(run.ErrorMessage, run.Summary),
			maxDraftNote,
		)
	case agent.RunStatusPending,
		agent.RunStatusGatheringContext,
		agent.RunStatusDiagnosing,
		agent.RunStatusAwaitingDecision:
		drafting = requested > built.now-draftLimitSeconds
		if drafting {
			built.stream = run.WorkflowID
		}

		return drafting, ""
	default:
		return false, ""
	}
}

func planDecision(plan *agentplan.AgentPlan) *recordcase.Decision {
	phase, ok := planPhase(plan.Status)
	if !ok {
		return nil
	}
	decision := &recordcase.Decision{
		Kind:           recordcase.DecisionPlan,
		ID:             plan.ID,
		RunID:          plan.RunID,
		Title:          plan.Title,
		Phase:          phase,
		ProposedAt:     plan.CreatedAt,
		StepCount:      plan.StepCount,
		CompletedSteps: plan.CompletedSteps,
		FailedStep:     plan.FailedStep,
		FailureError:   plan.FailureError,
		DecidedAt:      plan.DecidedAt,
		Note:           plan.ApprovalNote,
	}
	if phase == recordcase.PhaseUndoable {
		decision.CommitsAt = plan.CommitsAt
	}
	if plan.DecidedByUserID != nil {
		decision.DecidedByUserID = *plan.DecidedByUserID
	}
	if plan.Run != nil {
		decision.AgentDefinitionID = plan.Run.AgentDefinitionID
	}

	return decision
}

func planPhase(status agentplan.PlanStatus) (recordcase.DecisionPhase, bool) {
	switch status {
	case agentplan.PlanStatusPending:
		return recordcase.PhaseAwaiting, true
	case agentplan.PlanStatusApproving:
		return recordcase.PhaseUndoable, true
	case agentplan.PlanStatusApproved:
		return recordcase.PhaseRunning, true
	case agentplan.PlanStatusCompleted:
		return recordcase.PhaseDone, true
	case agentplan.PlanStatusFailed:
		return recordcase.PhaseFailed, true
	case agentplan.PlanStatusRejected:
		return recordcase.PhaseDeclined, true
	case agentplan.PlanStatusExpired:
		return "", false
	default:
		return "", false
	}
}

func proposalDecision(
	proposal *agent.AgentProposal,
	found *recordcaseports.Decisions,
) *recordcase.Decision {
	phase, ok := proposalPhase(proposal.Status)
	if !ok {
		return nil
	}
	decision := &recordcase.Decision{
		Kind:  recordcase.DecisionProposal,
		ID:    proposal.ID,
		RunID: proposal.RunID,
		Title: textcase.CapitalizeFirst(
			textcase.HumanizeSnakeCase(agenttoolports.ReadingOrder(proposal.ToolName)),
		),
		Phase:        phase,
		ProposedAt:   proposal.CreatedAt,
		StepCount:    1,
		FailureError: proposal.ExecutionError,
		DecidedAt:    proposal.ExecutedAt,
	}
	if phase == recordcase.PhaseDone {
		decision.CompletedSteps = 1
	}
	if phase == recordcase.PhaseFailed {
		failed := 1
		decision.FailedStep = &failed
	}
	if proposal.Run != nil {
		decision.AgentDefinitionID = proposal.Run.AgentDefinitionID
	}
	if decided := found.Decided[proposal.ID]; decided != nil {
		decision.Note = decided.Note
		decision.DecidedByUserID = decided.DecidedByUserID
		if decision.DecidedAt == nil {
			at := decided.CreatedAt
			decision.DecidedAt = &at
		}
		if phase == recordcase.PhaseUndoable {
			decision.CommitsAt = decided.CommitsAt
		}
	}

	return decision
}

func proposalPhase(status agent.ProposalStatus) (recordcase.DecisionPhase, bool) {
	switch status {
	case agent.ProposalStatusPending:
		return recordcase.PhaseAwaiting, true
	case agent.ProposalStatusApproving:
		return recordcase.PhaseUndoable, true
	case agent.ProposalStatusAccepted, agent.ProposalStatusModified:
		return recordcase.PhaseRunning, true
	case agent.ProposalStatusExecuted:
		return recordcase.PhaseDone, true
	case agent.ProposalStatusExecutionFailed:
		return recordcase.PhaseFailed, true
	case agent.ProposalStatusRejected:
		return recordcase.PhaseDeclined, true
	case agent.ProposalStatusExpired,
		agent.ProposalStatusSuperseded,
		agent.ProposalStatusSkipped,
		agent.ProposalStatusSimulated:
		return "", false
	default:
		return "", false
	}
}

// rank orders a record's decisions for its case. What is in motion comes
// first; then a plan that just stopped, which the person who approved it has
// to see; then what waits, so a record said to need approval always shows the
// thing to approve; then a plan that just ran; then the rest.
func rank(decision *recordcase.Decision, now int64) int {
	if decision.DecidedAt != nil && now-*decision.DecidedAt < justDecidedSeconds {
		switch decision.Phase {
		case recordcase.PhaseFailed:
			return justFailedRank
		case recordcase.PhaseDone:
			return justRanRank
		case recordcase.PhaseAwaiting,
			recordcase.PhaseUndoable,
			recordcase.PhaseRunning,
			recordcase.PhaseDeclined,
			recordcase.PhaseExpired:
		}
	}

	return phaseRank(decision.Phase) * rankStep
}

func phaseRank(phase recordcase.DecisionPhase) int {
	switch phase {
	case recordcase.PhaseUndoable, recordcase.PhaseRunning:
		return 0
	case recordcase.PhaseAwaiting:
		return 1
	case recordcase.PhaseFailed:
		return 2
	case recordcase.PhaseDone, recordcase.PhaseDeclined, recordcase.PhaseExpired:
		return 3
	default:
		return 4
	}
}

func kindRank(kind recordcase.DecisionKind) int {
	if kind == recordcase.DecisionPlan {
		return 0
	}

	return 1
}

func otherPending(rest []*recordcase.Decision, title string) int {
	titles := map[string]struct{}{title: {}}
	for _, decision := range rest {
		if decision.Phase == recordcase.PhaseAwaiting {
			titles[decision.Title] = struct{}{}
		}
	}

	return len(titles) - 1
}

func itemFor(items []*watchtower.Item, id pulid.ID) pulid.ID {
	for _, item := range items {
		if item.SourceKind.DeskLens() == watchtower.DeskLensDecisions &&
			item.SourceID == id.String() {
			return item.ID
		}
	}

	return pulid.Nil
}

func handedRuns(items []*watchtower.Item) []pulid.ID {
	runs := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		if item.HasAgent() && !slices.Contains(runs, item.AgentRunID) {
			runs = append(runs, item.AgentRunID)
		}
	}

	return runs
}
