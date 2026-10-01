package dispatchconsoleresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
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

	baseRate, err := base.DecimalFromString(input.BaseRate, "baseRate")
	if err != nil {
		return nil, err
	}

	fuelSurcharge := decimal.Zero
	if input.FuelSurcharge != nil && *input.FuelSurcharge != "" {
		fuelSurcharge, err = base.DecimalFromString(*input.FuelSurcharge, "fuelSurcharge")
		if err != nil {
			return nil, err
		}
	}

	accessorials := make([]repositories.CarrierAccessorialInput, 0, len(input.Accessorials))
	for _, acc := range input.Accessorials {
		if acc == nil {
			continue
		}

		amount, accErr := base.DecimalFromString(acc.Amount, "accessorials.amount")
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
		ProNumber:                base.StringValue(input.ProNumber),
		ExternalDriverName:       base.StringValue(input.ExternalDriverName),
		ExternalDriverPhone:      base.StringValue(input.ExternalDriverPhone),
		ExternalTractorNumber:    base.StringValue(input.ExternalTractorNumber),
		ExternalTrailerNumber:    base.StringValue(input.ExternalTrailerNumber),
		Replace:                  base.BoolValue(input.Replace),
		OverrideInsuranceWarning: base.BoolValue(input.OverrideInsuranceWarning),
	}, nil
}

func dispatchCarrierEligibilityToModel(
	result *carrier.EligibilityResult,
) *gqlmodel.DispatchCarrierEligibility {
	findings := make([]*gqlmodel.DispatchCarrierEligibilityFinding, 0, len(result.Findings))
	for _, finding := range result.Findings {
		findings = append(findings, &gqlmodel.DispatchCarrierEligibilityFinding{
			Code:             finding.Code,
			Source:           finding.Source,
			Severity:         finding.Severity,
			Message:          finding.Message,
			RequiresOverride: finding.RequiresOverride,
		})
	}
	return &gqlmodel.DispatchCarrierEligibility{
		Blockers:   append([]string{}, result.Blockers...),
		Warnings:   append([]string{}, result.Warnings...),
		Advisories: append([]string{}, result.Advisories...),
		Findings:   findings,
	}
}
