package dispatchcontrolresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlinput"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
)

func scoringWeightsFromInput(
	parse *gqlinput.Parser,
	raw map[string]any,
) dispatchcontrol.ScoringWeights {
	weights := make(dispatchcontrol.ScoringWeights, len(raw))
	for factor, value := range raw {
		if value == nil {
			continue
		}
		weights[factor] = parse.Number("scoringWeights."+factor, value)
	}

	return weights
}
