package deskbench

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeRouting(t *testing.T, dir, name, body string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	return path
}

func TestLoadRoutingCases_FillsTheDefaultsAndKeepsWhereEachCameFrom(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := writeRouting(t, dir, "billing.yaml", `
defaults:
  from: DispatchAssistant
cases:
  - name: past-due
    ask: "which invoices are past due"
    expect: BillingAssistant|Receivables
  - name: from-billing
    ask: "run the aging"
    from: BillingAssistant
    about: [customer]
    expect: current
`)

	cases, err := LoadRoutingCases([]string{dir})

	require.NoError(t, err)
	require.Len(t, cases, 2)
	assert.Equal(t, "DispatchAssistant", cases[0].From)
	assert.Equal(t, "BillingAssistant", cases[1].From)
	assert.Equal(t, []string{"customer"}, cases[1].About)
	assert.Equal(t, file, cases[0].Source)
}

func TestLoadRoutingCases_Refuses(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"an unknown field":         "cases:\n  - name: a\n    ask: b\n    expect: c\n    expected: d\n",
		"a case unnamed":           "cases:\n  - ask: b\n    expect: c\n",
		"a case without ask":       "cases:\n  - name: a\n    expect: c\n",
		"a case expecting nothing": "cases:\n  - name: a\n    ask: b\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			writeRouting(t, dir, "cases.yaml", body)

			_, err := LoadRoutingCases([]string{dir})

			require.Error(t, err)
		})
	}
}

func TestLoadRoutingCases_RefusesTheSameNameTwice(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeRouting(t, dir, "a.yaml", "cases:\n  - name: same\n    ask: b\n    expect: c\n")
	writeRouting(t, dir, "b.yaml", "cases:\n  - name: same\n    ask: d\n    expect: e\n")

	_, err := LoadRoutingCases([]string{dir})

	require.ErrorContains(t, err, `"same"`)
}

func TestRoutingPasses(t *testing.T) {
	t.Parallel()

	dispatch := &agentdefinition.Definition{
		ID:       pulid.MustNew("agdef_"),
		Name:     "Dispatch",
		Template: agentdefinition.TemplateDispatchAssistant,
	}
	billing := &agentdefinition.Definition{
		ID:       pulid.MustNew("agdef_"),
		Name:     "Billing",
		Template: agentdefinition.TemplateBillingAssistant,
	}
	better := &agentports.AgentMatch{Outcome: agentsuggestion.MatchBetterAgent, Agent: billing}
	compliance := &agentdefinition.Definition{
		ID:       pulid.MustNew("agdef_"),
		Name:     "Compliance",
		Template: agentdefinition.TemplateComplianceAssistant,
	}
	current := &agentports.AgentMatch{Outcome: agentsuggestion.MatchCurrentAgent, Agent: dispatch}
	delegated := &agentports.AgentMatch{
		Outcome:  agentsuggestion.MatchCurrentAgent,
		Agent:    dispatch,
		Delegate: compliance,
	}
	nobody := &agentports.AgentMatch{Outcome: agentsuggestion.MatchNoAgent}
	nothing := &agentports.AgentMatch{Outcome: agentsuggestion.MatchNone}

	cases := []struct {
		expect string
		found  *agentports.AgentMatch
		want   bool
	}{
		{"BillingAssistant", better, true},
		{"billing", better, true},
		{billing.ID.String(), better, true},
		{"Receivables|BillingAssistant", better, true},
		{"Receivables", better, false},
		{"current", better, false},
		{"current", current, true},
		{"DispatchAssistant", current, true},
		{"BillingAssistant", current, false},
		{"ComplianceAssistant", delegated, true},
		{"DispatchAssistant", delegated, true},
		{"current", delegated, true},
		{"BillingAssistant", delegated, false},
		{"none", nobody, true},
		{"nothing", nothing, true},
		{"none", nothing, false},
		{"BillingAssistant", nil, false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, RoutingPasses(tc.expect, dispatch, tc.found), tc.expect)
	}
}

func TestUnreachableExpectation_NamesTheAgentsNoEnabledChatAgentIs(t *testing.T) {
	t.Parallel()

	agents := []*agentdefinition.Definition{
		{Name: "Dispatch desk", Template: agentdefinition.TemplateDispatchAssistant},
		{Name: "Master data steward", Template: agentdefinition.TemplateMasterDataSteward},
	}

	assert.Equal(t, "no enabled chat agent is CustomerAssistant",
		UnreachableExpectation("CustomerAssistant", agents))
	assert.Equal(t, "no enabled chat agent is IntakeDesk or CustomerAssistant",
		UnreachableExpectation("IntakeDesk|CustomerAssistant", agents))
	assert.Empty(t, UnreachableExpectation("CustomerAssistant|MasterDataSteward", agents))
	assert.Empty(t, UnreachableExpectation("Dispatch desk", agents))
	assert.Empty(t, UnreachableExpectation("none|CustomerAssistant", agents))
	assert.Empty(t, UnreachableExpectation("current", agents))
}

