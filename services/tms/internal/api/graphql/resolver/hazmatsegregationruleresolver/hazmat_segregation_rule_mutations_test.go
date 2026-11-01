package hazmatsegregationruleresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func segregationRuleInput() gqlmodel.HazmatSegregationRuleInput {
	return gqlmodel.HazmatSegregationRuleInput{
		Status:          domaintypes.StatusActive,
		Name:            "Flammables from corrosives",
		ClassA:          hazardousmaterial.HazardousClass3,
		ClassB:          hazardousmaterial.HazardousClass8,
		SegregationType: hazmatsegregationrule.SegregationTypeProhibited,
	}
}

func TestHazmatSegregationRuleMutations_CheckTheSegregationRulePermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("hsr_").String()
	input := segregationRuleInput()

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"createHazmatSegregationRule",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateHazmatSegregationRule(ctx, input)
				return err
			},
		},
		{
			"updateHazmatSegregationRule",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateHazmatSegregationRule(ctx, id, input)
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

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceHazmatSegregationRule,
				call.operation,
			)
		})
	}
}

func TestHazmatSegregationRuleFromInput_MapsTheInputIntoTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	hazmatAID := pulid.MustNew("hm_")
	hazmatARaw := hazmatAID.String()
	distance := 3.5
	unit := "FT"
	notes := "Allowed in sealed drums"
	input := segregationRuleInput()
	input.SegregationType = hazmatsegregationrule.SegregationTypeDistance
	input.HazmatAId = &hazmatARaw
	input.MinimumDistance = &distance
	input.DistanceUnit = &unit
	input.HasExceptions = true
	input.ExceptionNotes = &notes

	entity, err := hazmatSegregationRuleFromInput(&input, authCtx)
	require.NoError(t, err)

	assert.Equal(t, authCtx.OrganizationID, entity.OrganizationID)
	assert.Equal(t, authCtx.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, hazardousmaterial.HazardousClass3, entity.ClassA)
	assert.Equal(t, hazardousmaterial.HazardousClass8, entity.ClassB)
	require.NotNil(t, entity.HazmatAID)
	assert.Equal(t, hazmatAID, *entity.HazmatAID)
	assert.Nil(t, entity.HazmatBID)
	assert.Equal(t, &distance, entity.MinimumDistance)
	assert.Equal(t, "FT", entity.DistanceUnit)
	assert.True(t, entity.HasExceptions)
	assert.Equal(t, notes, entity.ExceptionNotes)
}

func TestApplyHazmatSegregationRuleInput_ClearsWhatTheInputLeavesOut(t *testing.T) {
	t.Parallel()

	hazmatBID := pulid.MustNew("hm_")
	distance := 10.0
	existing := &hazmatsegregationrule.HazmatSegregationRule{
		HazmatBID:       &hazmatBID,
		MinimumDistance: &distance,
		DistanceUnit:    "M",
		Version:         3,
	}
	input := segregationRuleInput()

	require.NoError(t, applyHazmatSegregationRuleInput(existing, &input))

	assert.Nil(t, existing.HazmatBID)
	assert.Nil(t, existing.MinimumDistance)
	assert.Empty(t, existing.DistanceUnit)
	assert.Equal(t, int64(3), existing.Version)
}

func TestApplyHazmatSegregationRuleInput_NamesAMalformedMaterial(t *testing.T) {
	t.Parallel()

	raw := "nope"
	input := segregationRuleInput()
	input.HazmatBId = &raw

	err := applyHazmatSegregationRuleInput(&hazmatsegregationrule.HazmatSegregationRule{}, &input)

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "hazmatBId", valErr.Field)
}
