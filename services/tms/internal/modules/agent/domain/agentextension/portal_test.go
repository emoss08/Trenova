package agentextension_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostAllowedCoversTheSiteAndItsSubdomainsOnly(t *testing.T) {
	t.Parallel()

	domains := []string{"portal.acme.com", "*.dockhub.io"}
	assert.True(t, agentextension.HostAllowed("portal.acme.com", domains))
	assert.True(t, agentextension.HostAllowed("login.portal.acme.com", domains))
	assert.True(t, agentextension.HostAllowed("DockHub.io.", domains))
	assert.True(t, agentextension.HostAllowed("eu.dockhub.io", domains))
	assert.False(t, agentextension.HostAllowed("acme.com", domains))
	assert.False(t, agentextension.HostAllowed("portal.acme.com.evil.net", domains))
	assert.False(t, agentextension.HostAllowed("notdockhub.io", domains))
	assert.False(t, agentextension.HostAllowed("", domains))
}

func TestOffSiteListsEveryPageOutsideTheAddressesOnce(t *testing.T) {
	t.Parallel()

	visited := []string{
		"about:blank",
		"https://portal.acme.com/login",
		"https://attacker.example/collect?po=4512",
		"https://portal.acme.com/appointments",
		"https://attacker.example/collect?po=4512",
		"javascript:alert(1)",
		"",
	}
	assert.Equal(t,
		[]string{"https://attacker.example/collect?po=4512", "javascript:alert(1)"},
		agentextension.OffSite(visited, []string{"portal.acme.com"}),
	)
}

func validSite() *agentextension.PortalSite {
	return &agentextension.PortalSite{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           " Acme DC ",
		Domains:        []string{"Portal.Acme.com", "portal.acme.com", " sso.acme.com. "},
		StartURL:       "https://portal.acme.com/login",
	}
}

func TestPortalSiteNormalizesAndAcceptsAWellFormedSite(t *testing.T) {
	t.Parallel()

	site := validSite()
	site.Normalize()
	multiErr := errortypes.NewMultiError()
	site.Validate(multiErr)

	require.False(t, multiErr.HasErrors(), multiErr.Error())
	assert.Equal(t, "Acme DC", site.Name)
	assert.Equal(t, []string{"portal.acme.com", "sso.acme.com"}, site.Domains)
}

func TestPortalSiteRefusesAStartPageOffItsAddresses(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"not https":        "http://portal.acme.com/login",
		"another host":     "https://evil.example/login",
		"credentials":      "https://user:pass@portal.acme.com/login",
		"not a url at all": "portal.acme.com",
	}
	for name, start := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			site := validSite()
			site.StartURL = start
			site.Normalize()
			multiErr := errortypes.NewMultiError()
			site.Validate(multiErr)

			require.True(t, multiErr.HasErrors())
			assert.Equal(t, "startUrl", multiErr.Errors[0].Field)
		})
	}
}

func TestPortalSiteRefusesAnAddressThatIsNotAHostname(t *testing.T) {
	t.Parallel()

	site := validSite()
	site.Domains = []string{"https://portal.acme.com/path"}
	site.Normalize()
	multiErr := errortypes.NewMultiError()
	site.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.Contains(t, fields, "domains")
}

func TestCheckPortalInstructionRefusesTrenovaRecordIDs(t *testing.T) {
	t.Parallel()

	require.NoError(t, agentextension.CheckPortalInstruction(
		"Find the open dock appointments on October 20 for PO 4512"))
	require.ErrorIs(t,
		agentextension.CheckPortalInstruction("Find shp_01J8ZK3M4N5P6Q7R8S9T0V1W2X on the portal"),
		agentextension.ErrPortalInstructionHasRecordID,
	)
	require.ErrorIs(t, agentextension.CheckPortalInstruction("book"),
		agentextension.ErrPortalInstructionTooShort)
	require.ErrorIs(t,
		agentextension.CheckPortalInstruction(strings.Repeat("a", agentextension.MaxPortalInstructionRunes+1)),
		agentextension.ErrPortalInstructionTooLong,
	)
}

func TestNormalizePortalDetailsKeepsTextAndRefusesWhatCannotBeSent(t *testing.T) {
	t.Parallel()

	details, err := agentextension.NormalizePortalDetails(map[string]string{
		" PO number ": " 4512 ",
		"Empty":       "  ",
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"PO number": "4512"}, details)

	_, err = agentextension.NormalizePortalDetails(map[string]string{"portal_password": "x"})
	require.Error(t, err)

	_, err = agentextension.NormalizePortalDetails(map[string]string{
		"Shipment": "shp_01J8ZK3M4N5P6Q7R8S9T0V1W2X",
	})
	require.Error(t, err)

	many := make(map[string]string, agentextension.MaxPortalDetails+1)
	for idx := range agentextension.MaxPortalDetails + 1 {
		many[strings.Repeat("k", idx+1)] = "v"
	}
	_, err = agentextension.NormalizePortalDetails(many)
	require.Error(t, err)
}

