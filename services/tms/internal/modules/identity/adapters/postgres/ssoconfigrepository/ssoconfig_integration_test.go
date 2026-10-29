//go:build integration

package ssoconfigrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sso"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSavingOktaSettingsKeepsTheFederatedMFATrustAnAdministratorChose(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	ctx = data.Ctx(ctx)
	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	okta := &sso.SSOConfig{
		OrganizationID:   data.Organization.ID,
		BusinessUnitID:   data.BusinessUnit.ID,
		Name:             "Okta",
		Provider:         sso.SSOProviderOkta,
		Protocol:         sso.SSOProtocolOIDC,
		Enabled:          true,
		AttributeMap:     map[string]string{"email": "email"},
		OIDCIssuerURL:    "https://trenova.okta.com",
		OIDCClientID:     "client",
		OIDCClientSecret: "sealed",
		OIDCRedirectURL:  "https://app.example.test/api/v1/auth/sso/okta/callback",
		OIDCScopes:       []string{"openid", "email"},
	}
	saved, err := repo.Save(ctx, okta)
	require.NoError(t, err)

	idp := iamdb.IdentityProviders
	_, err = idp.Where(idp.ID.Eq(saved.ID)).Update(
		ctx,
		db,
		idp.AllowFederatedMFA.To(false),
		idp.FederatedMFAACRs.To([]string{"urn:okta:loa:2fa:any"}),
	)
	require.NoError(t, err)

	okta.OIDCClientID = "client-rotated"
	_, err = repo.Save(ctx, okta)
	require.NoError(t, err)

	loaded, err := repo.GetEnabledByOrganizationID(ctx, data.Organization.ID, sso.SSOProviderOkta)
	require.NoError(t, err)
	assert.Equal(t, "client-rotated", loaded.OIDCClientID)
	assert.False(t, loaded.AllowFederatedMFA)
	assert.Equal(t, []string{"urn:okta:loa:2fa:any"}, loaded.FederatedMFAACRs)
}
