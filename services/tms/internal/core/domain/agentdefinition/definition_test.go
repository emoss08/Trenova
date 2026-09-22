package agentdefinition_test

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validDefinition() *agentdefinition.Definition {
	d := &agentdefinition.Definition{
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		Name:            "Night dispatch helper",
		Instructions:    "You help the night dispatch desk. Check hours of service first.",
		AutonomyCeiling: agent.TierPropose,
		ToolNames:       []string{"get_shipment", "flag_for_manual_review"},
		TriggerMode:     agentdefinition.TriggerChat,
		Enabled:         true,
	}
	d.ApplyDefaults()

	return d
}

func fieldErrors(t *testing.T, d *agentdefinition.Definition) map[string]bool {
	t.Helper()

	multiErr := errortypes.NewMultiError()
	d.Validate(multiErr)

	fields := make(map[string]bool)
	for _, e := range multiErr.Errors {
		fields[e.Field] = true
	}

	return fields
}

func TestValidate_AcceptsAWellFormedDefinition(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	validDefinition().Validate(multiErr)
	require.False(t, multiErr.HasErrors(), "unexpected errors: %v", multiErr.Errors)
}

func TestValidate_AcceptsADefinitionWithoutATemplate(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.Template = ""

	assert.Empty(t, fieldErrors(t, d))
}

func TestValidate_RejectsAnUnknownTemplate(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.Template = agentdefinition.Template("Bogus")

	assert.True(t, fieldErrors(t, d)["template"])
}

func TestValidate_BoundsInstructionsAndGuardrails(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.Instructions = strings.Repeat("a", 20001)
	assert.True(t, fieldErrors(t, d)["instructions"])

	d = validDefinition()
	d.Guardrails = []string{"Never quote a rate", "   "}
	assert.True(t, fieldErrors(t, d)["guardrails[1]"], "a blank guardrail is reported by index")

	d = validDefinition()
	d.Guardrails = []string{strings.Repeat("x", 301)}
	assert.True(t, fieldErrors(t, d)["guardrails[0]"])

	d = validDefinition()
	d.Guardrails = make([]string, 21)
	for i := range d.Guardrails {
		d.Guardrails[i] = "rule"
	}
	assert.True(t, fieldErrors(t, d)["guardrails"])
}

func TestValidate_AllowsToolsOnAnyTemplate(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.Template = agentdefinition.TemplateGeneralAssistant
	d.ToolNames = []string{"correct_charge_code"}

	assert.False(t, fieldErrors(t, d)["toolNames"],
		"a template is a starting point, not a bound on what an organization may enable")
}

func TestValidate_RejectsDuplicateAndEmptyTools(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.ToolNames = []string{"a", "a", "  "}

	errs := fieldErrors(t, d)
	assert.True(t, errs["toolNames[1]"], "duplicate tool should be reported")
	assert.True(t, errs["toolNames[2]"], "empty tool name should be reported")
}

func TestValidate_RejectsTooManyTools(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.ToolNames = make([]string, 0, 65)
	for i := range 65 {
		d.ToolNames = append(d.ToolNames, string(rune('a'+i%26))+string(rune('0'+i/26)))
	}

	assert.True(t, fieldErrors(t, d)["toolNames"])
}

func TestValidate_RejectsToolTiersForToolsNotEnabled(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.ToolTiers = map[string]agent.AutonomyTier{
		"get_shipment": agent.TierAutoExecute,
		"assign_move":  agent.TierPropose,
	}

	errs := fieldErrors(t, d)
	assert.True(t, errs["toolTiers.assign_move"], "a tier for a tool the agent cannot use is a mistake")
	assert.False(t, errs["toolTiers.get_shipment"])

	d = validDefinition()
	d.ToolTiers = map[string]agent.AutonomyTier{"get_shipment": agent.AutonomyTier("Whenever")}
	assert.True(t, fieldErrors(t, d)["toolTiers.get_shipment"])
}

func TestValidate_ScheduledNeedsAValidCronAndTimezone(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.TriggerMode = agentdefinition.TriggerScheduled
	assert.True(t, fieldErrors(t, d)["cronExpression"], "a schedule with no cron cannot fire")

	d.CronExpression = "every day at noon"
	assert.True(t, fieldErrors(t, d)["cronExpression"])

	d.CronExpression = "0 6 * * 1-5"
	d.CronTimezone = "Mars/Olympus"
	assert.True(t, fieldErrors(t, d)["cronTimezone"])

	d.CronTimezone = "America/Chicago"
	assert.Empty(t, fieldErrors(t, d))
}

func TestValidate_EventNeedsAtLeastOneKnownEvent(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.TriggerMode = agentdefinition.TriggerEvent
	assert.True(t, fieldErrors(t, d)["eventKinds"])

	d.EventKinds = []agent.EventKind{"shipment.teleported"}
	assert.True(t, fieldErrors(t, d)["eventKinds[0]"])

	d.EventKinds = []agent.EventKind{agent.EventBillingQueueItemException}
	assert.Empty(t, fieldErrors(t, d))
}