func TestPortalDetailsFromTakesOnlyText(t *testing.T) {
	t.Parallel()

	details, err := agentextension.PortalDetailsFrom(map[string]any{
		"PO":      float64(4512),
		"Carrier": "Ridgeline",
		"Hazmat":  false,
		"Unset":   nil,
	})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"PO": "4512", "Carrier": "Ridgeline", "Hazmat": "false"},
		details)

	_, err = agentextension.PortalDetailsFrom(map[string]any{"Nested": map[string]any{}})
	require.Error(t, err)
	_, err = agentextension.PortalDetailsFrom("PO 4512")
	require.Error(t, err)

	none, err := agentextension.PortalDetailsFrom(nil)
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestPortalTaskTextSaysWhatTheBrowserMayDo(t *testing.T) {
	t.Parallel()

	read := &agentextension.PortalTaskText{
		SiteName:    "Acme DC",
		SitePurpose: "dock appointments",
		Domains:     []string{"sso.acme.com", "portal.acme.com"},
		Mode:        agentextension.PortalRunModeRead,
		Instruction: "Which slots are open on October 20?",
		Details:     map[string]string{"Warehouse": "Dallas", "PO number": "4512"},
	}
	task := read.Task()
	assert.Equal(t,
		"On the Acme DC portal (dock appointments), find out: Which slots are open on October 20?\n"+
			"Details to use:\n- PO number: 4512\n- Warehouse: Dallas\n"+
			"Only look things up. Do not submit, book, upload, accept, cancel, change or delete "+
			"anything. When you have the answer, finish with it in plain words, with reference "+
			"numbers, dates and times exactly as the portal shows them. If the portal does not "+
			"show it, say so.",
		task,
	)
	assert.NotContains(t, task, agentextension.PortalSecretPassword)
	assert.Contains(t, read.Guidance(), "Stay on portal.acme.com, sso.acme.com;")

	act := *read
	act.Mode = agentextension.PortalRunModeAct
	act.UsesSignIn = true
	acting := act.Task()
	assert.Contains(t, acting, ", do this: Which slots")
	assert.Contains(t, acting, "use the username portal_username and the password portal_password")
	assert.Contains(t, acting, "Do exactly this and nothing more.")
	assert.NotContains(t, acting, "Only look things up")
}

func TestParseBrowserUseSettingsDefaultsAndRefusals(t *testing.T) {
	t.Parallel()

	settings, err := agentextension.ParseBrowserUseSettings(map[string]string{"apiKey": " bu_key "})
	require.NoError(t, err)
	assert.Equal(t, "bu_key", settings.APIKey)
	assert.Equal(t, agentextension.BrowserUseModelDefault, settings.Model)
	assert.Equal(t, agentextension.ProxyCountryUS, settings.ProxyCountry)
	assert.Equal(t, agentextension.DefaultPortalMaxSteps, settings.MaxSteps)
	assert.Equal(t, agentextension.DefaultPortalDailyTasks, settings.DailyRequestLimit)
	assert.True(t, settings.RecordSessions)
	assert.False(t, settings.AllowScreenshots)

	_, err = agentextension.ParseBrowserUseSettings(map[string]string{
		"browserModel":     "gpt-2",
		"proxyCountry":     "zz",
		"maxSteps":         "500",
		"recordSessions":   "sometimes",
		"allowScreenshots": "yes please",
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.ElementsMatch(t, []string{
		"configuration.browserModel",
		"configuration.proxyCountry",
		"configuration.maxSteps",
		"configuration.recordSessions",
		"configuration.allowScreenshots",
	}, fields)
}

func TestOnlyReadingAPortalReturnsOutsideContent(t *testing.T) {
	t.Parallel()

	assert.True(t, agentextension.ReturnsExternalContent(agentextension.ToolReadPortal))
	assert.False(t, agentextension.ReturnsExternalContent(agentextension.ToolActOnPortal))
	assert.False(t, agentextension.ReturnsExternalContent(agentextension.ToolListPortalSites))

	spec, ok := agentextension.SpecFor(agentextension.TypeBrowserUse)
	require.True(t, ok)
	assert.Equal(t, agentextension.CapabilityPortalAutomation, spec.Capability)
	assert.Equal(t, agentextension.DefaultPortalDailyTasks, spec.DailyLimit(nil, errortypes.NewMultiError()))
}

func TestPortalRunFinishRecordsTheOutcome(t *testing.T) {
	t.Parallel()

	run := &agentextension.PortalRun{Status: agentextension.PortalRunStatusRunning}
	run.Finish(&agentextension.PortalRunOutcome{
		Status:      agentextension.PortalRunStatusSucceeded,
		Answer:      strings.Repeat("a", agentextension.MaxPortalAnswerRunes+10),
		VisitedURLs: nil,
		Steps:       7,
	}, 1_790_000_000)

	assert.Equal(t, agentextension.PortalRunStatusSucceeded, run.Status)
	assert.Len(t, []rune(run.Answer), agentextension.MaxPortalAnswerRunes)
	assert.Equal(t, []string{}, run.VisitedURLs)
	assert.Equal(t, 7, run.Steps)
	require.NotNil(t, run.FinishedAt)
	assert.True(t, run.Status.Finished())
	assert.False(t, agentextension.PortalRunStatusRunning.Finished())
}
