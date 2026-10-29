package shipmenteffects

import (
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
)

func mergeCommodityDetails(
	existing *commodity.Commodity,
	loaded *commodity.Commodity,
) *commodity.Commodity {
	if existing == nil {
		return loaded
	}

	merged := *loaded

	if existing.LinearFeetPerUnit != nil {
		merged.LinearFeetPerUnit = existing.LinearFeetPerUnit
	}

	if existing.HazardousMaterial != nil {
		merged.HazardousMaterial = existing.HazardousMaterial
	}

	if !existing.HazardousMaterialID.IsNil() {
		merged.HazardousMaterialID = existing.HazardousMaterialID
	}

	return &merged
}
