package agentplan

import "github.com/emoss08/trenova/shared/pulid"

func MostTargeted(targets []pulid.ID) pulid.ID {
	counts := make(map[pulid.ID]int, len(targets))
	best, bestCount := pulid.Nil, 0
	for _, target := range targets {
		if target.IsNil() {
			continue
		}
		counts[target]++
		if counts[target] > bestCount {
			best, bestCount = target, counts[target]
		}
	}

	return best
}