func TestRoutingReport_SaysHowManyWentWhereTheyShouldAndWhatChanged(t *testing.T) {
	t.Parallel()

	baseline := BuildRoutingReport("abc", "admin@trenova.app", time.Unix(0, 0), []*RoutingResult{
		{Case: "past-due", Passed: false},
		{Case: "aging", Passed: true},
	})
	report := BuildRoutingReport("def", "admin@trenova.app", time.Unix(0, 0), []*RoutingResult{
		{
			Case:    "past-due",
			From:    "Dispatch",
			Expect:  "BillingAssistant",
			Outcome: "BetterAgent",
			Agent:   "Billing",
			Tools:   []string{"invoice_list"},
			Passed:  true,
		},
		{Case: "aging", From: "Billing", Expect: "current", Outcome: "NoMatch"},
	})

	summary := RenderRouting(report, baseline)

	assert.Equal(t, 0.5, report.Accuracy())
	assert.Contains(t, summary, "1 of 2 questions went where they should (50%) at def.")
	assert.Contains(t, summary, "The run compared with: 1 of 2 (50%), +0 points.")
	assert.Contains(t, summary, "- `aging` now fails")
	assert.Contains(t, summary, "- `past-due` now passes")
	assert.Contains(
		t,
		summary,
		"| ✓ past-due | Dispatch | BillingAssistant | Billing | invoice_list |",
	)
	assert.Contains(t, summary, "| ✗ aging | Billing | current | nothing |  |")

	dir := t.TempDir()
	require.NoError(t, report.Write(dir, baseline))
	loaded, err := LoadRoutingReport(dir)
	require.NoError(t, err)
	assert.Equal(t, report.Passed, loaded.Passed)
	assert.Len(t, loaded.Cases, 2)
}

func TestShippedRoutingCases_NameAgentsByTheirTemplates(t *testing.T) {
	t.Parallel()

	cases, err := LoadRoutingCases([]string{"../../deskbench/routing"})

	require.NoError(t, err)
	require.NotEmpty(t, cases)
	keywords := map[string]bool{
		RoutingExpectCurrent: true,
		RoutingExpectNone:    true,
		RoutingExpectNothing: true,
	}
	for _, routing := range cases {
		assert.True(t, agentdefinition.Template(routing.From).IsValid(),
			"%s asks from %q, which is not a template", routing.Name, routing.From)
		for _, option := range strings.Split(routing.Expect, "|") {
			option = strings.TrimSpace(option)
			assert.True(t, keywords[option] || agentdefinition.Template(option).IsValid(),
				"%s expects %q, which is neither a template nor current, none or nothing",
				routing.Name, option)
		}
	}
}

func TestProfileConfusion_CountsWhatEachCaseExpectedAgainstWhatItGot(t *testing.T) {
	t.Parallel()

	dispatch := &agentdefinition.Definition{
		ID: pulid.MustNew(
			"agdef_",
		), Name: "Dispatch", Template: agentdefinition.TemplateDispatchAssistant,
	}
	billing := &agentdefinition.Definition{
		ID: pulid.MustNew(
			"agdef_",
		), Name: "Billing", Template: agentdefinition.TemplateBillingAssistant,
	}
	kept := &agentports.AgentMatch{Outcome: agentsuggestion.MatchCurrentAgent, Agent: dispatch}
	moved := &agentports.AgentMatch{Outcome: agentsuggestion.MatchBetterAgent, Agent: billing}
	nobody := &agentports.AgentMatch{Outcome: agentsuggestion.MatchNoAgent}

	assert.Equal(t, "DispatchAssistant", SelectedProfile(dispatch, kept))
	assert.Equal(t, "BillingAssistant", SelectedProfile(dispatch, moved))
	assert.Equal(t, RoutingExpectNone, SelectedProfile(dispatch, nobody))
	assert.Equal(t, "DispatchAssistant", ExpectedProfile("current", dispatch))
	assert.Equal(t, "BillingAssistant", ExpectedProfile("BillingAssistant|Receivables", dispatch))
	assert.Equal(t, RoutingExpectNothing, ExpectedProfile("Nothing", dispatch))

	report := BuildRoutingReport("abc", "bench", time.Now(), []*RoutingResult{
		{
			Case:            "a",
			ExpectedProfile: "BillingAssistant",
			SelectedProfile: "BillingAssistant",
			Passed:          true,
		},
		{Case: "b", ExpectedProfile: "BillingAssistant", SelectedProfile: "DispatchAssistant"},
		{
			Case:            "c",
			ExpectedProfile: "DispatchAssistant",
			SelectedProfile: "DispatchAssistant",
			Passed:          true,
		},
		{Case: "d", ExpectedProfile: "none", SelectedProfile: "BillingAssistant"},
	})
	confusion := report.Confusion()

	assert.Equal(t, []string{"BillingAssistant", "DispatchAssistant", "none"}, confusion.Profiles)
	assert.Equal(t, 1, confusion.Counts["BillingAssistant"]["DispatchAssistant"])
	recall, cases := confusion.Recall("BillingAssistant")
	assert.InDelta(t, 0.5, recall, 1e-9)
	assert.Equal(t, 2, cases)
	precision, picked := confusion.Precision("BillingAssistant")
	assert.InDelta(t, 0.5, precision, 1e-9)
	assert.Equal(t, 2, picked)

	rendered := RenderRouting(report, nil)
	assert.Contains(t, rendered, "## Profile selection")
	assert.Contains(t, rendered, "| BillingAssistant | **1** | 1 | · | 50% of 2 |")
}
