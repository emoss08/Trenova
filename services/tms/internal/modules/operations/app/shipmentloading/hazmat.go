package shipmentloading

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentvalidator"
	"github.com/emoss08/trenova/shared/pulid"
)

func hazmatCandidatesFromCommodityIDs(
	commodityIDs []pulid.ID,
) []shipmentvalidator.HazmatCommodityCandidate {
	candidates := make([]shipmentvalidator.HazmatCommodityCandidate, 0, len(commodityIDs))
	for index, commodityID := range commodityIDs {
		if commodityID.IsNil() {
			continue
		}

		candidates = append(candidates, shipmentvalidator.HazmatCommodityCandidate{
			Index:       index,
			CommodityID: commodityID,
		})
	}

	return candidates
}
