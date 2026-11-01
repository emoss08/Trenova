package rateresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/tariffmapping"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratematrixservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
)

func rateAgreementRuleToModel(
	entity *rateagreement.RateAgreementRule,
) (*gqlmodel.RateAgreementRule, error) {
	inputs, err := tariffmapping.TariffInputsToModel(entity.Inputs)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.RateAgreementRule{
		ID:                      entity.ID.String(),
		BusinessUnitID:          entity.BusinessUnitID.String(),
		OrganizationID:          entity.OrganizationID.String(),
		RateAgreementID:         entity.RateAgreementID.String(),
		Label:                   entity.Label,
		Status:                  string(entity.Status),
		OriginScopeType:         string(entity.OriginScopeType),
		OriginScopeValue:        entity.OriginScopeValue,
		OriginCity:              entity.OriginCity,
		DestinationScopeType:    string(entity.DestinationScopeType),
		DestinationScopeValue:   entity.DestinationScopeValue,
		DestinationCity:         entity.DestinationCity,
		Direction:               string(entity.Direction),
		LaneKey:                 entity.LaneKey,
		SpecificityScore:        int(entity.SpecificityScore),
		OriginRadiusMeters:      entity.OriginRadiusMeters,
		DestinationRadiusMeters: entity.DestinationRadiusMeters,
		OriginLatitude:          entity.OriginLatitude,
		OriginLongitude:         entity.OriginLongitude,
		DestinationLatitude:     entity.DestinationLatitude,
		DestinationLongitude:    entity.DestinationLongitude,
		ServiceTypeIds:          gqlid.PulidsToStrings(entity.ServiceTypeIDs),
		ShipmentTypeIds:         gqlid.PulidsToStrings(entity.ShipmentTypeIDs),
		TractorTypeIds:          gqlid.PulidsToStrings(entity.TractorTypeIDs),
		TrailerTypeIds:          gqlid.PulidsToStrings(entity.TrailerTypeIDs),
		CommodityIds:            gqlid.PulidsToStrings(entity.CommodityIDs),
		FreightClasses:          stringsOf(entity.FreightClasses),
		ServiceModels:           stringsOf(entity.ServiceModels),
		EquipmentClasses:        stringsOf(entity.EquipmentClasses),
		MinWeight:               gqldecimal.NullDecimalToStringPtr(entity.MinWeight),
		MaxWeight:               gqldecimal.NullDecimalToStringPtr(entity.MaxWeight),
		MinDistance:             gqldecimal.NullDecimalToStringPtr(entity.MinDistance),
		MaxDistance:             gqldecimal.NullDecimalToStringPtr(entity.MaxDistance),
		MinStops:                int16PtrAsInt(entity.MinStops),
		MaxStops:                int16PtrAsInt(entity.MaxStops),
		DaysOfWeek:              int(entity.DaysOfWeek),
		HazmatOnly:              entity.HazmatOnly,
		TempControlOnly:         entity.TempControlOnly,
		FormulaTemplateID:       entity.FormulaTemplateID.String(),
		Inputs:                  inputs,
		Currency:                entity.Currency,
		Priority:                int(entity.Priority),
		EffectiveFrom:           int(entity.EffectiveFrom),
		EffectiveTo:             gqlscalar.Int64PtrToIntPtr(entity.EffectiveTo),
		SupersedesRuleID:        gqlid.IDPtrFromPulidPtr(entity.SupersedesRuleID),
		Version:                 int(entity.Version),
		CreatedAt:               int(entity.CreatedAt),
		UpdatedAt:               int(entity.UpdatedAt),
	}, nil
}

func rateAgreementRulesToModel(
	rules []*rateagreement.RateAgreementRule,
) ([]*gqlmodel.RateAgreementRule, error) {
	out := make([]*gqlmodel.RateAgreementRule, 0, len(rules))
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		model, err := rateAgreementRuleToModel(rule)
		if err != nil {
			return nil, err
		}
		out = append(out, model)
	}

	return out, nil
}

