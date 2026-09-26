package agentruntime

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/shared/pulid"
)

// baselineCall is one write whose target is pinned as it is decided.
type baselineCall struct {
	req        *serviceports.RunRequest
	tool       serviceports.AgentTool
	call       serviceports.ToolCall
	proposalID pulid.ID
	persist    bool
	// fileRefused files the write even when its preview says it would be
	// refused: an earlier step of the same turn changes its record first.
	fileRefused bool
}

// fileBaseline pins the record a write would change and, where the tool can
// say, what the write would do to it, both read from one snapshot. The
// preview is kept beside the proposal, keyed by the id minted for it here,
// never on the action: the action travels through every later activity of
// the turn, and a preview there would grow the workflow's history with each
// proposal. An evaluation keeps nothing. Nothing here can fail the call: a
// baseline that cannot be taken is simply not kept. The caller reads the
// preview it returns for a refusal before filing the write.
func (s *Service) fileBaseline(
	ctx context.Context,
	b *baselineCall,
) (*serviceports.ProposalTarget, *serviceports.ProposalBaselineResult) {
	if s.previews == nil {
		return s.snapshotTarget(ctx, b.req, b.tool, b.call), nil
	}

	result := s.previews.Baseline(ctx, &serviceports.ProposalBaselineRequest{
		ProposalID: b.proposalID,
		Tool:       b.tool,
		Params: serviceports.ToolExecuteParams{
			OrganizationID: b.req.Actor.OrganizationID,
			BusinessUnitID: b.req.Actor.BusinessUnitID,
			Actor:          b.req.Actor,
			IdempotencyKey: b.call.ID,
			RunID:          b.req.RunID,
			Params:         b.call.Arguments,
		},
		Persist:     b.persist,
		FileRefused: b.fileRefused,
	})
	if result == nil {
		return s.snapshotTarget(ctx, b.req, b.tool, b.call), nil
	}

	return result.Target, result
}

// dependsOnEarlierStep reports whether the write is to a record an earlier
// unexecuted write of this turn also changes. The two are filed as steps of
// one plan, and what the later one would be refused over may be exactly what
// the earlier one changes, so the later one is filed as it stands and the
// plan's preview says it depends on the earlier step.
func dependsOnEarlierStep(
	tool serviceports.AgentTool,
	call serviceports.ToolCall,
	proposedSoFar []serviceports.PendingAction,
) bool {
	targeted, ok := tool.(serviceports.TargetedTool)
	if !ok {
		return false
	}
	target, ok := targeted.Target(call.Arguments)
	if !ok {
		return false
	}
	for i := range proposedSoFar {
		earlier := &proposedSoFar[i]
		if earlier.Executed || earlier.Simulated || earlier.Target == nil {
			continue
		}
		if earlier.Target.Resource == target.Resource && earlier.Target.ID == target.ID {
			return true
		}
	}

	return false
}

// refusedBeforeFiling is what the model is told when a write it proposed
// would be refused as it stands: each reason and the parameter it is about,
// and what to do instead of proposing the same call again.
func refusedBeforeFiling(toolName string, refusal *agent.PreviewWarning) toolOutcome {
	var b strings.Builder
	fmt.Fprintf(&b, "Tool %q was not proposed, because it would be refused as it stands:\n",
		toolName)
	for _, line := range refusal.ReasonLines() {
		b.WriteString("- ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("Ask the person for the value(s) needed, or look one up with a query tool " +
		"where a valid one can be found, then propose again with the corrected call. " +
		"Do not propose the same call again.")

	return refusedOutcome(
		aitrace.OutcomeInvalid,
		strings.TrimPrefix(strings.TrimSpace(refusal.Message), toolpreview.WouldFailPrefix),
		"%s",
		b.String(),
	)
}
