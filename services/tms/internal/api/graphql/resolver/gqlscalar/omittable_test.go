package gqlscalar

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyRequired_WritesASetValueAndLeavesAnAbsentOne(t *testing.T) {
	t.Parallel()

	name := "kept"
	require.NoError(t, ApplyRequired("name", "Name", graphql.Omittable[*string]{}, &name))
	assert.Equal(t, "kept", name)

	next := "changed"
	require.NoError(t, ApplyRequired("name", "Name", graphql.OmittableOf(&next), &name))
	assert.Equal(t, "changed", name)
}

func TestApplyRequired_RefusesAnExplicitNull(t *testing.T) {
	t.Parallel()

	name := "kept"
	err := ApplyRequired("name", "Name", graphql.OmittableOf[*string](nil), &name)

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "name", valErr.Field)
	assert.Equal(t, errortypes.ErrRequired, valErr.Code)
	assert.Equal(t, "kept", name)
}

func TestApplyRequiredInt_ConvertsToTheDomainWidth(t *testing.T) {
	t.Parallel()

	var version int64 = 3
	require.NoError(t, ApplyRequiredInt("version", "Version", graphql.Omittable[*int]{}, &version))
	assert.Equal(t, int64(3), version)

	next := 4
	require.NoError(t, ApplyRequiredInt("version", "Version", graphql.OmittableOf(&next), &version))
	assert.Equal(t, int64(4), version)

	err := ApplyRequiredInt("version", "Version", graphql.OmittableOf[*int](nil), &version)
	require.Error(t, err)
	assert.Equal(t, int64(4), version)
}

func TestApplyClearable_ClearsOnNullAndKeepsOnAbsent(t *testing.T) {
	t.Parallel()

	color := "#fff"
	ApplyClearable(graphql.Omittable[*string]{}, &color)
	assert.Equal(t, "#fff", color)

	ApplyClearable(graphql.OmittableOf[*string](nil), &color)
	assert.Empty(t, color)

	next := "#000"
	ApplyClearable(graphql.OmittableOf(&next), &color)
	assert.Equal(t, "#000", color)
}

func TestApplyNullable_SetsThePointerOnlyWhenTheFieldIsSent(t *testing.T) {
	t.Parallel()

	start := 1.5
	goal := &start
	ApplyNullable(graphql.Omittable[*float64]{}, &goal)
	assert.Equal(t, &start, goal)

	ApplyNullable(graphql.OmittableOf[*float64](nil), &goal)
	assert.Nil(t, goal)

	next := 2.5
	ApplyNullable(graphql.OmittableOf(&next), &goal)
	assert.Equal(t, &next, goal)
}
