package authctx

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindSessionCarriesAttributesAndRoleActivation(t *testing.T) {
	t.Parallel()

	roleID := pulid.MustNew("rol_")
	ctx := BindSession(t.Context(), SessionAttributes{
		UserID:           pulid.MustNew("usr_"),
		ActiveRoleIDs:    []pulid.ID{roleID},
		RoleActivation:   true,
		AuthenticatorAAL: 2,
	})

	session, ok := GetSessionAttributes(ctx)
	require.True(t, ok)
	assert.Equal(t, 2, session.AuthenticatorAAL)

	activation, ok := GetSessionRoleActivation(ctx)
	require.True(t, ok)
	assert.Equal(t, []pulid.ID{roleID}, activation.ActiveRoleIDs)
}

func TestBindSessionLeavesRolesAloneWhereActivationIsOff(t *testing.T) {
	t.Parallel()

	ctx := BindSession(t.Context(), SessionAttributes{AuthenticatorAAL: 1})

	_, ok := GetSessionRoleActivation(ctx)
	assert.False(t, ok)
}

func TestAuthContextSessionIsNilWithoutOne(t *testing.T) {
	t.Parallel()

	assert.Nil(t, (&AuthContext{UserID: pulid.MustNew("usr_")}).Session())
	assert.Nil(t, (*AuthContext)(nil).Session())
}