func TestValidate_ContinuousNeedsAnIntervalOfAtLeastAMinute(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.TriggerMode = agentdefinition.TriggerContinuous
	d.IntervalSeconds = 30
	assert.True(t, fieldErrors(t, d)["intervalSeconds"])

	d.IntervalSeconds = 300
	assert.Empty(t, fieldErrors(t, d))
}

func TestValidate_BoundsRunLimits(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.DecisionTimeoutSeconds = 10
	assert.True(t, fieldErrors(t, d)["decisionTimeoutSeconds"])

	d = validDefinition()
	d.MaxToolCalls = 0
	assert.True(t, fieldErrors(t, d)["maxToolCalls"])

	d = validDefinition()
	d.MaxToolCalls = 65
	assert.True(t, fieldErrors(t, d)["maxToolCalls"])

	d = validDefinition()
	d.MaxConcurrentRuns = 0
	assert.True(t, fieldErrors(t, d)["maxConcurrentRuns"])

	d = validDefinition()
	d.RunTimeoutSeconds = 5
	assert.True(t, fieldErrors(t, d)["runTimeoutSeconds"])

	d = validDefinition()
	d.ContextProviders = []agentdefinition.ContextProvider{"Weather"}
	assert.True(t, fieldErrors(t, d)["contextProviders[0]"])

	d = validDefinition()
	d.OutputMode = agentdefinition.OutputMode("Poem")
	assert.True(t, fieldErrors(t, d)["outputMode"])
}

func TestApplyDefaults_FillsWhatAnOlderRowOrRequestLeftUnset(t *testing.T) {
	t.Parallel()

	d := &agentdefinition.Definition{}
	d.ApplyDefaults()

	assert.Equal(t, agentdefinition.TriggerChat, d.TriggerMode)
	assert.Equal(t, agentdefinition.OutputConversational, d.OutputMode)
	assert.Equal(t, agentdefinition.DefaultDecisionTimeoutSeconds, d.DecisionTimeoutSeconds)
	assert.Equal(t, agentdefinition.DefaultMaxToolCalls, d.MaxToolCalls)
	assert.Equal(t, agentdefinition.DefaultRunTimeoutSeconds, d.RunTimeoutSeconds)
	assert.Equal(t, 1, d.MaxConcurrentRuns)
	assert.Equal(t, "UTC", d.CronTimezone)
}

// The ceiling exists to restrict. A per-tool tier lets an organization choose how
// much it trusts each tool, but nothing it writes can lift a tool above the
// ceiling.
func TestEffectiveTier_HonoursOverridesUnderTheCeiling(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		ceiling  agent.AutonomyTier
		override agent.AutonomyTier
		toolTier agent.AutonomyTier
		want     agent.AutonomyTier
	}{
		{"ceiling lowers auto-execute", agent.TierPropose, "", agent.TierAutoExecute, agent.TierPropose},
		{"ceiling lowers approval tier", agent.TierPropose, "", agent.TierActWithApproval, agent.TierPropose},
		{"tool default kept when it is under the ceiling", agent.TierAutoExecute, "", agent.TierPropose, agent.TierPropose},
		{"override lowers a tool", agent.TierAutoExecute, agent.TierPropose, agent.TierAutoExecute, agent.TierPropose},
		{"override raises a tool up to the ceiling", agent.TierAutoExecute, agent.TierAutoExecute, agent.TierPropose, agent.TierAutoExecute},
		{"override cannot pass the ceiling", agent.TierActWithApproval, agent.TierAutoExecute, agent.TierPropose, agent.TierActWithApproval},
		{"equal tiers are unchanged", agent.TierActWithApproval, "", agent.TierActWithApproval, agent.TierActWithApproval},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := validDefinition()
			d.AutonomyCeiling = tc.ceiling
			if tc.override != "" {
				d.ToolTiers = map[string]agent.AutonomyTier{"flag_for_manual_review": tc.override}
			}

			assert.Equal(t, tc.want, d.EffectiveTier("flag_for_manual_review", tc.toolTier))
		})
	}
}

func TestAllowsTool(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	assert.True(t, d.AllowsTool("flag_for_manual_review"))
	assert.True(t, d.AllowsTool("get_shipment"), "read tools are enabled the same way write tools are")
	assert.False(t, d.AllowsTool("correct_charge_code"))
}

func TestIsSystem_And_EffectiveShadow(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	assert.False(t, d.IsSystem())
	d.SystemKey = "billing_exception"
	assert.True(t, d.IsSystem())

	assert.False(t, d.EffectiveShadow(false))
	assert.True(t, d.EffectiveShadow(true), "an organization-wide pause wins")
	d.ShadowMode = true
	assert.True(t, d.EffectiveShadow(false))
}

func TestComputeNextRun_FollowsTheCronInTheDefinitionsTimezone(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.TriggerMode = agentdefinition.TriggerScheduled
	d.CronExpression = "0 6 * * *"
	d.CronTimezone = "America/Chicago"

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	from := time.Date(2026, time.September, 16, 7, 0, 0, 0, chicago)

	next, err := d.ComputeNextRun(from.Unix())
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, time.September, 17, 6, 0, 0, 0, chicago).Unix(), next)
}

