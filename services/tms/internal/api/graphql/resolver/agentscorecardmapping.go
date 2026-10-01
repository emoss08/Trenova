package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

// toGQLAgentScorecard shapes the counted record for the wire.
//
// The trust ladder rides along on the same object rather than as a field
// resolver: it comes from one query the service already made, and a lazy
// field would turn one request into two for a panel that always shows both.
func toGQLAgentScorecard(result *services.AgentScorecardResult) *gqlmodel.AgentScorecard {
	card := result.Scorecard

	out := &gqlmodel.AgentScorecard{
		AgentDefinitionID:     card.AgentDefinitionID,
		Window:                card.Window,
		Since:                 int(card.Since),
		Runs:                  card.Runs,
		RunsFailed:            card.RunsFailed,
		Exceptions:            card.Exceptions,
		Proposals:             card.Proposals,
		Approved:              card.Approved,
		Modified:              card.Modified,
		Rejected:              card.Rejected,
		Pending:               card.Pending,
		Executed:              card.Executed,
		ExecutionFailures:     card.ExecutionFailures,
		AutoExecuted:          card.AutoExecuted,
		ApprovalRate:          card.ApprovalRate(),
		InputTokens:           card.InputTokens,
		OutputTokens:          card.OutputTokens,
		CostUsd:               card.CostUSD.String(),
		EstimatedMinutesSaved: card.EstimatedMinutesSaved,
		ByTool:                make([]*agent.ToolOutcomeCount, 0, len(card.ByTool)),
		Trend:                 make([]*agent.ScorecardPoint, 0, len(card.Trend)),
		ToolTrust:             make([]*gqlmodel.AgentToolTrust, 0, len(result.ToolTrust)),
	}

	for index := range card.ByTool {
		out.ByTool = append(out.ByTool, &card.ByTool[index])
	}
	for index := range card.Trend {
		out.Trend = append(out.Trend, &card.Trend[index])
	}
	for _, trust := range result.ToolTrust {
		out.ToolTrust = append(out.ToolTrust, toGQLAgentToolTrust(trust))
	}

	return out
}

func toGQLAgentToolTrust(trust *agent.ToolTrust) *gqlmodel.AgentToolTrust {
	out := &gqlmodel.AgentToolTrust{
		ToolName:          trust.ToolName,
		Streak:            trust.Streak,
		Approvals:         trust.Approvals,
		Modifications:     trust.Modifications,
		Rejections:        trust.Rejections,
		ExecutionFailures: trust.ExecutionFailures,
		LastDecisionAt:    optionalUnix(trust.LastDecisionAt),
		PromotedAt:        optionalUnix(trust.PromotedAt),
		DemotedAt:         optionalUnix(trust.DemotedAt),
	}
	// An empty earned tier means the tool's tier was chosen by a person, not
	// granted by the ledger, which is a different thing from "granted the
	// lowest tier" and has to stay absent rather than become one.
	if trust.EarnedTier != "" {
		tier := trust.EarnedTier
		out.EarnedTier = &tier
	}

	return out
}

// optionalUnix carries a nullable instant to the wire without turning an
// absent timestamp into the epoch.
func optionalUnix(at *int64) *int {
	if at == nil {
		return nil
	}
	value := int(*at)

	return &value
}
