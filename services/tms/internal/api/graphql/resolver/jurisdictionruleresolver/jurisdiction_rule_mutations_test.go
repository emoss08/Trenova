package jurisdictionruleresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/places/domain/jurisdictionrule"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJurisdictionRuleMutations_CheckTheRESTRoutesPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("jr_").String()
	stateID := pulid.MustNew("st_").String()
	rule := gqlmodel.JurisdictionRuleInput{
		StateID: stateID,
		Status:  jurisdictionrule.Status("Active"),
	}
	override := gqlmodel.JurisdictionRuleOverrideInput{
		StateID: stateID,
		Reason:  "Insurer caps width",
	}
	verify := gqlmodel.VerifyJurisdictionRuleInput{
		VerificationState: jurisdictionrule.VerificationVerified,
	}
	rules, overrides := permission.ResourceJurisdictionRule,
		permission.ResourceJurisdictionRuleOverride

	calls := []struct {
		name      string
		resource  permission.Resource
		operation permission.Operation
		run       func(ctx context.Context, m *MutationResolver) error
	}{
		{
			"createJurisdictionRule",
			rules,
			permission.OpCreate,
			func(ctx context.Context, m *MutationResolver) error {
				_, err := m.CreateJurisdictionRule(ctx, rule)
				return err
			},
		},
		{
			"updateJurisdictionRule",
			rules,
			permission.OpUpdate,
			func(ctx context.Context, m *MutationResolver) error {
				_, err := m.UpdateJurisdictionRule(ctx, id, rule)
				return err
			},
		},
		{
			"verifyJurisdictionRule",
			rules,
			permission.OpApprove,
			func(ctx context.Context, m *MutationResolver) error {
				_, err := m.VerifyJurisdictionRule(ctx, id, verify)
				return err
			},
		},
		{
			"createJurisdictionRuleOverride",
			overrides,
			permission.OpCreate,
			func(ctx context.Context, m *MutationResolver) error {
				_, err := m.CreateJurisdictionRuleOverride(ctx, override)
				return err
			},
		},
		{
			"updateJurisdictionRuleOverride",
			overrides,
			permission.OpUpdate,
			func(ctx context.Context, m *MutationResolver) error {
				_, err := m.UpdateJurisdictionRuleOverride(ctx, id, override)
				return err
			},
		},
		{
			"deleteJurisdictionRuleOverride",
			overrides,
			permission.OpDelete,
			func(ctx context.Context, m *MutationResolver) error {
				_, err := m.DeleteJurisdictionRuleOverride(ctx, id)
				return err
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, &MutationResolver{deps})

			resolvertest.RequireRefused(t, err, engine, call.resource, call.operation)
		})
	}
}

func TestJurisdictionRuleOverrideFromInput_TakesTheTenantFromTheSession(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	stateID := pulid.MustNew("st_")
	width, leadTime := 8.0, 3

	entity, err := jurisdictionRuleOverrideFromInput(&gqlmodel.JurisdictionRuleOverrideInput{
		StateID:            stateID.String(),
		MaxWidthFeet:       &width,
		PermitLeadTimeDays: &leadTime,
		Reason:             "Our insurer caps width at eight feet",
	}, authCtx)
	require.NoError(t, err)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, stateID, entity.StateID)
	assert.Equal(t, &width, entity.MaxWidthFeet)
	assert.Equal(t, int16(3), *entity.PermitLeadTimeDays)
	assert.Nil(t, entity.MaxWeightPounds)
	assert.Nil(t, entity.DaylightOnly)
}

func TestJurisdictionRuleFromInput_MapsTheLimitsAndLeavesVerificationToTheService(t *testing.T) {
	t.Parallel()

	fee := "30.00"
	superload := 250000
	entity, err := jurisdictionRuleFromInput(&gqlmodel.JurisdictionRuleInput{
		StateID:               pulid.MustNew("st_").String(),
		Status:                jurisdictionrule.Status("Active"),
		MaxWidthFeet:          8.5,
		MaxWeightPounds:       80000,
		SuperloadWeightPounds: &superload,
		PermitLeadTimeDays:    1,
		PermitValidityDays:    5,
		PermitBaseFee:         &fee,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(80000), entity.MaxWeightPounds)
	assert.Equal(t, int64(250000), *entity.SuperloadWeightPounds)
	assert.Equal(t, "30", entity.PermitBaseFee.Decimal.String())
	assert.False(t, entity.PermitPerMileFee.Valid)
	assert.Empty(t, entity.VerificationState)
}
