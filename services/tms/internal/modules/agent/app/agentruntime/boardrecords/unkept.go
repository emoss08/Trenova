package boardrecords

import (
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
)

func UnkeptFigures(results []string, words string) []string {
	unkept := make(map[string]bool, 4)
	for _, content := range results {
		payload, ok := toolresult.Payload(content)
		fields, isRecord := payload.(map[string]any)
		metric, _ := fields["metric"].(string)
		listed, _ := fields["figures"].([]any)
		if !ok || !isRecord || metric == "" {
			continue
		}
		for _, entry := range listed {
			figure, _ := entry.(map[string]any)
			name, _ := figure["name"].(string)
			if name == "" {
				continue
			}
			value, present := figure["value"]
			unkept[name+":"+metric] = !present || value == nil
		}
	}
	placed := make([]string, 0, 1)
	for _, key := range slices.Sorted(maps.Keys(unkept)) {
		if unkept[key] && strings.Contains(words, "{"+key+"}") {
			placed = append(placed, "{"+key+"}")
		}
	}

	return placed
}
