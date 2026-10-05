package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLatestConstraintDefinitionReturnsTheNewestDefinition(t *testing.T) {
	t.Parallel()

	definition, err := LatestConstraintDefinition("ck_organization_onboarding_status")
	require.NoError(t, err)
	assert.Contains(t, definition, "ck_organization_onboarding_status")
}

func TestLatestConstraintDefinitionRefusesAnUnknownConstraint(t *testing.T) {
	t.Parallel()

	_, err := LatestConstraintDefinition("ck_does_not_exist")
	require.ErrorIs(t, err, ErrConstraintNotFound)
}
