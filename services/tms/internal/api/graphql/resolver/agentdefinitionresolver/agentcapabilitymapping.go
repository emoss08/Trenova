package agentdefinitionresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// capabilitiesUpdate reads the page's changes into the service's request.
func capabilitiesUpdate(
	agentID string,
	input *gqlmodel.UpdateAgentCapabilitiesInput,
) (*agentports.UpdateAgentCapabilitiesRequest, error) {
	id, err := pulid.MustParse(agentID)
	if err != nil {
		return nil, err
	}

	req := &agentports.UpdateAgentCapabilitiesRequest{
		AgentID:               id,
		Version:               int64(input.Version),
		Enabled:               input.Enabled,
		DailyRequestLimit:     input.DailyRequestLimit,
		ClearMonthlyBudget:    input.ClearMonthlyBudget != nil && *input.ClearMonthlyBudget,
		MaxChangeItems:        input.MaxChangeItems,
		BusinessHoursOnly:     input.BusinessHoursOnly,
		BusinessHoursStart:    input.BusinessHoursStart,
		BusinessHoursEnd:      input.BusinessHoursEnd,
		BusinessHoursTimezone: input.BusinessHoursTimezone,
	}
	if input.MonthlyBudgetUsd != nil {
		budget, pErr := decimal.NewFromString(*input.MonthlyBudgetUsd)
		if pErr != nil {
			return nil, errortypes.NewValidationError(
				"monthlyBudgetUsd", errortypes.ErrInvalid, "Budget must be an amount in dollars",
			)
		}
		req.MonthlyBudgetUSD = &budget
	}
	for _, tool := range input.Tools {
		if tool == nil {
			continue
		}
		req.Tools = append(req.Tools, agentports.AgentCapabilityToolChange{
			Key:  tool.Key,
			Mode: tool.Mode,
		})
	}
	if len(input.DelegateTopics) > 0 {
		req.DelegateTopics = make(map[pulid.ID]string, len(input.DelegateTopics))
		for _, topic := range input.DelegateTopics {
			if topic == nil {
				continue
			}
			delegate, pErr := pulid.MustParse(topic.AgentID)
			if pErr != nil {
				return nil, pErr
			}
			req.DelegateTopics[delegate] = topic.Topic
		}
	}

	return req, nil
}
