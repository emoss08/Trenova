package jurisdictionruleresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/places/domain/jurisdictionrule"
	"github.com/emoss08/trenova/pkg/authctx"
)

func jurisdictionRuleFromInput(
	input *gqlmodel.JurisdictionRuleInput,
) (*jurisdictionrule.JurisdictionRule, error) {
	stateID, err := gqlid.RequiredID("stateId", input.StateID)
	if err != nil {
		return nil, err
	}
	baseFee, err := gqldecimal.ParseNullDecimalField("permitBaseFee", input.PermitBaseFee)
	if err != nil {
		return nil, err
	}
	perMileFee, err := gqldecimal.ParseNullDecimalField("permitPerMileFee", input.PermitPerMileFee)
	if err != nil {
		return nil, err
	}

	entity := &jurisdictionrule.JurisdictionRule{
		StateID:               stateID,
		Status:                input.Status,
		MaxWidthFeet:          input.MaxWidthFeet,
		MaxHeightFeet:         input.MaxHeightFeet,
		MaxLengthFeet:         input.MaxLengthFeet,
		MaxWeightPounds:       int64(input.MaxWeightPounds),
		SuperloadWidthFeet:    input.SuperloadWidthFeet,
		SuperloadWeightPounds: gqlscalar.Int64Ptr(input.SuperloadWeightPounds),
		DaylightOnly:          input.DaylightOnly,
		RushHourRestricted:    input.RushHourRestricted,
		WeekendRestricted:     input.WeekendRestricted,
		HolidayRestricted:     input.HolidayRestricted,
		PermitLeadTimeDays:    int16(input.PermitLeadTimeDays),
		PermitValidityDays:    int16(input.PermitValidityDays),
		PermitBaseFee:         baseFee,
		PermitPerMileFee:      perMileFee,
		SourceNote:            gqlscalar.StringValue(input.SourceNote),
		SourceURL:             gqlscalar.StringValue(input.SourceURL),
		EffectiveStartDate:    gqlscalar.Int64Ptr(input.EffectiveStartDate),
		EffectiveEndDate:      gqlscalar.Int64Ptr(input.EffectiveEndDate),
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return entity, nil
}

func jurisdictionRuleOverrideFromInput(
	input *gqlmodel.JurisdictionRuleOverrideInput,
	authCtx *authctx.AuthContext,
) (*jurisdictionrule.Override, error) {
	stateID, err := gqlid.RequiredID("stateId", input.StateID)
	if err != nil {
		return nil, err
	}

	entity := &jurisdictionrule.Override{
		OrganizationID:     authCtx.OrganizationID,
		BusinessUnitID:     authCtx.BusinessUnitID,
		StateID:            stateID,
		MaxWidthFeet:       input.MaxWidthFeet,
		MaxHeightFeet:      input.MaxHeightFeet,
		MaxLengthFeet:      input.MaxLengthFeet,
		MaxWeightPounds:    gqlscalar.Int64Ptr(input.MaxWeightPounds),
		PermitLeadTimeDays: intPtrAsInt16(input.PermitLeadTimeDays),
		DaylightOnly:       input.DaylightOnly,
		HolidayRestricted:  input.HolidayRestricted,
		Reason:             input.Reason,
	}
	if input.Version != nil {
		entity.Version = int64(*input.Version)
	}

	return entity, nil
}

func intPtrAsInt16(value *int) *int16 {
	if value == nil {
		return nil
	}
	converted := int16(*value)

	return &converted
}
