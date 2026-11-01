package rateresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/tariffmapping"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func rateAgreementFromInput(
	input *gqlmodel.RateAgreementInput,
	authCtx *authctx.AuthContext,
) (*rateagreement.RateAgreement, error) {
	entity := &rateagreement.RateAgreement{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
	}
	if err := applyRateAgreementInput(entity, input); err != nil {
		return nil, err
	}

	return entity, nil
}

func applyRateAgreementInput(
	entity *rateagreement.RateAgreement,
	input *gqlmodel.RateAgreementInput,
) error {
	customerID, err := gqlid.PulidPtrFromOptionalString(input.CustomerID)
	if err != nil {
		return err
	}
	carrierID, err := gqlid.PulidPtrFromOptionalString(input.CarrierID)
	if err != nil {
		return err
	}
	documentID, err := gqlid.PulidPtrFromOptionalString(input.DocumentID)
	if err != nil {
		return err
	}
	billToCustomerID, err := gqlid.PulidPtrFromOptionalString(input.BillToCustomerID)
	if err != nil {
		return err
	}
	marginFloor, err := gqldecimal.ParseNullDecimalField(
		"marginFloorPercent",
		input.MarginFloorPercent,
	)
	if err != nil {
		return err
	}
	maxPay, err := gqldecimal.ParseNullDecimalField(
		"maxPayPercentOfSell",
		input.MaxPayPercentOfSell,
	)
	if err != nil {
		return err
	}
	inputs, err := tariffmapping.TariffInputsFromModel("inputs", input.Inputs)
	if err != nil {
		return err
	}
	rules, err := rateAgreementRulesFromInput(input.Rules)
	if err != nil {
		return err
	}
	accessorials, err := rateAgreementAccessorialsFromInput(input.Accessorials)
	if err != nil {
		return err
	}
	fuelBinding, err := rateAgreementFuelBindingFromInput(input.FuelBinding)
	if err != nil {
		return err
	}

	entity.PartyType = rateagreement.PartyType(input.PartyType)
	entity.CustomerID = customerID
	entity.CarrierID = carrierID
	entity.Code = input.Code
	entity.Name = input.Name
	entity.Description = gqlscalar.StringValue(input.Description)
	entity.AgreementType = rateagreement.AgreementType(input.AgreementType)
	entity.ContractRef = gqlscalar.StringValue(input.ContractRef)
	entity.DocumentID = documentID
	entity.Priority = int16(gqlscalar.IntValue(input.Priority))
	entity.EffectiveFrom = int64(input.EffectiveFrom)
	entity.EffectiveTo = gqlscalar.Int64Ptr(input.EffectiveTo)
	entity.AutoRenew = gqlscalar.BoolValue(input.AutoRenew)
	entity.RenewalNoticeDays = int16(gqlscalar.IntValue(input.RenewalNoticeDays))
	entity.Currency = input.Currency
	entity.Inputs = inputs
	entity.BillToCustomerID = billToCustomerID
	entity.MarginFloorPercent = marginFloor
	entity.MaxPayPercentOfSell = maxPay
	entity.Rules = rules
	entity.Accessorials = accessorials
	entity.FuelBinding = fuelBinding
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return nil
}

