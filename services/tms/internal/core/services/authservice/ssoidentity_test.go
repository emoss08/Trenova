package authservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeSSOLinks struct {
	links      []*tenant.SSOIdentityLink
	createErr  error
	logins     int
	onCreating func()
}

func (f *fakeSSOLinks) GetBySubject(
	_ context.Context,
	req repositories.GetSSOIdentityLinkBySubjectRequest,
) (*tenant.SSOIdentityLink, error) {
	for _, link := range f.links {
		if link.SSOConfigID == req.SSOConfigID && link.Issuer == req.Issuer &&
			link.Subject == req.Subject {
			return link, nil
		}
	}

	return nil, errortypes.NewNotFoundError("SSO identity link not found")
}

func (f *fakeSSOLinks) GetByUser(
	_ context.Context,
	req repositories.GetSSOIdentityLinkByUserRequest,
) (*tenant.SSOIdentityLink, error) {
	for _, link := range f.links {
		if link.SSOConfigID == req.SSOConfigID && link.Issuer == req.Issuer &&
			link.UserID == req.UserID {
			return link, nil
		}
	}

	return nil, errortypes.NewNotFoundError("SSO identity link not found")
}

func (f *fakeSSOLinks) Create(_ context.Context, link *tenant.SSOIdentityLink) error {
	if f.onCreating != nil {
		f.onCreating()
	}
	if f.createErr != nil {
		return f.createErr
	}
	f.links = append(f.links, link)

	return nil
}

func (f *fakeSSOLinks) RecordLogin(context.Context, *tenant.SSOIdentityLink, int64) error {
	f.logins++
	return nil
}

const testIssuer = "https://idp.example.com"

func ssoTestConfig(provider tenant.SSOProvider) *tenant.SSOConfig {
	return &tenant.SSOConfig{
		ID:             pulid.MustNew("sso_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Provider:       provider,
	}
}

func lookup(cfg *tenant.SSOConfig, identity ssoIdentity) *ssoUserLookup {
	return &ssoUserLookup{Config: cfg, Identity: identity, DisplayName: "Okta"}
}

func TestResolveSSOUser_FirstSignInLinksAVerifiedEmail(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	links := &fakeSSOLinks{}
	deps.svc.links = links
	usr := newTestUser(t)
	cfg := ssoTestConfig(tenant.SSOProviderOkta)

	deps.userRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(usr, nil)

	got, err := deps.svc.resolveSSOUser(t.Context(), lookup(cfg, ssoIdentity{
		Issuer: testIssuer, Subject: "okta|1", Email: "test@example.com", EmailTrusted: true,
	}))

	require.NoError(t, err)
	assert.Equal(t, usr.ID, got.ID)
	require.Len(t, links.links, 1)
	assert.Equal(t, usr.ID, links.links[0].UserID)
	assert.Equal(t, "okta|1", links.links[0].Subject)
	assert.Equal(t, cfg.OrganizationID, links.links[0].OrganizationID)
}

func TestResolveSSOUser_RefusesAnUnverifiedEmailOnFirstSignIn(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	links := &fakeSSOLinks{}
	deps.svc.links = links

	_, err := deps.svc.resolveSSOUser(t.Context(), lookup(ssoTestConfig(tenant.SSOProviderOkta),
		ssoIdentity{Issuer: testIssuer, Subject: "okta|1", Email: "test@example.com"}))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	assert.Empty(t, links.links)
	deps.userRepo.AssertNotCalled(t, "FindByEmail", mock.Anything, mock.Anything)
}

func TestResolveSSOUser_LaterSignInsResolveBySubjectEvenIfTheEmailChanges(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)
	cfg := ssoTestConfig(tenant.SSOProviderOkta)
	links := &fakeSSOLinks{links: []*tenant.SSOIdentityLink{{
		ID: pulid.MustNew("ssoil_"), SSOConfigID: cfg.ID, Issuer: testIssuer,
		Subject: "okta|1", UserID: usr.ID,
	}}}
	deps.svc.links = links

	deps.userRepo.On("FindByIDForLogin", mock.Anything, usr.ID).Return(usr, nil)

	got, err := deps.svc.resolveSSOUser(t.Context(), lookup(cfg, ssoIdentity{
		Issuer: testIssuer, Subject: "okta|1", Email: "renamed@example.com",
	}))

	require.NoError(t, err)
	assert.Equal(t, usr.ID, got.ID)
	assert.Equal(t, 1, links.logins)
	deps.userRepo.AssertNotCalled(t, "FindByEmail", mock.Anything, mock.Anything)
}

