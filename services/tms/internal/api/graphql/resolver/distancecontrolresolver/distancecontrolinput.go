package distancecontrolresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlinput"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecontrol"
)

func distanceControlFromInput(
	existing *distancecontrol.DistanceControl,
	input *gqlmodel.UpdateDistanceControlInput,
) (*distancecontrol.DistanceControl, error) {
	parse := gqlinput.NewParser()
	entity := &distancecontrol.DistanceControl{
		ID:                       existing.ID,
		BusinessUnitID:           existing.BusinessUnitID,
		OrganizationID:           existing.OrganizationID,
		Version:                  int64(input.Version),
		CreatedAt:                existing.CreatedAt,
		UpdatedAt:                existing.UpdatedAt,
		SearchVector:             existing.SearchVector,
		BusinessUnit:             existing.BusinessUnit,
		Organization:             existing.Organization,
		StoreMileage:             input.StoreMileage,
		StoredDistanceUnits:      input.StoredDistanceUnits,
		PostalCodeFallbackToCity: input.PostalCodeFallbackToCity,
		AutoCreateStoredMileage:  input.AutoCreateStoredMileage,
		LoadedMoveDistanceProfileID: parse.ID(
			"loadedMoveDistanceProfileId",
			input.LoadedMoveDistanceProfileID,
		),
		EmptyMoveDistanceProfileID: parse.ID(
			"emptyMoveDistanceProfileId",
			input.EmptyMoveDistanceProfileID,
		),
		PayDistanceProfileID: parse.ID(
			"payDistanceProfileId",
			input.PayDistanceProfileID,
		),
		BillingDistanceProfileID: parse.ID(
			"billingDistanceProfileId",
			input.BillingDistanceProfileID,
		),
		FuelDistanceProfileID: parse.ID(
			"fuelDistanceProfileId",
			input.FuelDistanceProfileID,
		),
		EtaOutOfRouteDistanceProfileID: parse.ID(
			"etaOutOfRouteDistanceProfileId",
			input.EtaOutOfRouteDistanceProfileID,
		),
		DistanceCalculatorShortestDistanceProfileID: parse.ID(
			"distanceCalculatorShortestDistanceProfileId",
			input.DistanceCalculatorShortestDistanceProfileID,
		),
		DistanceCalculatorPracticalDistanceProfileID: parse.ID(
			"distanceCalculatorPracticalDistanceProfileId",
			input.DistanceCalculatorPracticalDistanceProfileID,
		),
		CaptureJurisdictionMiles: input.CaptureJurisdictionMiles,
	}
	if err := parse.Err(); err != nil {
		return nil, err
	}

	return entity, nil
}
