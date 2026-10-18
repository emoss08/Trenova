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
			"If the portal asks for a sign-in code you were not given (one sent by text message "+
			"or email, or a prompt to approve on a phone), stop there and begin your answer with "+
			"SIGN-IN CODE NEEDED, then say how it asked for the code.\n"+
			"Once you are signed in, stay signed in: check your work on the page you are on or "+
			"through the portal's own links, and never sign out, open the sign-in page again or "+
			"open the portal in a new tab, which can ask for the sign-in and its code again.\n"+
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
	assert.NotContains(t, acting, agentextension.PortalSecretAuthenticator,
		"a site without an authenticator key never names the placeholder")

	act.UsesAuthenticator = true
	assert.Contains(t, act.Task(),
		"If it asks for a code from an authenticator app, type portal_bu_2fa_code into the "+
			"code box and submit it in the same step, before anything else: each code lasts only "+
			"about thirty seconds. If the portal says the code is wrong or has expired, type "+
			"portal_bu_2fa_code again, which makes a new code, at most twice more.\n")
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

func TestNormalizeAuthenticatorKeyReadsTheKeyAsAPersonCopiesIt(t *testing.T) {
	t.Parallel()

	tests := []struct{ raw, want string }{
		{raw: "jbsw y3dp ehpk 3pxp", want: "JBSWY3DPEHPK3PXP"},
		{raw: "JBSW-Y3DP-EHPK-3PXP", want: "JBSWY3DPEHPK3PXP"},
		{raw: "JBSWY3DPEHPK3PXP====", want: "JBSWY3DPEHPK3PXP"},
		{
			raw:  "otpauth://totp/Trenova:eric?secret=JBSWY3DPEHPK3PXP&issuer=Trenova",
			want: "JBSWY3DPEHPK3PXP",
		},
		{raw: "   ", want: ""},
	}
	for _, tt := range tests {
		key, err := agentextension.NormalizeAuthenticatorKey(tt.raw)
		require.NoError(t, err, tt.raw)
		assert.Equal(t, tt.want, key, tt.raw)
	}

	for _, raw := range []string{"123456", "not base32 at all!", "JBSWY3DP", "otpauth://totp/x"} {
		_, err := agentextension.NormalizeAuthenticatorKey(raw)
		require.ErrorIs(t, err, agentextension.ErrAuthenticatorKeyInvalid, raw)
	}
}

func TestAsksForSignInCodeTellsABlockedSignInFromAnAnswer(t *testing.T) {
	t.Parallel()

	assert.True(t, agentextension.AsksForSignInCode(
		"SIGN-IN CODE NEEDED: the portal sent a code by text message.", true),
		"the marker counts whatever the vendor says about success")
	assert.True(t, agentextension.AsksForSignInCode(
		"The portal is asking for a two-factor authentication (2FA) code.", false))
	assert.True(t, agentextension.AsksForSignInCode(
		"I need the 6-digit code from an authenticator app to continue.", false))
	assert.False(t, agentextension.AsksForSignInCode(
		"Two-factor authentication is turned on for this account.", true),
		"a finished lookup that mentions two-factor settings was not blocked")
	assert.False(t, agentextension.AsksForSignInCode("The slot was taken.", false))
}

func TestPortalDetailsRefuseEverySignInPlaceholder(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"portal_bu_2fa_code", "my_bu_2fa_code", "Portal_Password", "portal_sign_in_code",
	} {
		_, err := agentextension.NormalizePortalDetails(map[string]string{name: "x"})
		require.Error(t, err, name)
	}
}

func TestOnlyAStoredKeyIsAnAuthenticator(t *testing.T) {
	t.Parallel()

	site := &agentextension.PortalSite{Credentials: map[string]any{}}
	assert.False(t, site.HasAuthenticator())
	site.Credentials[agentextension.PortalCredentialAuthenticator] = "enc:key"
	assert.True(t, site.HasAuthenticator())

	var none *agentextension.PortalSite
	assert.False(t, none.HasAuthenticator())
}

