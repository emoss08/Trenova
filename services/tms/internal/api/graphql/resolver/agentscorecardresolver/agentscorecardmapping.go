package agentscorecardresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/agenttrustmapping"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentevaluationports"
)

// toGQLAgentScorecard shapes the counted record for the wire.
//
// The trust ladder rides along on the same object rather than as a field
// resolver: it comes from one query the service already made, and a lazy
// field would turn one request into two for a panel that always shows both.
func toGQLAgentScorecard(
	result *agentevaluationports.AgentScorecardResult,
) *gqlmodel.AgentScorecard {
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
		ByTool:                make([]*agentevaluation.ToolOutcomeCount, 0, len(card.ByTool)),
		ToolVerdicts:          make([]*agentevaluation.ToolVerdictCount, 0, len(card.ToolVerdicts)),
		Trend:                 make([]*agentevaluation.ScorecardPoint, 0, len(card.Trend)),
		ToolTrust:             agenttrustmapping.ToolTrusts(result.ToolTrust),
	}

	for index := range card.ByTool {
		out.ByTool = append(out.ByTool, &card.ByTool[index])
	}
	for index := range card.ToolVerdicts {
		out.ToolVerdicts = append(out.ToolVerdicts, &card.ToolVerdicts[index])
	}
	for index := range card.Trend {
		out.Trend = append(out.Trend, &card.Trend[index])
	}

	return out
}
