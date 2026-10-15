package aianalyticsservice

import (
	"cmp"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
)

func buildSuggestions(data *readings) *aianalyticsports.SuggestionStats {
	if data == nil || !data.suggestionsRead {
		return nil
	}

	stats := &aianalyticsports.SuggestionStats{
		Agents: make([]aianalyticsports.AgentSuggestionStats, 0, len(data.suggestions)),
	}
	for _, row := range data.suggestions {
		suggested := row.Suggested()
		if suggested == 0 {
			continue
		}
		stats.Add(row.SuggestionCounts)
		stats.Agents = append(stats.Agents, aianalyticsports.AgentSuggestionStats{
			AgentDefinitionID: row.AgentDefinitionID,
			Name:              row.AgentName,
			SuggestionCounts:  row.SuggestionCounts,
			Suggested:         suggested,
			TakenRate:         takenRate(row.SuggestionCounts),
		})
	}
	stats.Suggested = stats.SuggestionCounts.Suggested()
	if stats.Suggested == 0 {
		return nil
	}
	stats.TakenRate = takenRate(stats.SuggestionCounts)

	slices.SortFunc(stats.Agents, func(a, b aianalyticsports.AgentSuggestionStats) int {
		if a.Suggested != b.Suggested {
			return cmp.Compare(b.Suggested, a.Suggested)
		}
		if byName := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); byName != 0 {
			return byName
		}

		return strings.Compare(a.AgentDefinitionID.String(), b.AgentDefinitionID.String())
	})

	return stats
}

func takenRate(counts aianalyticsports.SuggestionCounts) *float64 {
	suggested := counts.Suggested()
	if suggested == 0 {
		return nil
	}
	rate := float64(counts.Taken()) / float64(suggested)

	return &rate
}