func rateAgreementRulesFromInput(
	inputs []*gqlmodel.RateAgreementRuleInput,
) ([]*rateagreement.RateAgreementRule, error) {
	rules := make([]*rateagreement.RateAgreementRule, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		rule, err := rateAgreementRuleFromInput(input)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

type ruleIDLists struct {
	serviceTypes  []pulid.ID
	shipmentTypes []pulid.ID
	tractorTypes  []pulid.ID
	trailerTypes  []pulid.ID
	commodities   []pulid.ID
}

func ruleIDListsFromInput(input *gqlmodel.RateAgreementRuleInput) (*ruleIDLists, error) {
	lists := new(ruleIDLists)
	var err error
	if lists.serviceTypes, err = idList(
		"serviceTypeIds",
		input.ServiceTypeIds,
	); err != nil {
		return nil, err
	}
	if lists.shipmentTypes, err = idList(
		"shipmentTypeIds",
		input.ShipmentTypeIds,
	); err != nil {
		return nil, err
	}
	if lists.tractorTypes, err = idList(
		"tractorTypeIds",
		input.TractorTypeIds,
	); err != nil {
		return nil, err
	}
	if lists.trailerTypes, err = idList(
		"trailerTypeIds",
		input.TrailerTypeIds,
	); err != nil {
		return nil, err
	}
	if lists.commodities, err = idList(
		"commodityIds",
		input.CommodityIds,
	); err != nil {
		return nil, err
	}

	return lists, nil
}

type ruleBounds struct {
	minWeight   decimal.NullDecimal
	maxWeight   decimal.NullDecimal
	minDistance decimal.NullDecimal
	maxDistance decimal.NullDecimal
}

func ruleBoundsFromInput(input *gqlmodel.RateAgreementRuleInput) (*ruleBounds, error) {
	bounds := new(ruleBounds)
	var err error
	if bounds.minWeight, err = gqldecimal.ParseNullDecimalField(
		"minWeight",
		input.MinWeight,
	); err != nil {
		return nil, err
	}
	if bounds.maxWeight, err = gqldecimal.ParseNullDecimalField(
		"maxWeight",
		input.MaxWeight,
	); err != nil {
		return nil, err
	}
	if bounds.minDistance, err = gqldecimal.ParseNullDecimalField(
		"minDistance",
		input.MinDistance,
	); err != nil {
		return nil, err
	}
	if bounds.maxDistance, err = gqldecimal.ParseNullDecimalField(
		"maxDistance",
		input.MaxDistance,
	); err != nil {
		return nil, err
	}

	return bounds, nil
}

func rateAgreementRuleFromInput(
	input *gqlmodel.RateAgreementRuleInput,
) (*rateagreement.RateAgreementRule, error) {
	ruleID, err := gqlid.ParseOptionalIDField("id", input.ID)
	if err != nil {
		return nil, err
	}
	templateID, err := gqlid.ParseIDField("formulaTemplateId", input.FormulaTemplateID)
	if err != nil {
		return nil, err
	}
	ids, err := ruleIDListsFromInput(input)
	if err != nil {
		return nil, err
	}
	bounds, err := ruleBoundsFromInput(input)
	if err != nil {
		return nil, err
	}
	values, err := tariffmapping.TariffInputsFromModel("inputs", input.Inputs)
	if err != nil {
		return nil, err
	}

	status := rateagreement.RuleStatus(gqlscalar.StringValue(input.Status))
	if status == "" {
		status = rateagreement.RuleStatusActive
	}
	direction := rateagreement.Direction(gqlscalar.StringValue(input.Direction))
	if direction == "" {
		direction = rateagreement.DirectionDirectional
	}

	return &rateagreement.RateAgreementRule{
		ID:                      ruleID,
		Label:                   gqlscalar.StringValue(input.Label),
		Status:                  status,
		OriginScopeType:         rategeo.ScopeType(input.OriginScopeType),
		OriginScopeValue:        gqlscalar.StringValue(input.OriginScopeValue),
		OriginCity:              gqlscalar.StringValue(input.OriginCity),
		DestinationScopeType:    rategeo.ScopeType(input.DestinationScopeType),
		DestinationScopeValue:   gqlscalar.StringValue(input.DestinationScopeValue),
		DestinationCity:         gqlscalar.StringValue(input.DestinationCity),
		Direction:               direction,
		OriginRadiusMeters:      input.OriginRadiusMeters,
		DestinationRadiusMeters: input.DestinationRadiusMeters,
		OriginLatitude:          input.OriginLatitude,
		OriginLongitude:         input.OriginLongitude,
		DestinationLatitude:     input.DestinationLatitude,
		DestinationLongitude:    input.DestinationLongitude,
		ServiceTypeIDs:          ids.serviceTypes,
		ShipmentTypeIDs:         ids.shipmentTypes,
		TractorTypeIDs:          ids.tractorTypes,
		TrailerTypeIDs:          ids.trailerTypes,
		CommodityIDs:            ids.commodities,
		FreightClasses:          stringsAs[commodity.FreightClass](input.FreightClasses),
		ServiceModels:           stringsAs[modeprofile.ServiceModel](input.ServiceModels),
		EquipmentClasses:        stringsAs[modeprofile.EquipmentClass](input.EquipmentClasses),
		MinWeight:               bounds.minWeight,
		MaxWeight:               bounds.maxWeight,
		MinDistance:             bounds.minDistance,
		MaxDistance:             bounds.maxDistance,
		MinStops:                intPtrAsInt16(input.MinStops),
		MaxStops:                intPtrAsInt16(input.MaxStops),
		DaysOfWeek:              int16(gqlscalar.IntValue(input.DaysOfWeek)),
		HazmatOnly:              gqlscalar.BoolValue(input.HazmatOnly),
		TempControlOnly:         gqlscalar.BoolValue(input.TempControlOnly),
		FormulaTemplateID:       templateID,
		Inputs:                  values,
		Currency:                gqlscalar.StringValue(input.Currency),
		Priority:                int16(gqlscalar.IntValue(input.Priority)),
		EffectiveFrom:           int64(input.EffectiveFrom),
		EffectiveTo:             gqlscalar.Int64Ptr(input.EffectiveTo),
	}, nil
}

func rateAgreementAccessorialsFromInput(
	inputs []*gqlmodel.RateAgreementAccessorialInput,
) ([]*rateagreement.RateAgreementAccessorial, error) {
	accessorials := make([]*rateagreement.RateAgreementAccessorial, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		accessorial, err := rateAgreementAccessorialFromInput(input)
		if err != nil {
			return nil, err
		}
		accessorials = append(accessorials, accessorial)
	}

	return accessorials, nil
}

func rateAgreementAccessorialFromInput(
	input *gqlmodel.RateAgreementAccessorialInput,
) (*rateagreement.RateAgreementAccessorial, error) {
	chargeID, err := gqlid.ParseIDField("accessorialChargeId", input.AccessorialChargeID)
	if err != nil {
		return nil, err
	}
	amount, err := gqldecimal.ParseDecimalField("amount", input.Amount, true)
	if err != nil {
		return nil, err
	}
	maxAmount, err := gqldecimal.ParseNullDecimalField("maxAmount", input.MaxAmount)
	if err != nil {
		return nil, err
	}
	serviceTypes, err := idList("serviceTypeIds", input.ServiceTypeIds)
	if err != nil {
		return nil, err
	}
	shipmentTypes, err := idList("shipmentTypeIds", input.ShipmentTypeIds)
	if err != nil {
		return nil, err
	}

	return &rateagreement.RateAgreementAccessorial{
		AccessorialChargeID: chargeID,
		Method:              accessorialcharge.Method(input.Method),
		RateUnit:            accessorialcharge.RateUnit(gqlscalar.StringValue(input.RateUnit)),
		Amount:              amount,
		Waived:              gqlscalar.BoolValue(input.Waived),
		AutoApply:           gqlscalar.BoolValue(input.AutoApply),
		Condition:           gqlscalar.StringValue(input.Condition),
		FreeUnits:           intPtrAsInt16(input.FreeUnits),
		MaxAmount:           maxAmount,
		ServiceTypeIDs:      serviceTypes,
		ShipmentTypeIDs:     shipmentTypes,
		EffectiveFrom:       gqlscalar.Int64Ptr(input.EffectiveFrom),
		EffectiveTo:         gqlscalar.Int64Ptr(input.EffectiveTo),
	}, nil
}

func rateAgreementFuelBindingFromInput(
	input *gqlmodel.RateAgreementFuelBindingInput,
) (*rateagreement.RateAgreementFuelBinding, error) {
	if input == nil {
		return nil, nil //nolint:nilnil // an agreement without fuel terms binds no program
	}
	programID, err := gqlid.ParseIDField("fuelSurchargeProgramId", input.FuelSurchargeProgramID)
	if err != nil {
		return nil, err
	}
	peg, err := gqldecimal.ParseNullDecimalField("pegPriceOverride", input.PegPriceOverride)
	if err != nil {
		return nil, err
	}
	increment, err := gqldecimal.ParseNullDecimalField(
		"incrementRateOverride",
		input.IncrementRateOverride,
	)
	if err != nil {
		return nil, err
	}
	capAmount, err := gqldecimal.ParseNullDecimalField("capAmount", input.CapAmount)
	if err != nil {
		return nil, err
	}

	return &rateagreement.RateAgreementFuelBinding{
		FuelSurchargeProgramID: programID,
		Waived:                 gqlscalar.BoolValue(input.Waived),
		PegPriceOverride:       peg,
		IncrementRateOverride:  increment,
		CapAmount:              capAmount,
	}, nil
}

func idList(field string, raw []string) ([]pulid.ID, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	return gqlid.ParseIDListField(field, raw)
}

func stringsAs[T ~string](values []string) []T {
	if len(values) == 0 {
		return nil
	}
	out := make([]T, 0, len(values))
	for _, value := range values {
		out = append(out, T(value))
	}

	return out
}

func intPtrAsInt16(value *int) *int16 {
	if value == nil {
		return nil
	}
	converted := int16(*value)

	return &converted
}
