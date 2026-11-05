package extensionportalservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentextensionservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionportal"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const lookUpInstruction = "Find the open dock appointments on October 20"

func (f *fixture) lookUp(
	site *agentextensionports.PortalSiteView,
) *agentextensionports.PortalActionRequest {
	return &agentextensionports.PortalActionRequest{
		TenantInfo:     f.tenant,
		SiteID:         site.ID,
		Instruction:    lookUpInstruction,
		IdempotencyKey: "approval-1",
		LookUpOnly:     true,
		Owner:          f.person(),
	}
}

func (f *fixture) readOf(
	site *agentextensionports.PortalSiteView,
) *agentextensionports.PortalReadRequest {
	return &agentextensionports.PortalReadRequest{
		TenantInfo: f.tenant,
		SiteID:     site.ID,
		Question:   "Which dock appointments are open on October 20?",
	}
}

func TestReadPortalRefusesASignInWhoseAccountMayChangeThings(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableBrowser(t)
	site, err := f.svc.CreateSite(t.Context(), f.siteRequest(nil))
	require.NoError(t, err)

	_, err = f.svc.ReadPortal(t.Context(), f.readOf(site))

	var refusal *agentextensionservice.ToolError
	require.ErrorAs(t, err, &refusal)
	assert.Contains(t, refusal.Message, "web_use_portal")
	assert.Contains(t, refusal.Message, "lookUpOnly")
	assert.Empty(t, f.browser.started)
	assert.Zero(t, f.usage.Failures, "a refused read reserves nothing")
}

func TestReadPortalRunsOnAViewOnlyAccount(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableBrowser(t)
	site, err := f.svc.CreateSite(t.Context(), f.siteRequest(
		func(req *agentextensionports.SavePortalSiteRequest) { req.ViewOnlyAccount = true },
	))
	require.NoError(t, err)
	f.browser.finishWith(finished("Open slots: 8:00 AM.", "https://portal.acme.com/slots"))

	outcome, err := f.svc.ReadPortal(t.Context(), f.readOf(site))

	require.NoError(t, err)
	assert.Equal(t, "Open slots: 8:00 AM.", outcome.Answer)
	assert.True(t, outcome.Site.ViewOnlyAccount)
	assert.False(t, outcome.Site.ReadNeedsAPerson)
}

func TestReadPortalRunsOnASiteWithoutASignIn(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableBrowser(t)
	site, err := f.svc.CreateSite(t.Context(), f.siteRequest(
		func(req *agentextensionports.SavePortalSiteRequest) {
			req.Username = ""
			req.Password = ""
		},
	))
	require.NoError(t, err)
	f.browser.finishWith(finished("Open slots: 8:00 AM.", "https://portal.acme.com/slots"))

	_, err = f.svc.ReadPortal(t.Context(), f.readOf(site))

	require.NoError(t, err)
}

func TestReadPortalRefusesABrowserProfileThatHoldsASignedInSession(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableBrowser(t)
	site, err := f.svc.CreateSite(t.Context(), f.siteRequest(
		func(req *agentextensionports.SavePortalSiteRequest) {
			req.Username = ""
			req.Password = ""
			req.BrowserProfileID = "profile-1"
		},
	))
	require.NoError(t, err)

	_, err = f.svc.ReadPortal(t.Context(), f.readOf(site))

	require.True(t, agentextensionservice.IsToolError(err))
	assert.Empty(t, f.browser.started)
}

func TestSavingASiteKeepsItsViewOnlyAccount(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	created, err := f.svc.CreateSite(t.Context(), f.siteRequest(
		func(req *agentextensionports.SavePortalSiteRequest) { req.ViewOnlyAccount = true },
	))
	require.NoError(t, err)
	assert.True(t, created.ViewOnlyAccount)

	updated, err := f.svc.UpdateSite(t.Context(), f.siteRequest(
		func(req *agentextensionports.SavePortalSiteRequest) {
			req.ID = created.ID
			req.Version = created.Version
			req.Username = ""
			req.Password = ""
		},
	))
	require.NoError(t, err)
	assert.False(t, updated.ViewOnlyAccount)
}

func TestLookUpOnlyPlansTheReadTaskOnAReadOnlySite(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableBrowser(t)
	site := f.addSite(t, false)

	plan, err := f.svc.PlanPortalAction(t.Context(), f.lookUp(site))

	require.NoError(t, err)
	assert.Contains(t, plan.SentTask, "find out: "+lookUpInstruction)
	assert.Contains(t, plan.SentTask, "Only look things up.")
	assert.NotContains(t, plan.SentTask, "do this:")
}

func TestAnActionIsStillRefusedOnAReadOnlySite(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableBrowser(t)
	site := f.addSite(t, false)
	req := f.lookUp(site)
	req.LookUpOnly = false

	_, err := f.svc.PlanPortalAction(t.Context(), req)

	require.True(t, agentextensionservice.IsToolError(err))
	assert.Contains(t, err.Error(), "Agents may only read")
}

func TestLookUpOnlyStartsARunInReadMode(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.enableBrowser(t)
	site := f.addSite(t, false)

	started, err := f.svc.StartPortalAction(t.Context(), f.lookUp(site))

	require.NoError(t, err)
	run := f.runs.only(t)
	assert.Equal(t, started.RunID, run.ID)
	assert.Equal(t, extensionportal.PortalRunModeRead, run.Mode)
	assert.Contains(t, run.Instruction, "Only look things up.")
	require.Len(t, f.browser.started, 1)
}
