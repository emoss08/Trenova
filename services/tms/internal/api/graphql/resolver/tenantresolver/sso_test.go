package tenantresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func ssoAuth() *authctx.AuthContext {
	userID := pulid.MustNew("usr_")
	return &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func TestMicrosoftSSOConfigReadsTheCallersOrganization(t *testing.T) {
	t.Parallel()

	auth := ssoAuth()
	service := tenantportsmock.NewMockOrganizationService(t)
	service.EXPECT().GetMicrosoftSSOConfig(mock.Anything, auth.OrganizationID).
		Return(&tenantports.MicrosoftSSOConfig{TenantID: "tenant", SecretConfigured: true}, nil).
		Once()
	engine := &resolvertest.RecordingPermissionEngine{}

	config, err := (&QueryResolver{&Deps{
		Core:                &resolvercore.Core{PermissionEngine: engine},
		OrganizationService: service,
	}}).MicrosoftSsoConfig(gqlctx.WithAuthContext(t.Context(), auth))

	require.NoError(t, err)
	assert.True(t, config.SecretConfigured)
	require.NotNil(t, engine.Request)
	assert.Equal(t, permission.ResourceOrganization.String(), engine.Request.Resource)
	assert.Equal(t, permission.OpRead, engine.Request.Operation)
}

func TestUpsertOktaSSOConfigWritesTheCallersOrganization(t *testing.T) {
	t.Parallel()

	auth := ssoAuth()
	secret := "s3cret"
	service := tenantportsmock.NewMockOrganizationService(t)
	service.EXPECT().
		UpsertOktaSSOConfig(
			mock.Anything,
			pagination.TenantInfo{
				OrgID:  auth.OrganizationID,
				BuID:   auth.BusinessUnitID,
				UserID: auth.UserID,
			},
			mock.MatchedBy(func(config *tenantports.OktaSSOConfig) bool {
				return config.OrganizationID == auth.OrganizationID.String() &&
					config.IssuerURL == "https://acme.okta.com" &&
					config.ClientSecret == "s3cret" &&
					config.EnforceSSO
			}),
		).
		Return(&tenantports.OktaSSOConfig{SecretConfigured: true}, nil).
		Once()
	engine := &resolvertest.RecordingPermissionEngine{}

	_, err := (&MutationResolver{&Deps{
		Core:                &resolvercore.Core{PermissionEngine: engine},
		OrganizationService: service,
	}}).UpsertOktaSsoConfig(gqlctx.WithAuthContext(t.Context(), auth), gqlmodel.OktaSSOConfigInput{
		Enabled:      true,
		EnforceSso:   true,
		IssuerURL:    "https://acme.okta.com",
		ClientID:     "client",
		ClientSecret: &secret,
		RedirectURL:  "https://app.example.com/callback",
	})

	require.NoError(t, err)
	require.NotNil(t, engine.Request)
	assert.Equal(t, permission.OpUpdate, engine.Request.Operation)
}

func TestSSOResolversRefuseWithoutOrganizationAccess(t *testing.T) {
	t.Parallel()

	engine := &resolvertest.GrantingPermissionEngine{}
	resolver := &QueryResolver{&Deps{Core: &resolvercore.Core{PermissionEngine: engine}}}

	_, err := resolver.OktaSsoConfig(gqlctx.WithAuthContext(t.Context(), ssoAuth()))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	require.Len(t, engine.Requests, 1)
	assert.Equal(t, permission.ResourceOrganization.String(), engine.Requests[0].Resource)
	assert.Equal(t, permission.OpRead, engine.Requests[0].Operation)
}

func TestOrganizationRefusesAnotherOrganization(t *testing.T) {
	t.Parallel()

	resolver := &QueryResolver{&Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &resolvertest.RecordingPermissionEngine{},
		},
		OrganizationService: tenantportsmock.NewMockOrganizationService(t),
	}}

	_, err := resolver.Organization(
		gqlctx.WithAuthContext(t.Context(), ssoAuth()),
		pulid.MustNew("org_").String(),
		nil,
		nil,
	)

	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}
