package base

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
)

func QuickFilterSpecsFromGraphQL(
	inputs []*gqlmodel.ShipmentQuickFilterInput,
) []shipment.QuickFilterSpec {
	if len(inputs) == 0 {
		return nil
	}

	specs := make([]shipment.QuickFilterSpec, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		specs = append(specs, shipment.QuickFilterSpec{
			Filter:             input.Filter,
			Hour:               input.Hour,
			WindowStartMinutes: input.WindowStartMinutes,
			WindowEndMinutes:   input.WindowEndMinutes,
		})
	}

	return specs
}
