package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

func assignMoveToCarrierRequestFromInput(
	input *gqlmodel.DispatchAssignMoveToCarrierInput,
	tenant pagination.TenantInfo,
) (*repositories.AssignMoveToCarrierRequest, error) {
	moveID, err := parseDispatchID(
		input.MoveID,
		"moveId",
		"Move ID is not a valid identifier",
	)
	if err != nil {
		return nil, err
	}

	carrierID, err := parseDispatchID(
		input.CarrierID,
		"carrierId",
		"Carrier ID is not a valid identifier",
	)
	if err != nil {
		return nil, err
	}

	baseRate, err := decimalFromString(input.BaseRate, "baseRate")
	if err != nil {
		return nil, err
	}

	fuelSurcharge := decimal.Zero
	if input.FuelSurcharge != nil && *input.FuelSurcharge != "" {
		fuelSurcharge, err = decimalFromString(*input.FuelSurcharge, "fuelSurcharge")
		if err != nil {
			return nil, err
		}
	}

	accessorials := make([]repositories.CarrierAccessorialInput, 0, len(input.Accessorials))
	for _, acc := range input.Accessorials {
		if acc == nil {
			continue
		}

		amount, accErr := decimalFromString(acc.Amount, "accessorials.amount")
		if accErr != nil {
			return nil, accErr
		}

		item := repositories.CarrierAccessorialInput{
			Description: acc.Description,
			Amount:      amount,
		}
		if acc.AccessorialChargeID != nil && *acc.AccessorialChargeID != "" {
			chargeID, idErr := parseDispatchID(
				*acc.AccessorialChargeID,
				"accessorials.accessorialChargeId",
				"Accessorial charge ID is not a valid identifier",
			)
			if idErr != nil {
				return nil, idErr
			}
			item.AccessorialChargeID = &chargeID
		}
		accessorials = append(accessorials, item)
	}

	return &repositories.AssignMoveToCarrierRequest{
		TenantInfo:               tenant,
		ShipmentMoveID:           moveID,
		CarrierID:                carrierID,
		RateMethod:               input.RateMethod,
		BaseRate:                 baseRate,
		FuelSurcharge:            fuelSurcharge,
		Accessorials:             accessorials,
		ProNumber:                stringValue(input.ProNumber),
		ExternalDriverName:       stringValue(input.ExternalDriverName),
		ExternalDriverPhone:      stringValue(input.ExternalDriverPhone),
		ExternalTractorNumber:    stringValue(input.ExternalTractorNumber),
		ExternalTrailerNumber:    stringValue(input.ExternalTrailerNumber),
		Replace:                  boolValue(input.Replace),
		OverrideInsuranceWarning: boolValue(input.OverrideInsuranceWarning),
	}, nil
}
