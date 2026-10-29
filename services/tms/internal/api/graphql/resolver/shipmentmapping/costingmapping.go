package shipmentmapping

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/modules/billing/app/costingservice"
)

func ProfitabilityEstimateToModel(
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
		MarginPercent:   gqldecimal.NullDecimalToStringPtr(estimate.MarginPercent),
		BreakEvenRpm:    gqldecimal.NullDecimalToStringPtr(estimate.BreakEvenRPM),
		MissingDistance: estimate.MissingDistance,
	}

	if estimate.Profile != nil {
		model.CostPerMile = estimate.Profile.TotalCPM.String()
		model.TargetMarginPercent = gqldecimal.NullDecimalToStringPtr(
			estimate.Profile.TargetMarginPercent,
		)
	}

	return model
}
