package shipmentvalidator

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type HazmatConflict struct {
	leftIndex      int
	rightIndex     int
	leftCommodity  *commodity.Commodity
	rightCommodity *commodity.Commodity
	rule           *hazmatsegregationrule.HazmatSegregationRule
}

type HazmatCommodityCandidate struct {
	Index       int
	CommodityID pulid.ID
}

func AddHazmatConflictsToMultiError(multiErr *errortypes.MultiError, conflicts []HazmatConflict) {
	for _, conflict := range conflicts {
		multiErr.WithIndex("commodities", conflict.leftIndex).Add(
			"commodityId",
			errortypes.ErrInvalidOperation,
			"Violates hazmat segregation rule \"{0}\" ({1}) — conflicts with \"{2}\"", conflict.rule.Name, conflict.rule.SegregationType, conflict.rightCommodity.Name,
		)
		multiErr.WithIndex("commodities", conflict.rightIndex).Add(
			"commodityId",
			errortypes.ErrInvalidOperation,
			"Violates hazmat segregation rule \"{0}\" ({1}) — conflicts with \"{2}\"", conflict.rule.Name, conflict.rule.SegregationType, conflict.leftCommodity.Name,
		)
	}
}

//nolint:govet // existing scoped variable reuse is local and behavior-preserving
func EvaluateHazmatConflicts(
	candidates []HazmatCommodityCandidate,
	hazmatCommodities map[pulid.ID]*commodity.Commodity,
	rules []*hazmatsegregationrule.HazmatSegregationRule,
) []HazmatConflict {
	conflicts := make([]HazmatConflict, 0)

	for i := range candidates {
		leftCandidate := candidates[i]
		leftCommodity, ok := hazmatCommodities[leftCandidate.CommodityID]
		if !ok {
			continue
		}

		for j := i + 1; j < len(candidates); j++ {
			rightCandidate := candidates[j]
			rightCommodity, ok := hazmatCommodities[rightCandidate.CommodityID]
			if !ok {
				continue
			}

			matchedRule := FindMatchingHazmatRule(rules, leftCommodity, rightCommodity)
			if matchedRule == nil {
				continue
			}

			conflicts = append(conflicts, HazmatConflict{
				leftIndex:      leftCandidate.Index,
				rightIndex:     rightCandidate.Index,
				leftCommodity:  leftCommodity,
				rightCommodity: rightCommodity,
				rule:           matchedRule,
			})
		}
	}

	return conflicts
}

func evaluateShipmentHazmatConflicts(
	commodities []*shipment.ShipmentCommodity,
	hazmatCommodities map[pulid.ID]*commodity.Commodity,
	rules []*hazmatsegregationrule.HazmatSegregationRule,
) []HazmatConflict {
	return EvaluateHazmatConflicts(
		hazmatCandidatesFromShipmentCommodities(commodities),
		hazmatCommodities,
		rules,
	)
}

func hazmatCandidatesFromShipmentCommodities(
	commodities []*shipment.ShipmentCommodity,
) []HazmatCommodityCandidate {
	candidates := make([]HazmatCommodityCandidate, 0, len(commodities))
	for index, shipmentCommodity := range commodities {
		if shipmentCommodity == nil || shipmentCommodity.CommodityID.IsNil() {
			continue
		}

		candidates = append(candidates, HazmatCommodityCandidate{
			Index:       index,
			CommodityID: shipmentCommodity.CommodityID,
		})
	}

	return candidates
}
