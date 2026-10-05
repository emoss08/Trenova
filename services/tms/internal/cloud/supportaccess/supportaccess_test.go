package supportaccess_test

import (
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	supportctx "github.com/emoss08/trenova/internal/cloud/supportaccess"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecideRoute(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		method string
		route  string
		write  bool
		want   supportctx.Decision
	}{
		{"read", http.MethodGet, "/api/v1/shipments/", false, supportctx.DecisionAllow},
		{"read of a denied area", http.MethodGet, "/api/v1/api-keys/", false, supportctx.DecisionAllow},
		{"write while read-only", http.MethodPost, "/api/v1/shipments/", false, supportctx.DecisionReadOnly},
		{"delete while read-only", http.MethodDelete, "/api/v1/workers/:workerID/", false, supportctx.DecisionReadOnly},
		{"upload while read-only", http.MethodPost, "/api/v1/documents/upload/", false, supportctx.DecisionReadOnly},
		{"write while elevated", http.MethodPost, "/api/v1/shipments/", true, supportctx.DecisionAllow},
		{"preview while read-only", http.MethodPost, "/api/v1/shipments/calculate-totals", false, supportctx.DecisionAllow},
		{"permission check", http.MethodPost, "/api/v1/me/permissions/check", false, supportctx.DecisionAllow},
		{"graphql transport", http.MethodPost, "/graphql", false, supportctx.DecisionAllow},
		{"own password", http.MethodPost, "/api/v1/users/me/change-password/", true, supportctx.DecisionDenied},
		{"switch organization", http.MethodPost, "/api/v1/users/me/switch-organization/", true, supportctx.DecisionDenied},
		{"user password reset", http.MethodPost, "/api/v1/users/:userID/reset-password/", true, supportctx.DecisionDenied},
		{"api key", http.MethodPost, "/api/v1/api-keys/", true, supportctx.DecisionDenied},
		{"role", http.MethodPut, "/api/v1/roles/:roleID", true, supportctx.DecisionDenied},
		{"identity provider", http.MethodPost, "/api/v1/organizations/:id/iam/identity-providers", true, supportctx.DecisionDenied},
		{"okta sso", http.MethodPut, "/api/v1/organizations/:id/okta-sso", true, supportctx.DecisionDenied},
		{"grant", http.MethodPost, "/api/v1/support-access/grant/", true, supportctx.DecisionDenied},
		{"assistant", http.MethodPost, "/api/v1/assistant/turns/", true, supportctx.DecisionDenied},
		{"presence", http.MethodPost, "/api/v1/shipments/:shipmentID/comments/presence", true, supportctx.DecisionDenied},
		{"push subscription", http.MethodPost, "/api/v1/push/subscriptions/", true, supportctx.DecisionDenied},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := supportctx.DecideRoute(supportctx.RouteRequest{
				Method:      tc.method,
				Route:       tc.route,
				WriteActive: tc.write,
			})
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDecideMutation(t *testing.T) {
	t.Parallel()

	assert.Equal(t, supportctx.DecisionReadOnly, supportctx.DecideMutation(supportctx.MutationRequest{
		FieldName: "updateShipment", Source: "shipment.graphqls",
	}))
	assert.Equal(t, supportctx.DecisionAllow, supportctx.DecideMutation(supportctx.MutationRequest{
		FieldName: "updateShipment", Source: "shipment.graphqls", WriteActive: true,
	}))
	assert.Equal(t, supportctx.DecisionDenied, supportctx.DecideMutation(supportctx.MutationRequest{
		FieldName: "createApiKey", Source: "api_key.graphqls", WriteActive: true,
	}))
	assert.Equal(t, supportctx.DecisionDenied, supportctx.DecideMutation(supportctx.MutationRequest{
		FieldName: "updateMyPreferences", Source: "self_service.graphqls", WriteActive: true,
	}))
	assert.Equal(t, supportctx.DecisionDenied, supportctx.DecideMutation(supportctx.MutationRequest{
		FieldName: "myCommuteSettings", Source: "worker.graphqls", WriteActive: true,
	}))
	assert.Equal(t, supportctx.DecisionDenied, supportctx.DecideMutation(supportctx.MutationRequest{
		FieldName: "approveAgentProposal", Source: "decisions.graphqls", WriteActive: true,
	}))
}

func TestTokenRoundTripsAndRefusesTampering(t *testing.T) {
	t.Parallel()

	org, bu, session := pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("sps_")
	issued, err := supportctx.IssueToken(org, bu, session)
	require.NoError(t, err)

	parsed, err := supportctx.ParseToken(issued.Value)
	require.NoError(t, err)
	assert.Equal(t, org, parsed.OrganizationID)
	assert.Equal(t, bu, parsed.BusinessUnitID)
	assert.Equal(t, session, parsed.SessionID)
	assert.True(t, parsed.Matches(issued.SecretHash))
	assert.NotContains(t, issued.SecretHash, parsed.Secret)

	other, err := supportctx.IssueToken(org, bu, session)
	require.NoError(t, err)
	assert.False(t, parsed.Matches(other.SecretHash))

	for _, value := range []string{"", "v1.a.b.c", "v2." + issued.Value[3:], "v1.x.y.z.secret"} {
		_, err = supportctx.ParseToken(value)
		require.ErrorIs(t, err, supportctx.ErrMalformedToken, value)
	}
}

func TestPermissionSourceGrantsReadsOrAdministratorWritesMinusTheDenyList(t *testing.T) {
	t.Parallel()

	source := supportctx.NewPermissionSource(supportctx.PermissionSourceParams{
		Registry: permission.NewRegistry(),
	})
	active := &supportctx.Active{
		OrganizationID:  pulid.MustNew("org_"),
		PrincipalUserID: pulid.MustNew("usr_"),
		GrantMode:       supportaccess.AccessModeReadWrite,
		ExpiresAt:       5000,
		Now:             1000,
	}
	ctx := supportctx.WithActive(t.Context(), active)

	_, ok, err := source.DelegatedPermissions(t.Context(), active.PrincipalUserID, active.OrganizationID)
	require.NoError(t, err)
	assert.False(t, ok, "no support session on the context")

	_, ok, err = source.DelegatedPermissions(ctx, pulid.MustNew("usr_"), active.OrganizationID)
	require.NoError(t, err)
	assert.False(t, ok, "another user in the same request")

	readOnly, ok, err := source.DelegatedPermissions(ctx, active.PrincipalUserID, active.OrganizationID)
	require.NoError(t, err)
	require.True(t, ok)
	shipment := readOnly.Resources[permission.ResourceShipment.String()]
	require.NotNil(t, shipment)
	assert.Equal(t, []string{string(permission.OpRead)}, shipment.Operations)

	active.ElevatedUntil = 2000
	readWrite, ok, err := source.DelegatedPermissions(ctx, active.PrincipalUserID, active.OrganizationID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Contains(t, readWrite.Resources[permission.ResourceShipment.String()].Operations,
		string(permission.OpUpdate))

	for _, denied := range supportctx.DeniedResources() {
		perms, present := readWrite.Resources[denied.String()]
		if !present {
			continue
		}
		assert.Equal(t, []string{string(permission.OpRead)}, perms.Operations, denied.String())
	}
}
