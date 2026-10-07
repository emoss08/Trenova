package agentdraftservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCatalog() []serviceports.ToolCatalogEntry {
	return []serviceports.ToolCatalogEntry{
		{
			Name:      "get_shipment",
			Kind:      serviceports.ToolCatalogKindQuery,
			Resource:  permission.ResourceShipment,
			Operation: permission.OpRead,
		},
		{
			Name:            "update_customer",
			Kind:            serviceports.ToolCatalogKindAction,
			Resource:        permission.ResourceCustomer,
			Operation:       permission.OpUpdate,
			MaxAutonomyTier: agent.TierAutoExecute,
		},
		{
			Name:            "send_customer_email",
			Kind:            serviceports.ToolCatalogKindAction,
			Resource:        permission.ResourceCustomer,
			Operation:       permission.OpUpdate,
			MaxAutonomyTier: agent.TierActWithApproval,
		},
		{
			Name:      "remember",
			Kind:      serviceports.ToolCatalogKindAction,
			Core:      true,
			Resource:  permission.ResourceAgentDefinition,
			Operation: permission.OpUpdate,
		},
	}
}

func chatReply() *draftReply {
	return &draftReply{
		Name:            "Customer desk",
		Description:     "Answers customer questions.",
		Icon:            agentdefinition.IconHeadset,
		Accent:          agentdefinition.AccentTeal,
		Instructions:    "Answer questions about customers and their shipments.",
		Guardrails:      []string{"Never promise a delivery date."},
		TriggerMode:     string(agentdefinition.TriggerChat),
		Tools:           []draftTool{{Name: "get_shipment", Tier: string(agent.TierPropose)}},
		AutonomyCeiling: string(agent.TierActWithApproval),
		OutputMode:      string(agentdefinition.OutputConversational),
	}
}

func sanitize(
	t *testing.T,
	reply *draftReply,
) (*serviceports.SaveAgentDefinitionRequest, []serviceports.AgentDraftNote) {
	t.Helper()

	req, notes, err := sanitizeDraft(&sanitizeInput{
		reply:       reply,
		catalog:     testCatalog(),
		timezone:    "America/Chicago",
		description: "Answer customer questions about where their loads are.",
	})
	require.NoError(t, err)

	return req, notes
}

func noteFields(notes []serviceports.AgentDraftNote) []string {
	fields := make([]string, 0, len(notes))
	for _, note := range notes {
		fields = append(fields, note.Field)
	}

	return fields
}

func TestSanitizeDraftKeepsAValidDraftAndStartsItInShadow(t *testing.T) {
	t.Parallel()

	req, notes := sanitize(t, chatReply())

	assert.Empty(t, notes)
	assert.Equal(t, "Customer desk", req.Name)
	assert.Equal(t, agentdefinition.IconHeadset, req.Icon)
	assert.Equal(t, agentdefinition.AccentTeal, req.Accent)
	assert.Equal(t, []string{"get_shipment"}, req.ToolNames)
	assert.Empty(t, req.ToolTiers)
	assert.Equal(t, agent.TierActWithApproval, req.AutonomyCeiling)
	assert.Equal(t, agentdefinition.TriggerChat, req.TriggerMode)
	assert.Equal(t, agentdefinition.DataAccessInternal, req.DataAccessCeiling)
	assert.True(t, req.ShadowMode)
	assert.True(t, req.Enabled)
	assert.True(t, req.ID.IsNil())
}

func TestSanitizeDraftDropsToolsTheRegistryDoesNotOffer(t *testing.T) {
	t.Parallel()

	reply := chatReply()
	reply.Tools = []draftTool{
		{Name: "get_shipment"},
		{Name: "exec_shell"},
		{Name: "get_shipment"},
		{Name: "remember"},
		{Name: "  update_customer  "},
	}

	req, notes := sanitize(t, reply)

	assert.Equal(t, []string{"get_shipment", "update_customer"}, req.ToolNames)
	require.Len(t, notes, 1)
	assert.Equal(t, "toolNames", notes[0].Field)
	assert.Equal(t, "exec_shell", notes[0].Value)
}

