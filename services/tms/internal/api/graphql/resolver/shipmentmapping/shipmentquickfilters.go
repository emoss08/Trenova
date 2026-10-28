package shipmentmapping

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/quickfilter"
)

func QuickFilterSpecsFromGraphQL(
	inputs []*gqlmodel.ShipmentQuickFilterInput,
) []quickfilter.QuickFilterSpec {
	if len(inputs) == 0 {
		return nil
	}

	specs := make([]quickfilter.QuickFilterSpec, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		specs = append(specs, quickfilter.QuickFilterSpec{
			Filter:             input.Filter,
			Hour:               input.Hour,
			WindowStartMinutes: input.WindowStartMinutes,
			WindowEndMinutes:   input.WindowEndMinutes,
		})
	}

	return specs
}
