package iamservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func identityProviderRequest(issuer string) *iamports.IdentityProviderRequest {
	return &iamports.IdentityProviderRequest{
		Name:            "Keycloak",
		Protocol:        iam.IdentityProviderProtocolOIDC,
		Enabled:         true,
		OIDCIssuerURL:   issuer,
		OIDCClientID:    "trenova",
		OIDCRedirectURL: "https://app.trenova.example/api/v1/auth/sso/callback",
		OIDCScopes:      []string{"openid", "email"},
	}
}

func fieldHasError(err error, field string) bool {
	multiErr, ok := err.(*errortypes.MultiError)
	if !ok {
		return false
	}
	for _, fieldErr := range multiErr.Errors {
		if fieldErr.Field == field {
			return true
		}
	}
	return false
}

func TestIdentityProviderIssuerMustBeAPublicHTTPSAddress(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop()}).(*service)
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	existing := &iam.IdentityProvider{OIDCClientSecret: "sealed"}

	for _, issuer := range []string{
		"http://idp.example.com/realms/trenova",
		"https://127.0.0.1/realms/trenova",
		"https://169.254.169.254/latest/meta-data",
		"https://admin:pw@idp.example.com/realms/trenova",
	} {
		_, err := svc.identityProviderFromRequest(
			tenantInfo,
			identityProviderRequest(issuer),
			existing,
		)
		require.True(t, fieldHasError(err, "oidcIssuerUrl"), issuer)
	}

	_, err := svc.identityProviderFromRequest(
		tenantInfo,
		identityProviderRequest("https://idp.example.com/realms/trenova"),
		existing,
	)
	require.False(t, fieldHasError(err, "oidcIssuerUrl"))

	private := New(Params{
		Logger: zap.NewNop(),
		SecurityConfig: &securityconfig.Config{
			SSO: securityconfig.SSOConfig{AllowPrivateNetworkIssuers: true},
		},
	}).(*service)
	_, err = private.identityProviderFromRequest(
		tenantInfo,
		identityProviderRequest("https://10.2.0.4/realms/trenova"),
		existing,
	)
	require.False(t, fieldHasError(err, "oidcIssuerUrl"))
}
