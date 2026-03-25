package shipmentservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

func createHazmatSegregationRule(
	controlRepo repositories.ShipmentControlRepository,
	commodityRepo repositories.CommodityRepository,
	hazmatRuleRepo repositories.HazmatSegregationRuleRepository,
) validationframework.TenantedRule[*shipment.Shipment] {
	return validationframework.NewTenantedRule[*shipment.Shipment]("hazmat_segregation").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *shipment.Shipment,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			control, err := controlRepo.Get(ctx, repositories.GetShipmentControlRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
			})
			if err != nil {
				multiErr.Add(
					"shipmentControl",
					errortypes.ErrInvalid,
					"Unable to load shipment control",
				)
				return nil
			}

			if !control.CheckHazmatSegregation || len(entity.Commodities) < 2 {
				return nil
			}

			commodityIDs := uniqueShipmentCommodityIDs(entity.Commodities)
			if len(commodityIDs) < 2 {
				return nil
			}

			commodities, err := commodityRepo.GetByIDs(ctx, repositories.GetCommoditiesByIDsRequest{
				TenantInfo: pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
				CommodityIDs: commodityIDs,
			})
			if err != nil {
				multiErr.Add(
					"commodities",
					errortypes.ErrInvalid,
					"Unable to load commodity hazmat data",
				)
				return nil
			}

			hazmatCommodities := mapHazmatCommodities(commodities)
			if len(hazmatCommodities) < 2 {
				return nil
			}

			rules, err := hazmatRuleRepo.ListActiveByTenant(ctx, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			})
			if err != nil {
				multiErr.Add(
					"commodities",
					errortypes.ErrInvalid,
					"Unable to load hazmat segregation rules",
				)
				return nil
			}

			for i := 0; i < len(entity.Commodities); i++ {
				leftShipmentCommodity := entity.Commodities[i]
				if leftShipmentCommodity == nil {
					continue
				}

				leftCommodity, ok := hazmatCommodities[leftShipmentCommodity.CommodityID]
				if !ok {
					continue
				}

				for j := i + 1; j < len(entity.Commodities); j++ {
					rightShipmentCommodity := entity.Commodities[j]
					if rightShipmentCommodity == nil {
						continue
					}

					rightCommodity, ok := hazmatCommodities[rightShipmentCommodity.CommodityID]
					if !ok {
						continue
					}

					matchedRule := findMatchingHazmatRule(rules, leftCommodity, rightCommodity)
					if matchedRule == nil {
						continue
					}

					multiErr.WithIndex("commodities", i).Add(
						"commodityId",
						errortypes.ErrInvalidOperation,
						fmt.Sprintf(
							"Violates hazmat segregation rule %q (%s) — conflicts with %q",
							matchedRule.Name,
							matchedRule.SegregationType,
							rightCommodity.Name,
						),
					)
					multiErr.WithIndex("commodities", j).Add(
						"commodityId",
						errortypes.ErrInvalidOperation,
						fmt.Sprintf(
							"Violates hazmat segregation rule %q (%s) — conflicts with %q",
							matchedRule.Name,
							matchedRule.SegregationType,
							leftCommodity.Name,
						),
					)
				}
			}

			return nil
		})
}

func uniqueShipmentCommodityIDs(items []*shipment.ShipmentCommodity) []pulid.ID {
	ids := make([]pulid.ID, 0, len(items))
	seen := make(map[pulid.ID]struct{}, len(items))

	for _, item := range items {
		if item == nil || item.CommodityID.IsNil() {
			continue
		}
		if _, ok := seen[item.CommodityID]; ok {
			continue
		}

		seen[item.CommodityID] = struct{}{}
		ids = append(ids, item.CommodityID)
	}

	return ids
}

func mapHazmatCommodities(items []*commodity.Commodity) map[pulid.ID]*commodity.Commodity {
	result := make(map[pulid.ID]*commodity.Commodity, len(items))

	for _, item := range items {
		if item == nil || item.HazardousMaterial == nil || item.HazardousMaterial.ID.IsNil() {
			continue
		}

		result[item.ID] = item
	}

	return result
}

func findMatchingHazmatRule(
	rules []*hazmatsegregationrule.HazmatSegregationRule,
	leftCommodity *commodity.Commodity,
	rightCommodity *commodity.Commodity,
) *hazmatsegregationrule.HazmatSegregationRule {
	for _, rule := range rules {
		if rule == nil {
			continue
		}

		if !matchesHazmatClasses(
			rule,
			leftCommodity.HazardousMaterial,
			rightCommodity.HazardousMaterial,
		) {
			continue
		}

		if !matchesHazmatMaterials(
			rule,
			leftCommodity.HazardousMaterial,
			rightCommodity.HazardousMaterial,
		) {
			continue
		}

		return rule
	}

	return nil
}

func matchesHazmatClasses(
	rule *hazmatsegregationrule.HazmatSegregationRule,
	left *hazardousmaterial.HazardousMaterial,
	right *hazardousmaterial.HazardousMaterial,
) bool {
	return (rule.ClassA == left.Class && rule.ClassB == right.Class) ||
		(rule.ClassA == right.Class && rule.ClassB == left.Class)
}

func matchesHazmatMaterials(
	rule *hazmatsegregationrule.HazmatSegregationRule,
	left *hazardousmaterial.HazardousMaterial,
	right *hazardousmaterial.HazardousMaterial,
) bool {
	if rule.HazmatAID == nil && rule.HazmatBID == nil {
		return true
	}

	return materialPairMatches(rule.HazmatAID, rule.HazmatBID, left.ID, right.ID) ||
		materialPairMatches(rule.HazmatAID, rule.HazmatBID, right.ID, left.ID)
}

func materialPairMatches(expectedA, expectedB *pulid.ID, actualA, actualB pulid.ID) bool {
	if expectedA != nil && *expectedA != actualA {
		return false
	}

	if expectedB != nil && *expectedB != actualB {
		return false
	}

	return true
}