func TestNormalizeSignInCodeReadsACodeAsAPersonTypesIt(t *testing.T) {
	t.Parallel()

	tests := []struct{ raw, want string }{
		{raw: "123456", want: "123456"},
		{raw: " 123 456 ", want: "123456"},
		{raw: "123-456", want: "123456"},
		{raw: "Ab12Cd", want: "Ab12Cd"},
	}
	for _, tt := range tests {
		code, err := agentextension.NormalizeSignInCode(tt.raw)
		require.NoError(t, err, tt.raw)
		assert.Equal(t, tt.want, code, tt.raw)
	}

	refused := []string{"", "123", "1234567890123", "12!456", "12\n3456", "١٢٣٤٥٦"}
	for _, raw := range refused {
		_, err := agentextension.NormalizeSignInCode(raw)
		require.ErrorIs(t, err, agentextension.ErrSignInCodeInvalid, raw)
	}
}

func TestContinueWithCodeEntersThePlaceholderAndCarriesOnWithTheTask(t *testing.T) {
	t.Parallel()

	task := "On Acme DC, find out: Which slots are open?"
	assert.Equal(t, "The portal is showing where to enter the sign-in code it sent. Enter "+
		"portal_sign_in_code there and submit it straight away, in the same step. Then carry "+
		"on with the task below from where you stopped, without repeating anything already "+
		"done. Stay signed in: do not reload the sign-in page, sign out or open the portal in "+
		"a new tab. If the portal says the code is wrong or has expired, stop there and begin "+
		"your answer with SIGN-IN CODE NEEDED, then say what it said.\n\n"+task,
		agentextension.ContinueWithCode(task))
}

func TestContinueAfterSignInCarriesOnWithoutSigningInAgain(t *testing.T) {
	t.Parallel()

	task := "On Acme DC, find out: Which slots are open?"
	assert.Equal(t, "A person has signed in to the portal in this browser themselves. Carry "+
		"on with the task below from the page the browser is on, without repeating anything "+
		"already done. Stay signed in: do not reload the sign-in page, sign out or open the "+
		"portal in a new tab. If the portal asks for a sign-in code again, stop there and "+
		"begin your answer with SIGN-IN CODE NEEDED, then say how it asked for the code.\n\n"+
		task, agentextension.ContinueAfterSignIn(task))
}

func TestAPortalRunWaitingOnACodeKeepsItsProgressUntilACodeContinuesIt(t *testing.T) {
	t.Parallel()

	run := &agentextension.PortalRun{Status: agentextension.PortalRunStatusRunning}
	assert.Nil(t, run.CodeExpiresAt())
	assert.False(t, run.Continued())

	run.AwaitCode(&agentextension.PortalRunOutcome{
		Answer:      "  SIGN-IN CODE NEEDED: " + strings.Repeat("a", 600),
		VisitedURLs: []string{"https://portal.acme.com/login"},
		Steps:       2,
	}, 1_790_000_000)

	assert.Equal(t, agentextension.PortalRunStatusAwaitingCode, run.Status)
	assert.False(t, run.Status.Finished(), "a run waiting on a code has not ended")
	assert.Len(t, []rune(run.CodePrompt), agentextension.MaxPortalCodePromptRunes)
	assert.True(t, strings.HasPrefix(run.CodePrompt, "SIGN-IN CODE NEEDED"))
	assert.Equal(t, []string{"https://portal.acme.com/login"}, run.VisitedURLs)
	assert.Equal(t, []string{}, run.OffSiteURLs)
	assert.Equal(t, 2, run.Steps)
	require.NotNil(t, run.CodeExpiresAt())
	assert.Equal(t, int64(1_790_000_000+agentextension.PortalCodeWaitSeconds),
		*run.CodeExpiresAt())
	assert.Nil(t, run.FinishedAt)

	run.ContinuedWith("task-2")
	assert.Equal(t, agentextension.PortalRunStatusRunning, run.Status)
	assert.Equal(t, "task-2", run.VendorTaskID)
	assert.Equal(t, 1, run.CodeAttempts)
	assert.True(t, run.Continued())
	assert.Empty(t, run.CodePrompt)
	assert.Nil(t, run.CodeRequestedAt)
	assert.Nil(t, run.CodeExpiresAt())
	assert.Equal(t, 2, run.Steps, "what it did before the code stays on the run")
}
