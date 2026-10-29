package agentresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentactivityports"
)

type ActivityVisibility struct {
	Proposals  bool
	Exceptions bool
}

func ActivitySummaryToModel(
	summary *agentactivityports.AgentActivitySummary,
	visible ActivityVisibility,
) *gqlmodel.AgentActivitySummary {
	out := &gqlmodel.AgentActivitySummary{
		Since:              int(summary.Since),
		Runs:               summary.Runs,
		RunsFailed:         summary.RunsFailed,
		RunsWorking:        summary.RunsWorking,
		RunsAwaiting:       summary.RunsAwaiting,
		DecisionWindowDays: summary.DecisionWindowDays,
	}
	if visible.Proposals {
		pending := summary.PendingProposals
		decided := summary.Decided
		out.PendingProposals = &pending
		out.Decided = &decided
		out.ApprovedAsProposed = summary.ApprovedAsProposed
		if summary.OldestPendingAt != nil {
			oldest := int(*summary.OldestPendingAt)
			out.OldestPendingAt = &oldest
		}
	}
	if visible.Exceptions {
		open := summary.OpenExceptions
		out.OpenExceptions = &open
	}

	return out
}
