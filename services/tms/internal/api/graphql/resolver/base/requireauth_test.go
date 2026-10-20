package base

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireAuth_RefusesAnAgentPrincipal(t *testing.T) {
	t.Parallel()

	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeAgent,
		PrincipalID:    pulid.MustNew("agd_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	got, err := (&Core{}).RequireAuth(ctx)

	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestRequireAuth_AcceptsAPerson(t *testing.T) {
	t.Parallel()

	auth := &authctx.AuthContext{
		PrincipalType: authctx.PrincipalTypeUser,
		PrincipalID:   pulid.MustNew("usr_"),
	}

	got, err := (&Core{}).RequireAuth(gqlctx.WithAuthContext(t.Context(), auth))

	require.NoError(t, err)
	assert.Same(t, auth, got)
}
