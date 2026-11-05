package userservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sso"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRequireTargetSignIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		configs []*sso.SSOConfig
		refused bool
	}{
		{name: "no single sign-on"},
		{name: "offered but not enforced", configs: []*sso.SSOConfig{{EnforceSSO: false}}},
		{
			name: "enforced by any enabled provider",
			configs: []*sso.SSOConfig{
				{EnforceSSO: false},
				{EnforceSSO: true, Provider: sso.SSOProvider("entra-id")},
			},
			refused: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configs := tenantportsmock.NewMockSSOConfigRepository(t)
			target := &tenant.OrganizationMembership{
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
			}
			configs.EXPECT().
				ListEnabledByOrganizationID(mock.Anything, target.OrganizationID).
				Return(tt.configs, nil).
				Once()
			svc := &Service{ssoConfigs: configs}

			err := svc.requireTargetSignIn(
				t.Context(),
				&session.Session{UserID: pulid.MustNew("usr_")},
				target,
			)

			if !tt.refused {
				require.NoError(t, err)
				return
			}
			assert.True(t, errortypes.IsAuthorizationError(err))
		})
	}
}

func TestAFederatedSessionSwitchesOnlyWhereItsIssuerIsTrusted(t *testing.T) {
	t.Parallel()

	const issuer = "https://idp.example.com"
	tests := []struct {
		name    string
		configs []*sso.SSOConfig
		sess    session.Session
		refused bool
	}{
		{
			name:    "no provider in the target",
			sess:    session.Session{ExternalSubject: "sub", ExternalIssuer: issuer},
			refused: true,
		},
		{
			name:    "another issuer in the target",
			configs: []*sso.SSOConfig{{OIDCIssuerURL: "https://other.example.com"}},
			sess:    session.Session{ExternalSubject: "sub", ExternalIssuer: issuer},
			refused: true,
		},
		{
			name:    "a session from before the issuer was kept",
			configs: []*sso.SSOConfig{{OIDCIssuerURL: issuer}},
			sess:    session.Session{ExternalSubject: "sub"},
			refused: true,
		},
		{
			name:    "the same issuer in the target",
			configs: []*sso.SSOConfig{{OIDCIssuerURL: issuer}},
			sess:    session.Session{ExternalSubject: "sub", ExternalIssuer: issuer},
		},
		{name: "a password session"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configs := tenantportsmock.NewMockSSOConfigRepository(t)
			target := &tenant.OrganizationMembership{
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
			}
			configs.EXPECT().
				ListEnabledByOrganizationID(mock.Anything, target.OrganizationID).
				Return(tt.configs, nil).
				Once()
			svc := &Service{ssoConfigs: configs}
			sess := tt.sess
			sess.UserID = pulid.MustNew("usr_")

			err := svc.requireTargetSignIn(t.Context(), &sess, target)

			if !tt.refused {
				require.NoError(t, err)
				return
			}
			assert.True(t, errortypes.IsAuthorizationError(err))
		})
	}
}
