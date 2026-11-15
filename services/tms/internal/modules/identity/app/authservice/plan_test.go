package authservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/sso"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type failingGate struct{}

func (failingGate) Enabled(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return false, errors.New("the plan could not be read")
}

func signInService(
	t *testing.T,
	gate platformcatalogports.FeatureGate,
	enforced bool,
) (*Service, *sso.SSOConfig) {
	t.Helper()

	org := &tenant.Organization{
		ID:             pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Acme Freight",
		LoginSlug:      "acme",
	}
	cfg := &sso.SSOConfig{
		ID:         pulid.MustNew("sso_"),
		Name:       "Acme Okta",
		Provider:   sso.SSOProviderOkta,
		Protocol:   sso.SSOProtocolOIDC,
		Enabled:    true,
		EnforceSSO: enforced,
	}

	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByLoginSlug(mock.Anything, "acme").Return(org, nil).Maybe()
	configs := tenantportsmock.NewMockSSOConfigRepository(t)
	configs.EXPECT().ListEnabledByOrganizationID(mock.Anything, org.ID).
		Return([]*sso.SSOConfig{cfg}, nil).
		Maybe()

	return &Service{
		or:       orgs,
		ssoRepo:  configs,
		features: gate,
		l:        zap.NewNop(),
	}, cfg
}

func TestListAuthProviders_ListsSingleSignOnOnlyWhenItIsOnThePlan(t *testing.T) {
	t.Parallel()

	svc, cfg := signInService(t, plantest.AllFeatures(), false)
	providers, err := svc.ListAuthProviders(t.Context(), "acme")
	require.NoError(t, err)
	require.Len(t, providers, 1)
	assert.Equal(t, cfg.ID, providers[0].ID)
	assert.Equal(t, sso.SSOProviderOkta, providers[0].Provider)

	svc, _ = signInService(t, plantest.Without(platformcatalog.FeatureIdentity), false)
	providers, err = svc.ListAuthProviders(t.Context(), "acme")
	require.NoError(t, err)
	assert.Empty(t, providers)
}

func TestListAuthProviders_AGateErrorListsNoSingleSignOn(t *testing.T) {
	t.Parallel()

	svc, _ := signInService(t, failingGate{}, false)
	providers, err := svc.ListAuthProviders(t.Context(), "acme")

	require.NoError(t, err)
	assert.Empty(t, providers)
}

func TestGetTenantLoginMetadata_OffThePlanOffersPasswordSignInOnly(t *testing.T) {
	t.Parallel()

	svc, _ := signInService(t, plantest.AllFeatures(), true)
	meta, err := svc.GetTenantLoginMetadata(t.Context(), "acme")
	require.NoError(t, err)
	assert.Equal(t, []string{string(sso.SSOProviderOkta)}, meta.EnabledProviders)
	assert.True(t, meta.EnforceSSO)
	assert.False(t, meta.PasswordEnabled)

	for name, gate := range map[string]platformcatalogports.FeatureGate{
		"security and identity off": plantest.Without(platformcatalog.FeatureIdentity),
		"the gate fails":            failingGate{},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svc, _ := signInService(t, gate, true)
			meta, err := svc.GetTenantLoginMetadata(t.Context(), "acme")
			require.NoError(t, err)
			assert.Equal(t, "Acme Freight", meta.OrganizationName)
			assert.Empty(t, meta.EnabledProviders)
			assert.False(t, meta.EnforceSSO)
			assert.True(t, meta.PasswordEnabled)
		})
	}
}

func TestEnforcePasswordLoginPolicy_EnforcesSingleSignOnOnlyOnThePlan(t *testing.T) {
	t.Parallel()

	user := &tenant.User{ID: pulid.MustNew("usr_")}
	target := func(svc *Service) *tenant.Organization {
		org, err := svc.or.GetByLoginSlug(t.Context(), "acme")
		require.NoError(t, err)
		return org
	}

	svc, _ := signInService(t, plantest.AllFeatures(), true)
	err := svc.enforcePasswordLoginPolicy(t.Context(), "acme", user, target(svc))
	require.ErrorIs(t, err, errSSORequired)

	svc, _ = signInService(t, failingGate{}, true)
	err = svc.enforcePasswordLoginPolicy(t.Context(), "acme", user, target(svc))
	require.ErrorIs(t, err, errSSORequired, "a gate error keeps single sign-on enforced")

	svc, _ = signInService(t, plantest.Without(platformcatalog.FeatureIdentity), true)
	err = svc.enforcePasswordLoginPolicy(t.Context(), "acme", user, target(svc))
	require.NoError(t, err, "with single sign-on off the plan, password sign-in is the way in")
}
