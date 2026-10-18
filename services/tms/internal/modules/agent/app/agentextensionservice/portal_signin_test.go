package agentextensionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAuthenticatorKey = "JBSWY3DPEHPK3PXP"

func (f *fixture) siteRequest(
	mutate func(*agentextensionports.SavePortalSiteRequest),
) *agentextensionports.SavePortalSiteRequest {
	req := &agentextensionports.SavePortalSiteRequest{
		TenantInfo: f.tenant,
		Name:       "Acme DC",
		Domains:    []string{"portal.acme.com"},
		StartURL:   "https://portal.acme.com/login",
		Username:   "dispatch@trenova.test",
		Password:   "s3cret-pass",
		Enabled:    true,
	}
	if mutate != nil {
		mutate(req)
	}

	return req
}

func fieldsOf(t *testing.T, err error) []string {
	t.Helper()

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}

	return fields
}

func TestAnAuthenticatorKeyIsStoredEncryptedAndSentAsTheCodePlaceholder(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableBrowser(t)
	site, err := f.svc.CreateSite(t.Context(), f.siteRequest(
		func(req *agentextensionports.SavePortalSiteRequest) {
			req.AuthenticatorKey = "jbsw y3dp ehpk 3pxp"
		},
	))
	require.NoError(t, err)
	assert.True(t, site.HasAuthenticator)

	stored, err := f.sites.Get(t.Context(), f.tenant, site.ID)
	require.NoError(t, err)
	key := configspec.ReadString(stored.Credentials, agentextension.PortalCredentialAuthenticator)
	assert.NotEmpty(t, key)
	assert.NotContains(t, key, testAuthenticatorKey)

	f.browser.finishWith(finished("Open slots: 8:00 AM.", "https://portal.acme.com/slots"))
	_, err = f.svc.ReadPortal(t.Context(), &agentextensionports.PortalReadRequest{
		TenantInfo: f.tenant,
		SiteID:     site.ID,
		Question:   "Which dock appointments are open on October 20?",
	})
	require.NoError(t, err)

	require.Len(t, f.browser.started, 1)
	sent := f.browser.started[0]
	assert.Equal(t, testAuthenticatorKey, sent.Secrets[agentextension.PortalSecretAuthenticator],
		"the key, normalized, is what Browser Use turns into a fresh code")
	assert.Contains(t, sent.Instruction, "type portal_bu_2fa_code into the code box and submit it")
	assert.NotContains(t, sent.Instruction, testAuthenticatorKey)
}

func TestAnAuthenticatorKeyStaysUntilItIsCleared(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	site, err := f.svc.CreateSite(t.Context(), f.siteRequest(
		func(req *agentextensionports.SavePortalSiteRequest) {
			req.AuthenticatorKey = testAuthenticatorKey
		},
	))
	require.NoError(t, err)

	kept, err := f.svc.UpdateSite(t.Context(), f.siteRequest(
		func(req *agentextensionports.SavePortalSiteRequest) {
			req.ID = site.ID
			req.Username, req.Password = "", ""
			req.Version = site.Version
		},
	))
	require.NoError(t, err)
	assert.True(t, kept.HasSignIn)
	assert.True(t, kept.HasAuthenticator, "a key left blank keeps the stored one")

	cleared, err := f.svc.UpdateSite(t.Context(), f.siteRequest(
		func(req *agentextensionports.SavePortalSiteRequest) {
			req.ID = site.ID
			req.Username, req.Password = "", ""
			req.ClearAuthenticator = true
			req.Version = kept.Version
		},
	))
	require.NoError(t, err)
	assert.True(t, cleared.HasSignIn, "clearing the key keeps the username and password")
	assert.False(t, cleared.HasAuthenticator)
}

func TestAnAuthenticatorKeyIsRefusedWhenItCannotBeUsed(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	_, err := f.svc.CreateSite(t.Context(), f.siteRequest(
		func(req *agentextensionports.SavePortalSiteRequest) {
			req.AuthenticatorKey = "123456"
		},
	))
	assert.Equal(t, []string{"authenticatorKey"}, fieldsOf(t, err),
		"a six-digit code is not the setup key")

	_, err = f.svc.CreateSite(t.Context(), f.siteRequest(
		func(req *agentextensionports.SavePortalSiteRequest) {
			req.Username, req.Password = "", ""
			req.AuthenticatorKey = testAuthenticatorKey
		},
	))
	assert.Equal(t, []string{"authenticatorKey"}, fieldsOf(t, err),
		"a key is used with a sign-in")
}

func TestAPortalThatAsksForACodeSaysWhatToChange(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableBrowser(t)
	site := f.addSite(t, false)
	succeeded := false
	f.browser.finishWith(&agentextensionports.PortalTaskState{
		Phase:     agentextensionports.PortalTaskFinished,
		Succeeded: &succeeded,
		Answer: "The portal is asking for a two-factor authentication (2FA) code. Please " +
			"provide it.",
		VisitedURLs: []string{"https://portal.acme.com/login"},
	})

	_, err := f.svc.ReadPortal(t.Context(), &agentextensionports.PortalReadRequest{
		TenantInfo: f.tenant,
		SiteID:     site.ID,
		Question:   "Which dock appointments are open on October 20?",
	})
	require.True(t, IsToolError(err))
	assert.Contains(t, err.Error(), "Acme DC asked for a sign-in code the browser was not given")
	assert.Contains(t, err.Error(), "add the portal's authenticator key")

	run := f.runs.only(t)
	assert.Equal(t, agentextension.PortalRunStatusFailed, run.Status)
	assert.Contains(t, run.Failure, "authenticator key")
	assert.Equal(t, []string{"session-1"}, f.browser.closed,
		"a read no one can relay a code to closes its browser")
}

func TestTrenovaItselfCannotBeAPortalSite(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.svc.appHost = "cloud.trenova.app"

	_, err := f.svc.CreateSite(t.Context(), f.siteRequest(
		func(req *agentextensionports.SavePortalSiteRequest) {
			req.Domains = []string{"trenova.app"}
			req.StartURL = "https://cloud.trenova.app/login"
		},
	))
	assert.Contains(t, fieldsOf(t, err), "domains",
		"a site on a parent domain covers the app's own host too")

	f.enableBrowser(t)
	stored := &agentextension.PortalSite{
		ID:             pulid.MustNew(agentextension.PortalSiteIDPrefix),
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		Name:           "Trenova App",
		Domains:        []string{"cloud.trenova.app"},
		StartURL:       "https://cloud.trenova.app/login",
		Enabled:        true,
		AllowActions:   true,
		Credentials:    map[string]any{},
	}
	_, err = f.sites.Create(t.Context(), stored)
	require.NoError(t, err)

	_, err = f.svc.PlanPortalAction(t.Context(), &agentextensionports.PortalActionRequest{
		TenantInfo:  f.tenant,
		SiteID:      stored.ID,
		Instruction: "Post the comment Eric was here on SEED-SHP-003",
	})
	require.True(t, IsToolError(err), "a site saved before the rule is refused when used")
	assert.Contains(t, err.Error(), "Trenova itself cannot be a portal site")
	assert.Empty(t, f.browser.started)
}