func TestSanitizeDraftCapsTiersAtTheToolAndTheCeiling(t *testing.T) {
	t.Parallel()

	reply := chatReply()
	reply.AutonomyCeiling = string(agent.TierAutoExecute)
	reply.Tools = []draftTool{
		{Name: "update_customer", Tier: string(agent.TierAutoExecute)},
		{Name: "send_customer_email", Tier: string(agent.TierAutoExecute)},
		{Name: "get_shipment", Tier: string(agent.TierAutoExecute)},
	}

	req, notes := sanitize(t, reply)

	assert.Equal(t, agent.TierAutoExecute, req.ToolTiers["update_customer"])
	assert.Equal(t, agent.TierActWithApproval, req.ToolTiers["send_customer_email"])
	assert.NotContains(t, req.ToolTiers, "get_shipment")
	assert.Equal(t, []string{"toolTiers.send_customer_email"}, noteFields(notes))

	reply.AutonomyCeiling = string(agent.TierPropose)
	req, notes = sanitize(t, reply)

	assert.Equal(t, agent.TierPropose, req.ToolTiers["update_customer"])
	assert.Equal(t, agent.TierPropose, req.ToolTiers["send_customer_email"])
	assert.ElementsMatch(
		t,
		[]string{"toolTiers.update_customer", "toolTiers.send_customer_email"},
		noteFields(notes),
	)
}

func TestSanitizeDraftDropsAnUnknownTierAndAnUnknownCeiling(t *testing.T) {
	t.Parallel()

	reply := chatReply()
	reply.AutonomyCeiling = "Unlimited"
	reply.Tools = []draftTool{{Name: "update_customer", Tier: "Whenever"}}

	req, notes := sanitize(t, reply)

	assert.Equal(t, agent.TierPropose, req.AutonomyCeiling)
	assert.Empty(t, req.ToolTiers)
	assert.ElementsMatch(
		t,
		[]string{"autonomyCeiling", "toolTiers.update_customer"},
		noteFields(notes),
	)
}

func TestSanitizeDraftDropsEventsTheSystemDoesNotRaise(t *testing.T) {
	t.Parallel()

	known := agent.KnownEvents()[0].Kind
	reply := chatReply()
	reply.TriggerMode = string(agentdefinition.TriggerEvent)
	reply.EventKinds = []string{"shipment.teleported", string(known), string(known)}
	reply.OutputMode = ""

	req, notes := sanitize(t, reply)

	assert.Equal(t, agentdefinition.TriggerEvent, req.TriggerMode)
	assert.Equal(t, []agent.EventKind{known}, req.EventKinds)
	assert.Equal(t, agentdefinition.OutputReport, req.OutputMode)
	require.Len(t, notes, 1)
	assert.Equal(t, "eventKinds", notes[0].Field)
	assert.Equal(t, "shipment.teleported", notes[0].Value)
}

func TestSanitizeDraftFallsBackToChatWhenNoEventIsReal(t *testing.T) {
	t.Parallel()

	reply := chatReply()
	reply.TriggerMode = string(agentdefinition.TriggerEvent)
	reply.EventKinds = []string{"shipment.teleported"}

	req, notes := sanitize(t, reply)

	assert.Equal(t, agentdefinition.TriggerChat, req.TriggerMode)
	assert.Empty(t, req.EventKinds)
	assert.ElementsMatch(t, []string{"eventKinds", "triggerMode"}, noteFields(notes))
}

func TestSanitizeDraftKeepsAValidSchedule(t *testing.T) {
	t.Parallel()

	reply := chatReply()
	reply.TriggerMode = string(agentdefinition.TriggerScheduled)
	reply.CronExpression = " 0 7 * * 1-5 "
	reply.CronTimezone = "America/Denver"

	req, notes := sanitize(t, reply)

	assert.Empty(t, notes)
	assert.Equal(t, agentdefinition.TriggerScheduled, req.TriggerMode)
	assert.Equal(t, "0 7 * * 1-5", req.CronExpression)
	assert.Equal(t, "America/Denver", req.CronTimezone)
}

