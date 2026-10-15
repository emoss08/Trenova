package aianalyticsservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ratePtr(rate float64) *float64 { return &rate }

func TestBuildSuggestions_TotalsEachAgentAndRanksTheMostSuggestedFirst(t *testing.T) {
	t.Parallel()

	billing := pulid.ID("agdef_billing")
	service := pulid.ID("agdef_service")
	data := &readings{
		suggestionsRead: true,
		suggestions: []aianalyticsports.SuggestionRow{
			{
				AgentDefinitionID: billing,
				AgentName:         "Billing",
				SuggestionCounts:  aianalyticsports.SuggestionCounts{Switched: 1, Kept: 2, PassedOver: 1},
			},
			{
				AgentDefinitionID: service,
				AgentName:         "Customer service",
				SuggestionCounts: aianalyticsports.SuggestionCounts{
					Switched:     3,
					SentThrough:  2,
					AutoSwitched: 1,
					PassedOver:   2,
				},
			},
		},
	}

	stats := buildSuggestions(data)

	require.NotNil(t, stats)
	assert.Equal(t, &aianalyticsports.SuggestionStats{
		SuggestionCounts: aianalyticsports.SuggestionCounts{
			Switched:     4,
			SentThrough:  2,
			AutoSwitched: 1,
			Kept:         2,
			PassedOver:   3,
		},
		Suggested: 12,
		TakenRate: ratePtr(7.0 / 12.0),
		Agents: []aianalyticsports.AgentSuggestionStats{
			{
				AgentDefinitionID: service,
				Name:              "Customer service",
				SuggestionCounts: aianalyticsports.SuggestionCounts{
					Switched:     3,
					SentThrough:  2,
					AutoSwitched: 1,
					PassedOver:   2,
				},
				Suggested: 8,
				TakenRate: ratePtr(6.0 / 8.0),
			},
			{
				AgentDefinitionID: billing,
				Name:              "Billing",
				SuggestionCounts:  aianalyticsports.SuggestionCounts{Switched: 1, Kept: 2, PassedOver: 1},
				Suggested:         4,
				TakenRate:         ratePtr(0.25),
			},
		},
	}, stats)
}

func TestBuildSuggestions_LeavesTheSectionOutWhenNothingWasSuggested(t *testing.T) {
	t.Parallel()

	assert.Nil(t, buildSuggestions(&readings{suggestionsRead: true}))
	assert.Nil(t, buildSuggestions(&readings{
		suggestionsRead: true,
		suggestions:     []aianalyticsports.SuggestionRow{{AgentDefinitionID: "agdef_x"}},
	}))
}

func TestBuildSuggestions_LeavesTheSectionOutWhenItWasNotRead(t *testing.T) {
	t.Parallel()

	assert.Nil(t, buildSuggestions(&readings{
		suggestions: []aianalyticsports.SuggestionRow{{
			AgentDefinitionID: "agdef_x",
			SuggestionCounts:  aianalyticsports.SuggestionCounts{Switched: 1},
		}},
	}), "a view narrowed to one task has no suggestions to show")
}
