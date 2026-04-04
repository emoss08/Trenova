package shipmentservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type hazmatConflict struct {
	leftIndex      int
	rightIndex     int
	leftCommodity  *commodity.Commodity
	rightCommodity *commodity.Commodity
	rule           *hazmatsegregationrule.HazmatSegregationRule
}

type hazmatCommodityCandidate struct {
	index       int
	commodityID pulid.ID
}

func (s *service) evaluateHazmatSegregationRequest(
	ctx context.Context,
	control *tenant.ShipmentControl,
	req *repositories.CheckHazmatSegregationRequest,
) ([]hazmatConflict, error) {
	if !control.CheckHazmatSegregation {
		return nil, nil
	}

	commodities, err := s.commodityRepo.GetByIDs(ctx, repositories.GetCommoditiesByIDsRequest{
		TenantInfo:   req.TenantInfo,
		CommodityIDs: req.CommodityIDs,
	})
	if err != nil {
		return nil, err
	}

	hazmatCommodities := mapHazmatCommodities(commodities)
	if len(hazmatCommodities) < 2 {
		return nil, nil
	}

	rules, err := s.hazmatRuleRepo.ListActiveByTenant(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	return evaluateHazmatConflicts(
		hazmatCandidatesFromCommodityIDs(req.CommodityIDs),
		hazmatCommodities,
		rules,
	), nil
}

func addHazmatConflictsToMultiError(multiErr *errortypes.MultiError, conflicts []hazmatConflict) {
	for _, conflict := range conflicts {
		multiErr.WithIndex("commodities", conflict.leftIndex).Add(
			"commodityId",
			errortypes.ErrInvalidOperation,
			fmt.Sprintf(
				"Violates hazmat segregation rule %q (%s) — conflicts with %q",
				conflict.rule.Name,
				conflict.rule.SegregationType,
				conflict.rightCommodity.Name,
			),
		)
		multiErr.WithIndex("commodities", conflict.rightIndex).Add(
			"commodityId",
			errortypes.ErrInvalidOperation,
			fmt.Sprintf(
				"Violates hazmat segregation rule %q (%s) — conflicts with %q",
				conflict.rule.Name,
				conflict.rule.SegregationType,
				conflict.leftCommodity.Name,
			),
		)
	}
}

func evaluateHazmatConflicts(
	candidates []hazmatCommodityCandidate,
	hazmatCommodities map[pulid.ID]*commodity.Commodity,
	rules []*hazmatsegregationrule.HazmatSegregationRule,
) []hazmatConflict {
	conflicts := make([]hazmatConflict, 0)

	for i := 0; i < len(candidates); i++ {
		leftCandidate := candidates[i]
		leftCommodity, ok := hazmatCommodities[leftCandidate.commodityID]
		if !ok {
			continue
		}

		for j := i + 1; j < len(candidates); j++ {
			rightCandidate := candidates[j]
			rightCommodity, ok := hazmatCommodities[rightCandidate.commodityID]
			if !ok {
				continue
			}

			matchedRule := findMatchingHazmatRule(rules, leftCommodity, rightCommodity)
			if matchedRule == nil {
				continue
			}

			conflicts = append(conflicts, hazmatConflict{
				leftIndex:      leftCandidate.index,
				rightIndex:     rightCandidate.index,
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
) []hazmatConflict {
	return evaluateHazmatConflicts(hazmatCandidatesFromShipmentCommodities(commodities), hazmatCommodities, rules)
}

func hazmatCandidatesFromCommodityIDs(commodityIDs []pulid.ID) []hazmatCommodityCandidate {
	candidates := make([]hazmatCommodityCandidate, 0, len(commodityIDs))
	for index, commodityID := range commodityIDs {
		if commodityID.IsNil() {
			continue
		}

		candidates = append(candidates, hazmatCommodityCandidate{
			index:       index,
			commodityID: commodityID,
		})
	}

	return candidates
}

func hazmatCandidatesFromShipmentCommodities(
	commodities []*shipment.ShipmentCommodity,
) []hazmatCommodityCandidate {
	candidates := make([]hazmatCommodityCandidate, 0, len(commodities))
	for index, shipmentCommodity := range commodities {
		if shipmentCommodity == nil || shipmentCommodity.CommodityID.IsNil() {
			continue
		}

		candidates = append(candidates, hazmatCommodityCandidate{
			index:       index,
			commodityID: shipmentCommodity.CommodityID,
		})
	}

	return candidates
}
