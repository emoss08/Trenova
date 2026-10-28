package dispatchconsoleresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

func assignMoveToCarrierRequestFromInput(
	input *gqlmodel.DispatchAssignMoveToCarrierInput,
	tenant pagination.TenantInfo,
) (*carrierassignmentports.AssignMoveToCarrierRequest, error) {
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

	baseRate, err := gqldecimal.DecimalFromString(input.BaseRate, "baseRate")
	if err != nil {
		return nil, err
	}

	fuelSurcharge := decimal.Zero
	if input.FuelSurcharge != nil && *input.FuelSurcharge != "" {
		fuelSurcharge, err = gqldecimal.DecimalFromString(*input.FuelSurcharge, "fuelSurcharge")
		if err != nil {
			return nil, err
		}
	}

	accessorials := make(
		[]carrierassignmentports.CarrierAccessorialInput,
		0,
		len(input.Accessorials),
	)
	for _, acc := range input.Accessorials {
		if acc == nil {
			continue
		}

		amount, accErr := gqldecimal.DecimalFromString(acc.Amount, "accessorials.amount")
		if accErr != nil {
			return nil, accErr
		}

		item := carrierassignmentports.CarrierAccessorialInput{
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

	return &carrierassignmentports.AssignMoveToCarrierRequest{
		TenantInfo:               tenant,
		ShipmentMoveID:           moveID,
		CarrierID:                carrierID,
		RateMethod:               input.RateMethod,
		BaseRate:                 baseRate,
		FuelSurcharge:            fuelSurcharge,
		Accessorials:             accessorials,
		ProNumber:                gqlscalar.StringValue(input.ProNumber),
		ExternalDriverName:       gqlscalar.StringValue(input.ExternalDriverName),
		ExternalDriverPhone:      gqlscalar.StringValue(input.ExternalDriverPhone),
		ExternalTractorNumber:    gqlscalar.StringValue(input.ExternalTractorNumber),
		ExternalTrailerNumber:    gqlscalar.StringValue(input.ExternalTrailerNumber),
		Replace:                  gqlscalar.BoolValue(input.Replace),
		OverrideInsuranceWarning: gqlscalar.BoolValue(input.OverrideInsuranceWarning),
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
