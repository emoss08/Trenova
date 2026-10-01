package shipmentresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/services/costingservice"
)

func shipmentProfitabilityEstimateToModel(
	estimate *costingservice.ShipmentProfitabilityEstimate,
) *gqlmodel.ShipmentProfitabilityEstimate {
	if estimate == nil {
		return nil
	}

	model := &gqlmodel.ShipmentProfitabilityEstimate{
		ShipmentID:      estimate.ShipmentID.String(),
		LoadedMiles:     estimate.LoadedMiles,
		DeadheadMiles:   estimate.DeadheadMiles,
		TotalMiles:      estimate.TotalMiles,
		EstimatedCost:   estimate.EstimatedCost.String(),
		Profit:          estimate.Profit.String(),
		MarginPercent:   base.NullDecimalToStringPtr(estimate.MarginPercent),
		BreakEvenRpm:    base.NullDecimalToStringPtr(estimate.BreakEvenRPM),
		MissingDistance: estimate.MissingDistance,
	}

	if estimate.Profile != nil {
		model.CostPerMile = estimate.Profile.TotalCPM.String()
		model.TargetMarginPercent = base.NullDecimalToStringPtr(
			estimate.Profile.TargetMarginPercent,
		)
	}

	return model
}
