package gqlid

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyRequiredID_LeavesTheValueWhenOmitted(t *testing.T) {
	t.Parallel()

	existing := pulid.MustNew("st_")
	dest := existing

	require.NoError(t, ApplyRequiredID("stateId", "State", graphql.Omittable[*string]{}, &dest))
	assert.Equal(t, existing, dest)
}

func TestApplyRequiredID_SetsAGivenID(t *testing.T) {
	t.Parallel()

	next := pulid.MustNew("st_")
	raw := next.String()
	var dest pulid.ID

	require.NoError(t, ApplyRequiredID("stateId", "State", graphql.OmittableOf(&raw), &dest))
	assert.Equal(t, next, dest)
}

func TestApplyRequiredID_RefusesNull(t *testing.T) {
	t.Parallel()

	dest := pulid.MustNew("st_")
	err := ApplyRequiredID("stateId", "State", graphql.OmittableOf[*string](nil), &dest)

	var valErr *errortypes.Error
	require.ErrorAs(t, err, &valErr)
	assert.Equal(t, "stateId", valErr.Field)
}

func TestApplyOptionalID_ClearsOnNull(t *testing.T) {
	t.Parallel()

	dest := pulid.MustNew("fc_")

	require.NoError(t, ApplyOptionalID("fleetCodeId", graphql.OmittableOf[*string](nil), &dest))
	assert.True(t, dest.IsNil())
}