func rateAgreementVersionToModel(
	entity *rateagreement.RateAgreementVersion,
) (*gqlmodel.RateAgreementVersion, error) {
	accessorialTerms, err := optionalJSONMap(len(entity.AccessorialTerms), entity.AccessorialTerms)
	if err != nil {
		return nil, err
	}
	fuelTerms, err := gqlscalar.OptionalJSON(entity.FuelTerms)
	if err != nil {
		return nil, err
	}
	changeSummary, err := optionalJSONMap(len(entity.ChangeSummary), entity.ChangeSummary)
	if err != nil {
		return nil, err
	}
	var accessorialNames map[string]any
	if len(entity.AccessorialNames) > 0 {
		accessorialNames = make(map[string]any, len(entity.AccessorialNames))
		for chargeID, code := range entity.AccessorialNames {
			accessorialNames[chargeID] = code
		}
	}

	return &gqlmodel.RateAgreementVersion{
		ID:                     entity.ID.String(),
		RateAgreementID:        entity.RateAgreementID.String(),
		VersionNumber:          int(entity.VersionNumber),
		EffectiveFrom:          int(entity.EffectiveFrom),
		EffectiveTo:            gqlscalar.Int64PtrToIntPtr(entity.EffectiveTo),
		Code:                   entity.Code,
		Name:                   entity.Name,
		Priority:               int(entity.Priority),
		AgreementEffectiveFrom: int(entity.AgreementEffectiveFrom),
		AgreementEffectiveTo:   gqlscalar.Int64PtrToIntPtr(entity.AgreementEffectiveTo),
		AutoRenew:              entity.AutoRenew,
		RenewalNoticeDays:      int(entity.RenewalNoticeDays),
		BillToCustomerID:       gqlid.IDPtrFromPulidPtr(entity.BillToCustomerID),
		AccessorialTerms:       accessorialTerms,
		FuelTerms:              fuelTerms,
		AccessorialNames:       accessorialNames,
		ChangeMessage:          entity.ChangeMessage,
		ChangeSummary:          changeSummary,
		CreatedByID:            gqlid.IDPtr(entity.CreatedByID),
		CreatedAt:              int(entity.CreatedAt),
	}, nil
}

func optionalJSONMap(size int, value any) (map[string]any, error) {
	if size == 0 {
		return nil, nil //nolint:nilnil // an empty map maps to a null GraphQL field
	}

	return gqlscalar.OptionalJSON(value)
}

func rateAgreementLanePageToModel(
	page *rateagreementports.RateAgreementLanePage,
) (*gqlmodel.RateAgreementLanePage, error) {
	results, err := rateAgreementRulesToModel(page.Lanes)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.RateAgreementLanePage{
		Results:      results,
		Count:        page.Count,
		CountExact:   page.CountExact,
		Retired:      page.Retired,
		RetiredExact: page.RetiredExact,
		Limit:        page.Limit,
		Offset:       page.Offset,
	}, nil
}

func rateAgreementLaneSummaryToModel(
	summary *rateagreementports.RateAgreementLaneSummary,
) *gqlmodel.RateAgreementLaneSummary {
	groups := make([]*gqlmodel.RateAgreementLanePricingGroup, 0, len(summary.Groups))
	for _, group := range summary.Groups {
		if group == nil {
			continue
		}
		groups = append(groups, &gqlmodel.RateAgreementLanePricingGroup{
			FormulaTemplateID:   gqlid.IDPtrFromPulidPtr(group.FormulaTemplateID),
			FormulaTemplateName: group.FormulaTemplateName,
			Currency:            group.Currency,
			Lanes:               group.Lanes,
			MinRate:             gqldecimal.NullDecimalToStringPtr(group.MinRate),
			MaxRate:             gqldecimal.NullDecimalToStringPtr(group.MaxRate),
		})
	}

	return &gqlmodel.RateAgreementLaneSummary{Groups: groups, Count: summary.Count}
}

func rateIncreasePlanToModel(
	plan *rateagreementservice.RateIncreasePlan,
) (*gqlmodel.RateIncreasePlan, error) {
	lines := make([]*gqlmodel.RateIncreaseLine, 0, len(plan.Lines))
	for _, line := range plan.Lines {
		if line == nil {
			continue
		}
		before, err := tariffmapping.TariffInputsToModel(line.Before)
		if err != nil {
			return nil, err
		}
		after, err := tariffmapping.TariffInputsToModel(line.After)
		if err != nil {
			return nil, err
		}
		changed := line.Changed
		if changed == nil {
			changed = []string{}
		}
		lines = append(lines, &gqlmodel.RateIncreaseLine{
			RateAgreementID: line.RateAgreementID.String(),
			AgreementCode:   line.AgreementCode,
			AgreementName:   line.AgreementName,
			RuleID:          line.RuleID.String(),
			LaneKey:         line.LaneKey,
			Label:           line.Label,
			Before:          before,
			After:           after,
			Changed:         changed,
		})
	}

	return &gqlmodel.RateIncreasePlan{
		EffectiveFrom:  int(plan.EffectiveFrom),
		AgreementCount: plan.AgreementCount,
		Lines:          lines,
		SkippedNoMoney: plan.SkippedNoMoney,
	}, nil
}

func rateMatrixReviewersToModel(
	reviewers []ratematrixservice.Reviewer,
) []*gqlmodel.RateMatrixReviewer {
	out := make([]*gqlmodel.RateMatrixReviewer, 0, len(reviewers))
	for _, reviewer := range reviewers {
		out = append(out, &gqlmodel.RateMatrixReviewer{
			ID:            reviewer.ID.String(),
			Name:          reviewer.Name,
			Username:      reviewer.Username,
			ProfilePicURL: gqlscalar.EmptyToNil(reviewer.ProfilePicURL),
		})
	}

	return out
}

func stringsOf[T ~string](values []T) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}

	return out
}

func int16PtrAsInt(value *int16) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)

	return &converted
}