func TestResolveSSOUser_RefusesASecondIdentityClaimingAnAlreadyLinkedUser(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)
	cfg := ssoTestConfig(tenant.SSOProviderOkta)
	links := &fakeSSOLinks{links: []*tenant.SSOIdentityLink{{
		ID: pulid.MustNew("ssoil_"), SSOConfigID: cfg.ID, Issuer: testIssuer,
		Subject: "okta|victim", UserID: usr.ID,
	}}}
	deps.svc.links = links

	deps.userRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(usr, nil)

	_, err := deps.svc.resolveSSOUser(t.Context(), lookup(cfg, ssoIdentity{
		Issuer: testIssuer, Subject: "okta|impostor", Email: "test@example.com", EmailTrusted: true,
	}))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
	assert.Len(t, links.links, 1)
}

func TestResolveSSOUser_RefusesATokenWithoutASubject(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.svc.links = &fakeSSOLinks{}

	_, err := deps.svc.resolveSSOUser(t.Context(), lookup(ssoTestConfig(tenant.SSOProviderOkta),
		ssoIdentity{Issuer: testIssuer, Email: "test@example.com", EmailTrusted: true}))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
}

func TestResolveSSOUser_EnforcesAllowedDomains(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	deps.svc.links = &fakeSSOLinks{}
	cfg := ssoTestConfig(tenant.SSOProviderOkta)
	cfg.AllowedDomains = []string{"carrier.example"}

	_, err := deps.svc.resolveSSOUser(t.Context(), lookup(cfg, ssoIdentity{
		Issuer: testIssuer, Subject: "okta|1", Email: "test@example.com", EmailTrusted: true,
	}))

	require.Error(t, err)
}

func TestResolveSSOUser_ConcurrentLinkForAnotherUserIsRefused(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)
	usr := newTestUser(t)
	cfg := ssoTestConfig(tenant.SSOProviderOkta)
	links := &fakeSSOLinks{
		createErr: &pgconn.PgError{Code: pgerrcode.UniqueViolation},
	}
	links.onCreating = func() {
		links.links = append(links.links, &tenant.SSOIdentityLink{
			SSOConfigID: cfg.ID, Issuer: testIssuer, Subject: "okta|1",
			UserID: pulid.MustNew("usr_"),
		})
	}
	deps.svc.links = links

	deps.userRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(usr, nil)

	_, err := deps.svc.resolveSSOUser(t.Context(), lookup(cfg, ssoIdentity{
		Issuer: testIssuer, Subject: "okta|1", Email: "test@example.com", EmailTrusted: true,
	}))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthenticationError(err))
}

func TestIdentityFromOIDCClaims(t *testing.T) {
	t.Parallel()

	t.Run("generic providers trust only a verified email and never fall back", func(t *testing.T) {
		t.Parallel()

		identity := identityFromOIDCClaims(testIssuer, oidcClaims{
			Subject:           "okta|1",
			PreferredUsername: "victim@example.com",
		}, tenant.SSOProviderOkta)
		assert.Empty(t, identity.Email)
		assert.False(t, identity.EmailTrusted)

		identity = identityFromOIDCClaims(testIssuer, oidcClaims{
			Subject:       "okta|1",
			Email:         "Person@Example.com",
			EmailVerified: oidcBool{set: true, value: true},
		}, tenant.SSOProviderOkta)
		assert.Equal(t, "person@example.com", identity.Email)
		assert.True(t, identity.EmailTrusted)
	})

	t.Run("microsoft uses the tenant-checked username", func(t *testing.T) {
		t.Parallel()

		identity := identityFromOIDCClaims(testIssuer, oidcClaims{
			Subject:           "aad|1",
			PreferredUsername: "Person@Carrier.example",
		}, tenant.SSOProviderAzureAD)
		assert.Equal(t, "person@carrier.example", identity.Email)
		assert.True(t, identity.EmailTrusted)
	})
}

func TestOIDCBool_AcceptsBooleanAndStringForms(t *testing.T) {
	t.Parallel()

	for raw, want := range map[string]bool{
		`true`: true, `"true"`: true, `false`: false, `"false"`: false, `null`: false, `"yes"`: false,
	} {
		var b oidcBool
		require.NoError(t, b.UnmarshalJSON([]byte(raw)))
		assert.Equal(t, want, b.IsTrue(), raw)
	}
}
