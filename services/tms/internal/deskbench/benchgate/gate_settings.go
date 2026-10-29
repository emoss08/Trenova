package benchgate

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/deskbench/benchcell"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
)

const batchWriteTag = "batch-write"

// SettingRecommendation is a provider setting in AI control the gate's figures
// argue for. The gate never changes a setting: an organization turns each one
// on or off itself, and real use decides whether it works.
type SettingRecommendation struct {
	// Provider names the chat provider the setting belongs to, by the model
	// the cell ran (and its protocol, for code mode).
	Provider string              `json:"provider"`
	Setting  string              `json:"setting"`
	Value    string              `json:"value"`
	Route    benchscenario.Route `json:"route,omitempty"`
	Reason   string              `json:"reason"`
}

func routeSettings(configuration mixedSelection) []SettingRecommendation {
	recommendations := make([]SettingRecommendation, 0, 2*len(benchscenario.AllRoutes()))
	for _, route := range benchscenario.AllRoutes() {
		cell := configuration[route]
		parsed, err := benchcell.ParseRef(cell)
		if cell == "" || err != nil {
			continue
		}
		if parsed.Effort != "" {
			recommendations = append(recommendations, SettingRecommendation{
				Provider: parsed.Name,
				Setting:  "Reasoning effort",
				Value:    strings.ToLower(string(parsed.Effort)),
				Route:    route,
				Reason:   "the effort rule chose it for this route",
			})
		}
		if parsed.Advisor != nil {
			recommendations = append(recommendations, SettingRecommendation{
				Provider: parsed.Name,
				Setting:  "Advisor",
				Value:    parsed.Advisor.Model,
				Route:    route,
				Reason:   "the advisor rule held on this route",
			})
		}
	}

	return recommendations
}

func codeModeSettings(model CellModel, writes bool, reason string) []SettingRecommendation {
	provider := model.Model + " (" + model.Provider + ")"
	recommendations := []SettingRecommendation{{
		Provider: provider,
		Setting:  "Scripts",
		Value:    "on",
		Reason:   reason,
	}}
	if writes {
		recommendations = append(recommendations, SettingRecommendation{
			Provider: provider,
			Setting:  "Scripts may propose writes",
			Value:    "on",
			Reason:   "no unauthorized effect on the batch-write adversarial pairs",
		})
	}

	return recommendations
}

// Recommendations are the provider settings run 1's figures argue for, to be
// re-checked in run 2 before anyone acts on them.
func (r *GateResult) Recommendations() []SettingRecommendation {
	recommendations := routeSettings(r.Configuration)
	seen := make(map[CellModel]bool, len(r.CodeMode))
	for idx := range r.CodeMode {
		decision := &r.CodeMode[idx]
		if !decision.Recommended || seen[decision.Model] {
			continue
		}
		seen[decision.Model] = true
		recommendations = append(recommendations, codeModeSettings(decision.Model,
			decision.WritesRecommended, "the code-mode rule held on "+decision.Cell)...)
	}

	return recommendations
}

// Recommendations are the provider settings that held in run 2.
func (c *GateConfirmation) Recommendations() []SettingRecommendation {
	recommendations := routeSettings(c.Configuration)
	for idx := range c.CodeMode {
		recheck := &c.CodeMode[idx]
		if !recheck.Holds {
			continue
		}
		recommendations = append(recommendations, codeModeSettings(recheck.Model,
			recheck.Run2.WritesRecommended, "the code-mode rule held again on "+recheck.Cell)...)
	}

	return recommendations
}

func renderRecommendations(b *strings.Builder, recommendations []SettingRecommendation) {
	b.WriteString("\n## Recommended settings\n\nWhat the figures argue for, set on each chat " +
		"provider in AI Control. They are recommendations: the gate changes no setting, and an " +
		"organization decides from its own use.\n\n")
	if len(recommendations) == 0 {
		b.WriteString("No setting is recommended.\n")
		return
	}
	for idx := range recommendations {
		recommendation := &recommendations[idx]
		fmt.Fprintf(b, "- %s: %s %s", recommendation.Provider, recommendation.Setting,
			recommendation.Value)
		if recommendation.Route != "" {
			fmt.Fprintf(b, " (route %s)", recommendation.Route)
		}
		fmt.Fprintf(b, ". %s.\n", recommendation.Reason)
	}
}
