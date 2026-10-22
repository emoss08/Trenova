package deskbench

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecommendations_NameTheSettingsTheConfigurationUses(t *testing.T) {
	t.Parallel()

	gate := &GateResult{
		Configuration: mixedSelection{
			RouteChat:       "claude-haiku-5-5@low+advisor",
			RouteUnattended: "claude-haiku-5-5@medium",
		},
		CodeMode: []CodeModeDecision{
			{
				Cell:              "claude-haiku-5-5@low",
				Model:             CellModel{Provider: "AnthropicMessages", Model: "claude-haiku-5-5"},
				Recommended:       true,
				WritesRecommended: true,
			},
			{
				Cell:  "gpt-6-luna@medium",
				Model: CellModel{Provider: "OpenAIResponses", Model: "gpt-6-luna"},
			},
		},
	}

	recommendations := gate.Recommendations()
	assert.Contains(t, recommendations, SettingRecommendation{
		Provider: "claude-haiku-5-5", Setting: "Reasoning effort", Value: "low", Route: RouteChat,
		Reason: "the effort rule chose it for this route",
	})
	assert.Contains(t, recommendations, SettingRecommendation{
		Provider: "claude-haiku-5-5", Setting: "Advisor", Value: "claude-opus-5-5",
		Route: RouteChat, Reason: "the advisor rule held on this route",
	})
	assert.Equal(t, []string{"on"}, settingsOf(recommendations, "Scripts"),
		"only the model the rule held for")
	assert.Equal(t, []string{"on"}, settingsOf(recommendations, "Scripts may propose writes"))

	var b strings.Builder
	renderRecommendations(&b, recommendations)
	assert.Contains(t, b.String(), "the gate changes no setting")
	assert.Contains(t, b.String(), "claude-haiku-5-5 (AnthropicMessages): Scripts on")
}

func TestRecommendations_WritesNeedTheirOwnAdversarialPairs(t *testing.T) {
	t.Parallel()

	gate := &GateResult{CodeMode: []CodeModeDecision{{
		Cell:        "claude-haiku-5-5@low",
		Model:       CellModel{Provider: "AnthropicMessages", Model: "claude-haiku-5-5"},
		Recommended: true,
	}}}
	assert.Equal(t, []string{"on"}, settingsOf(gate.Recommendations(), "Scripts"))
	assert.Empty(t, settingsOf(gate.Recommendations(), "Scripts may propose writes"))
}