func TestComputeNextRun_ContinuousAddsTheInterval(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.TriggerMode = agentdefinition.TriggerContinuous
	d.IntervalSeconds = 600

	next, err := d.ComputeNextRun(1_000)
	require.NoError(t, err)
	assert.Equal(t, int64(1_600), next)
}

func TestIsDue(t *testing.T) {
	t.Parallel()

	d := validDefinition()
	d.TriggerMode = agentdefinition.TriggerScheduled
	assert.False(t, d.IsDue(500), "no next run recorded means nothing is due yet")

	next := int64(400)
	d.NextRunAt = &next
	assert.True(t, d.IsDue(500))
	assert.False(t, d.IsDue(300))

	ends := int64(450)
	d.EndsAt = &ends
	assert.False(t, d.IsDue(500), "a definition past its end never fires")

	d.Enabled = false
	d.EndsAt = nil
	assert.False(t, d.IsDue(500))

	chat := validDefinition()
	chat.NextRunAt = &next
	assert.False(t, chat.IsDue(500), "a chat agent is never due")
}

func TestTemplates_DescribeEveryStarter(t *testing.T) {
	t.Parallel()

	for _, template := range agentdefinition.AllTemplates() {
		assert.True(t, template.IsValid())
		assert.NotEmpty(t, template.Label())
		assert.NotEmpty(t, template.Description())
		assert.NotEmpty(t, template.StarterInstructions())
		assert.True(t, template.StarterTrigger().IsValid())
	}

	assert.Equal(t, agentdefinition.TriggerEvent, agentdefinition.TemplateBillingException.StarterTrigger())
	assert.Contains(t, agentdefinition.TemplateBillingException.StarterEvents(), agent.EventBillingQueueItemException)
	assert.Empty(t, agentdefinition.TemplateGeneralAssistant.StarterTools())
}

/*
A template's starter tools have to be able to answer the question the template
exists for.

The seeded "Compliance desk" — described as answering "what is expiring", with
instructions saying medical cards are the thing most often missed — shipped
holding get_worker and search_worker and nothing else. Asked which drivers had
a medical card expiring within thirty days, it correctly reported that it could
not filter on dates, said out loud "I don't see list_expiring_credentials in my
available tools", and then answered anyway from arithmetic it cannot do. It got
the answer backwards.

The seeds now derive from these lists rather than carrying their own copy, so
this test is what keeps the lists honest as the catalog grows.
*/
func TestTemplates_CarryTheToolsTheirPurposeRequires(t *testing.T) {
	t.Parallel()

	assert.Contains(t, agentdefinition.TemplateComplianceAssistant.StarterTools(),
		"list_expiring_credentials",
		"a compliance agent that cannot filter credentials by date cannot do its job")

	assert.Contains(t, agentdefinition.TemplateDispatchAssistant.StarterTools(),
		"list_expiring_credentials",
		"dispatch decides who is legal to send")
	assert.Contains(t, agentdefinition.TemplateDispatchAssistant.StarterTools(),
		"list_time_off",
		"dispatch decides who is available")

	assert.Contains(t, agentdefinition.TemplateBillingAssistant.StarterTools(),
		"list_shipments")
}

// A duplicate name is silently dropped by the registry, and an empty one fails
// validation, so a starter list carrying either is a definition nobody can save.
func TestTemplates_StarterToolsAreWellFormed(t *testing.T) {
	t.Parallel()

	for _, template := range agentdefinition.AllTemplates() {
		seen := make(map[string]struct{})
		for _, tool := range template.StarterTools() {
			assert.NotEmpty(t, tool, template.Label())
			_, duplicate := seen[tool]
			assert.False(t, duplicate, "%s lists %q twice", template.Label(), tool)
			seen[tool] = struct{}{}
		}
	}
}

/*
No per-tool cap is an empty map, not an absent one.

tool_daily_limits was added as NOT NULL DEFAULT '{}' while its bun tag said
nullzero, and nullzero writes SQL NULL for a nil map. So every insert of an
agent with no per-tool cap — which is every system agent — was rejected by the
constraint, on Postgres as much as on SQLite. The full seed run had been red
on master since, and a fresh organization could not be given its agents.

Its older sibling tool_tiers is nullable, which is why that one never showed
the problem and why the tag looked right.
*/
func TestApplyDefaults_LeavesNoNilWhereTheColumnRefusesOne(t *testing.T) {
	t.Parallel()

	d := &agentdefinition.Definition{}
	d.ApplyDefaults()

	require.NotNil(t, d.ToolDailyLimits)
	assert.Empty(t, d.ToolDailyLimits)
}

// A cap somebody set is not a default to overwrite.
func TestApplyDefaults_KeepsTheCapsAlreadySet(t *testing.T) {
	t.Parallel()

	d := &agentdefinition.Definition{ToolDailyLimits: map[string]int{"email_customer": 5}}
	d.ApplyDefaults()

	assert.Equal(t, 5, d.ToolDailyLimits["email_customer"])
}
