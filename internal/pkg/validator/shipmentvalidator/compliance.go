// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package shipmentvalidator

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator/framework"
	"github.com/shopspring/decimal"
)

// StandardWeightLimits contains standard weight limits for trucks in the US
var StandardWeightLimits = struct {
	MaxGrossWeight decimal.Decimal // Max gross weight in pounds
	MaxAxleWeight  decimal.Decimal // Max single axle weight in pounds
	MaxTandemAxle  decimal.Decimal // Max tandem axle weight in pounds
}{
	MaxGrossWeight: decimal.NewFromInt(80000), // 80,000 lbs for interstate
	MaxAxleWeight:  decimal.NewFromInt(20000), // 20,000 lbs for single axle
	MaxTandemAxle:  decimal.NewFromInt(34000), // 34,000 lbs for tandem axle
}

// AddWeightComplianceRules adds weight compliance validation rules to the validation engine
func AddWeightComplianceRules(engine *framework.ValidationEngine, shp *shipment.Shipment) {
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageCompliance,
			framework.ValidationPriorityHigh,
			func(_ context.Context, multiErr *errors.MultiError) error {
				// Calculate total weight from commodities if available
				var totalWeight int64
				if shp.Commodities != nil {
					for _, comm := range shp.Commodities {
						totalWeight += comm.Weight
					}
				} else if shp.Weight != nil {
					totalWeight = *shp.Weight
				}

				// Validate gross vehicle weight
				if totalWeight > 0 &&
					decimal.NewFromInt(totalWeight).
						GreaterThan(StandardWeightLimits.MaxGrossWeight) {
					multiErr.Add(
						"weight",
						errors.ErrInvalid,
						fmt.Sprintf(
							"Total weight exceeds maximum allowed (80,000 lbs) for interstate transport. Current: %d lbs",
							totalWeight,
						),
					)
				}
				return nil
			},
		),
	)
}

// AddHOSComplianceRules adds Hours of Service compliance validation rules to the validation engine
func AddHOSComplianceRules(engine *framework.ValidationEngine, _ *shipment.Shipment) {
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageCompliance,
			framework.ValidationPriorityHigh,
			func(_ context.Context, _ *errors.MultiError) error {
				// This would check if the planned moves could potentially cause HOS violations
				// For example, by calculating total driving time and comparing to available driver hours
				// This is a placeholder for future implementation
				return nil
			},
		),
	)
}

// AddHazmatComplianceRules adds hazardous materials compliance validation rules to the validation engine
func AddHazmatComplianceRules(engine *framework.ValidationEngine, shp *shipment.Shipment) {
	engine.AddRule(
		framework.NewValidationRule(
			framework.ValidationStageCompliance,
			framework.ValidationPriorityHigh,
			func(_ context.Context, multiErr *errors.MultiError) error {
				// Check if shipment has hazmat commodities
				hasHazmat := false
				if shp.Commodities != nil {
					for _, comm := range shp.Commodities {
						if comm.Commodity != nil && comm.Commodity.HazardousMaterialID != nil {
							hasHazmat = true
							break
						}
					}
				}

				// If it has hazmat, validate required documentation
				if hasHazmat {
					// Currently, there are no specific hazmat documentation fields in the Shipment struct
					// This is a placeholder for future implementation when those fields are added
					multiErr.Add(
						"hazmat",
						errors.ErrInvalid,
						"Hazardous materials documentation is required for shipments containing hazardous materials",
					)
				}

				return nil
			},
		),
	)
}