func TestSanitizeDraftClearsABadCron(t *testing.T) {
	t.Parallel()

	reply := chatReply()
	reply.TriggerMode = string(agentdefinition.TriggerScheduled)
	reply.CronExpression = "every weekday at seven"
	reply.CronTimezone = "Mars/Olympus"

	req, notes := sanitize(t, reply)

	assert.Equal(t, agentdefinition.TriggerChat, req.TriggerMode)
	assert.Empty(t, req.CronExpression)
	assert.Equal(t, "America/Chicago", req.CronTimezone)
	assert.ElementsMatch(
		t,
		[]string{"cronExpression", "cronTimezone", "triggerMode"},
		noteFields(notes),
	)
}

func TestSanitizeDraftHoldsAContinuousAgentToTheShortestInterval(t *testing.T) {
	t.Parallel()

	reply := chatReply()
	reply.TriggerMode = string(agentdefinition.TriggerContinuous)
	reply.IntervalSeconds = 5

	req, notes := sanitize(t, reply)

	assert.Equal(t, agentdefinition.TriggerContinuous, req.TriggerMode)
	assert.Equal(t, agentdefinition.MinIntervalSeconds, req.IntervalSeconds)
	assert.Equal(t, []string{"intervalSeconds"}, noteFields(notes))
}

func TestSanitizeDraftDropsAnUnknownTriggerIconAndAccent(t *testing.T) {
	t.Parallel()

	reply := chatReply()
	reply.TriggerMode = "Telepathy"
	reply.Icon = "unicorn"
	reply.Accent = "plaid"

	req, notes := sanitize(t, reply)

	assert.Equal(t, agentdefinition.TriggerChat, req.TriggerMode)
	assert.Empty(t, req.Icon)
	assert.Empty(t, req.Accent)
	assert.ElementsMatch(t, []string{"triggerMode", "icon", "accent"}, noteFields(notes))
}

func TestSanitizeDraftTrimsGuardrailsToWhatASaveAccepts(t *testing.T) {
	t.Parallel()

	reply := chatReply()
	reply.Guardrails = []string{
		"Never promise a delivery date.",
		"  ",
		"Never promise a delivery date.",
		strings.Repeat("x", agentdefinition.MaxGuardrailRunes+1),
	}
	for idx := range agentdefinition.MaxGuardrails {
		reply.Guardrails = append(reply.Guardrails, "Rule "+string(rune('A'+idx)))
	}

	req, notes := sanitize(t, reply)

	assert.Len(t, req.Guardrails, agentdefinition.MaxGuardrails)
	assert.Equal(t, "Never promise a delivery date.", req.Guardrails[0])
	assert.Contains(t, noteFields(notes), "guardrails")
}

func TestSanitizeDraftNamesAnUnnamedAgentFromItsDescription(t *testing.T) {
	t.Parallel()

	reply := chatReply()
	reply.Name = "   "

	req, notes := sanitize(t, reply)

	assert.Equal(t, "Answer customer questions about where their loads are.", req.Name)
	assert.Equal(t, []string{"name"}, noteFields(notes))
}

func TestSanitizeDraftFailsWhenNothingIsUsable(t *testing.T) {
	t.Parallel()

	reply := &draftReply{
		Name:  "Nothing",
		Tools: []draftTool{{Name: "exec_shell"}},
	}

	_, _, err := sanitizeDraft(&sanitizeInput{reply: reply, catalog: testCatalog()})

	require.Error(t, err)
	var business *errortypes.BusinessError
	assert.ErrorAs(t, err, &business)
}

func TestParseDraftReplyRefusesEmptyAndUnreadableText(t *testing.T) {
	t.Parallel()

	_, err := parseDraftReply("   ")
	require.Error(t, err)

	_, err = parseDraftReply("Sure! Here is an agent.")
	require.Error(t, err)

	reply, err := parseDraftReply(
		`{"name":"Desk","instructions":"Help.","tools":[{"name":"get_shipment","tier":"Propose"}]}`,
	)
	require.NoError(t, err)
	assert.Equal(t, "Desk", reply.Name)
	assert.Equal(t, []draftTool{{Name: "get_shipment", Tier: "Propose"}}, reply.Tools)
}
