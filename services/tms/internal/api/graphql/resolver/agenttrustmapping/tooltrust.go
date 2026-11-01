package agenttrustmapping

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
)

func ToolTrust(trust *agenttooltrust.ToolTrust) *gqlmodel.AgentToolTrust {
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
	if trust.EarnedTier != "" {
		tier := trust.EarnedTier
		out.EarnedTier = &tier
	}

	return out
}

func ToolTrusts(rows []*agenttooltrust.ToolTrust) []*gqlmodel.AgentToolTrust {
	out := make([]*gqlmodel.AgentToolTrust, 0, len(rows))
	for _, row := range rows {
		out = append(out, ToolTrust(row))
	}

	return out
}

func optionalUnix(at *int64) *int {
	if at == nil {
		return nil
	}
	value := int(*at)

	return &value
}
