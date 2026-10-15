package aianalyticsresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestSuggestionsModel_CarriesEveryCountAndAgent(t *testing.T) {
	t.Parallel()

	rate := 0.5
	agentRate := 0.75
	counts := aianalyticsports.SuggestionCounts{
		Switched:     1,
		SentThrough:  2,
		AutoSwitched: 3,
		Kept:         4,
		PassedOver:   5,
	}

	model := suggestionsModel(&aianalyticsports.SuggestionStats{
		SuggestionCounts: counts,
		Suggested:        15,
		TakenRate:        &rate,
		Agents: []aianalyticsports.AgentSuggestionStats{{
			AgentDefinitionID: pulid.ID("agdef_billing"),
			Name:              "Billing",
			SuggestionCounts:  counts,
			Suggested:         15,
			TakenRate:         &agentRate,
		}},
	})

	assert.Equal(t, &gqlmodel.AIAnalyticsSuggestions{
		Suggested:    15,
		Switched:     1,
		SentThrough:  2,
		AutoSwitched: 3,
		Kept:         4,
		PassedOver:   5,
		TakenRate:    &rate,
		Agents: []*gqlmodel.AIAnalyticsAgentSuggestions{{
			AgentDefinitionID: "agdef_billing",
			Name:              "Billing",
			Suggested:         15,
			Switched:          1,
			SentThrough:       2,
			AutoSwitched:      3,
			Kept:              4,
			PassedOver:        5,
			TakenRate:         &agentRate,
		}},
	}, model)
	assert.Nil(t, suggestionsModel(nil))
}
