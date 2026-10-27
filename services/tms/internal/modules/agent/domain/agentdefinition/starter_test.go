package agentdefinition_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

/*
A template's starters are ten switches with a default arm each, so a template
added without one of them compiles and saves a definition that does not work:
an event trigger with no events fails validation the first time somebody tries
to save it, a schedule with no cron never runs, and a desk with no ceiling
inherits one it should not have.

Building the definition the way the seed builds it and validating it is what
catches a missed arm here rather than in the form.
*/
func starterDefinition(template agentdefinition.Template) *agentdefinition.Definition {
	return &agentdefinition.Definition{
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		Name:            template.Label(),
		Description:     template.Description(),
		Template:        template,
		Instructions:    template.StarterInstructions(),
		ToolNames:       template.StarterTools(),
		AutonomyCeiling: template.StarterCeiling(),
		TriggerMode:     template.StarterTrigger(),
		EventKinds:      template.StarterEvents(),
		CronExpression:  template.StarterCron(),
		OutputMode:      agentdefinition.OutputReport,
		ContextProviders: []agentdefinition.ContextProvider{
			agentdefinition.ContextOrganization,
			agentdefinition.ContextClock,
			agentdefinition.ContextTools,
		},
	}
}

func TestTemplates_EveryStarterSavesAsItStands(t *testing.T) {
	t.Parallel()

	for _, template := range agentdefinition.AllTemplates() {
		t.Run(string(template), func(t *testing.T) {
			t.Parallel()

			definition := starterDefinition(template)
			definition.ApplyDefaults()

			multiErr := errortypes.NewMultiError()
			definition.Validate(multiErr)
			require.Falsef(t, multiErr.HasErrors(),
				"the %s starter does not validate: %v", template.Label(), multiErr,
			)
		})
	}
}

// A trigger and the thing that drives it are set by two different switches, so
// they can disagree without anything failing to compile.
func TestTemplates_TriggersCarryWhatDrivesThem(t *testing.T) {
	t.Parallel()

	for _, template := range agentdefinition.AllTemplates() {
		t.Run(string(template), func(t *testing.T) {
			t.Parallel()

			switch template.StarterTrigger() {
			case agentdefinition.TriggerEvent:
				require.NotEmptyf(t, template.StarterEvents(),
					"%s is event-triggered with no events", template.Label(),
				)
				for _, kind := range template.StarterEvents() {
					require.Truef(t, kind.IsValid(),
						"%s waits on %q, which nothing raises", template.Label(), kind,
					)
				}
				require.Emptyf(t, template.StarterCron(),
					"%s is event-triggered and also carries a cron", template.Label(),
				)
			case agentdefinition.TriggerScheduled:
				require.NotEmptyf(t, template.StarterCron(),
					"%s is scheduled with no cron", template.Label(),
				)
			case agentdefinition.TriggerChat, agentdefinition.TriggerContinuous:
				require.Emptyf(t, template.StarterEvents(),
					"%s is not event-triggered but lists events", template.Label(),
				)
			}
		})
	}
}

// The desks that reach outside the company start where a person still sees
// every message before it goes.
func TestTemplates_OutwardFacingDesksStartAtPropose(t *testing.T) {
	t.Parallel()

	for _, template := range []agentdefinition.Template{
		agentdefinition.TemplateCustomerUpdateDesk,
		agentdefinition.TemplateCarrierRiskDesk,
		agentdefinition.TemplateLoadEntryCheck,
		agentdefinition.TemplateInsightAnalyst,
		agentdefinition.TemplateEDIDesk,
	} {
		require.Equal(t, agent.TierPropose, template.StarterCeiling(), template.Label())
	}
}

func TestTemplates_TheNewDesksWaitOnTheirOwnEvents(t *testing.T) {
	t.Parallel()

	cases := map[agentdefinition.Template][]agent.EventKind{
		agentdefinition.TemplateDetentionDesk: {
			agent.EventDetentionOccurrenceOpened,
			agent.EventDetentionNoticeDue,
		},
		agentdefinition.TemplateCredentialDesk: {
			agent.EventWorkerCredentialExpiring,
		},
		agentdefinition.TemplateCustomerUpdateDesk: {
			agent.EventShipmentMoveArrived,
			agent.EventShipmentMoveDeparted,
		},
		agentdefinition.TemplateCarrierRiskDesk: {
			agent.EventCarrierIntelEventOpened,
		},
		agentdefinition.TemplateDispatchAssignment: {
			agent.EventShipmentMoveCoverageAtRisk,
			agent.EventShipmentMoveUnassigned,
		},
		agentdefinition.TemplateIntakeDesk: {
			agent.EventInboundMessageClassified,
			agent.EventEDITenderReceived,
		},
		agentdefinition.TemplateBillingException: {
			agent.EventBillingQueueItemException,
			agent.EventBillingQueueItemOnHold,
			agent.EventAccountingConnectionDegraded,
		},
		agentdefinition.TemplateLoadEntryCheck: {
			agent.EventShipmentCreated,
		},
		agentdefinition.TemplateServiceFailureDesk: {
			agent.EventServiceFailureDetected,
		},
		agentdefinition.TemplateInsightAnalyst: {
			agent.EventInsightDetected,
		},
		agentdefinition.TemplateEDIDesk: {
			agent.EventEDIFileQuarantined,
		},
	}

	for template, kinds := range cases {
		require.ElementsMatch(t, kinds, template.StarterEvents(), template.Label())
	}
}

/*
The intake desk is the one desk that may earn running unattended, because the
owner decided an inbox the organization trusts should answer a status question
without a person. The grant is not the template's to make: each inbox tool holds
itself to a proposal unless the message's mailbox handles it without review. The
template only makes room.

It holds nothing that creates a load or money on its own. A tender's attachment
already wakes shipment intake through the document it became; a second desk
holding shipment_create would put two proposals for the same load in front of a
person. A tender written only in an email's body reaches no other agent, so the
desk drafts it and proposes inbox_create_load, which never runs unattended. A
tender that arrives over EDI never becomes a document, so the desk answers it,
and accepting one is always a proposal a person decides. A detention dispute's
money is the detention desk's, so detention_resolve_dispute is not the intake desk's.
*/
func TestTemplates_TheIntakeDeskMayEarnAutonomyTheMailboxGrants(t *testing.T) {
	t.Parallel()

	desk := agentdefinition.TemplateIntakeDesk
	require.Equal(t, agent.TierAutoExecute, desk.StarterCeiling())
	require.Equal(t, agentdefinition.TriggerEvent, desk.StarterTrigger())

	tools := desk.StarterTools()
	for _, tool := range []string{
		"inbox_get_inbound_message",
		"inbox_link_inbound_message",
		"inbox_mark_inbound_message",
		"inbox_send_reply",
		"shipment_attach_document",
		"edi_get_transfer",
		"edi_accept_tender",
		"edi_decline_tender",
		"inbox_list_triage_assignees",
		"inbox_unlink_inbound_message",
		"inbox_draft_reply",
		"inbox_await_reply",
		"inbox_snooze_inbound_message",
		"inbox_wake_inbound_message",
		"inbox_file_attachment",
		"inbox_resolve_conflict",
		"inbox_record_discrepancy",
		"inbox_draft_load",
		"inbox_create_load",
		"inbox_block_sender",
	} {
		require.Containsf(t, tools, tool, "the intake desk needs %s", tool)
	}
	for _, tool := range []string{
		"shipment_create", "receivable_post_customer_payment", "dispatch_create_tender",
		"edi_send_tender", "edi_replay_message", "detention_resolve_dispute",
	} {
		require.NotContainsf(t, tools, tool, "the intake desk must not hold %s", tool)
	}
}

func TestTemplates_EveryEventIsHeardByAStarter(t *testing.T) {
	t.Parallel()

	heard := make(map[agent.EventKind][]agentdefinition.Template)
	for _, template := range agentdefinition.AllTemplates() {
		for _, kind := range template.StarterEvents() {
			heard[kind] = append(heard[kind], template)
		}
	}

	for _, event := range agent.KnownEvents() {
		require.NotEmptyf(t, heard[event.Kind],
			"nothing starts from %q, so a hand-off from the watchtower suggests no agent for it",
			event.Kind,
		)
	}
}

func TestTemplates_TheNewDesksHoldWhatTheirWorkNeedsAndNothingThatLeaves(t *testing.T) {
	t.Parallel()

	cases := []struct {
		template agentdefinition.Template
		needs    []string
		never    []string
	}{
		{
			template: agentdefinition.TemplateLoadEntryCheck,
			needs: []string{
				"shipment_get", "shipment_list", "rate_explain", "rate_quote",
				"masterdata_list_hold_reasons", "shipment_add_comment", "shipment_place_hold",
			},
			never: []string{"customer_send_email", "shipment_update", "shipment_cancel"},
		},
		{
			template: agentdefinition.TemplateServiceFailureDesk,
			needs: []string{
				"exception_list_service_failures", "exception_get_service_failure",
				"exception_list_service_failure_reason_codes", "exception_resolve_service_failure",
				"customer_get_update_preferences", "customer_send_email",
			},
			never: []string{"receivable_post_customer_payment", "worker_send_message"},
		},
		{
			template: agentdefinition.TemplateCarrierRiskDesk,
			needs: []string{
				"carrier_list_intelligence_events", "carrier_get_intelligence_event",
				"carrier_acknowledge_intelligence_event", "carrier_resolve_intelligence_event", "carrier_get",
			},
			never: []string{"carrier_set_status", "customer_send_email"},
		},
		{
			template: agentdefinition.TemplateComplianceAssistant,
			needs: []string{
				"carrier_list_intelligence_events", "carrier_list", "carrier_get",
				"carrier_record_intelligence_review", "worker_list_credentials",
			},
		},
		{
			template: agentdefinition.TemplateInsightAnalyst,
			needs:    []string{"insight_get", "insight_list", "report_preview", "insight_dismiss"},
			never:    []string{"customer_send_email", "report_create", "report_update"},
		},
		{
			template: agentdefinition.TemplateDetentionDesk,
			needs: []string{
				"detention_get_occurrence", "inbox_get_inbound_message", "inbox_list_inbound_messages",
				"detention_resolve_dispute",
			},
			never: []string{
				"inbox_send_reply", "inbox_block_sender", "inbox_create_load",
			},
		},
		{
			template: agentdefinition.TemplateEDIDesk,
			needs: []string{
				"edi_list_inbound_files", "edi_get_inbound_file", "edi_list_transfers",
				"edi_get_partner", "edi_reprocess_inbound_file", "edi_list_messages",
				"edi_get_ack_health", "edi_inspect_x12", "edi_preview_x12",
			},
			never: []string{
				"shipment_create", "customer_send_email", "inbox_send_reply",
				"edi_accept_tender", "edi_retry_message", "edi_replay_message",
			},
		},
	}

	for _, tc := range cases {
		tools := tc.template.StarterTools()
		for _, tool := range tc.needs {
			require.Containsf(t, tools, tool, "%s needs %s", tc.template.Label(), tool)
		}
		for _, tool := range tc.never {
			require.NotContainsf(t, tools, tool, "%s must not hold %s", tc.template.Label(), tool)
		}
	}
}

func TestTemplates_TheLoadMonitorReadsWhetherPartnersAcknowledgeWhatWasSent(t *testing.T) {
	t.Parallel()

	require.Contains(t, agentdefinition.TemplateLoadMonitor.StarterTools(), "edi_get_ack_health")
}

func TestTemplates_TheInsightAnalystIsCappedAtTwentyRunsADay(t *testing.T) {
	t.Parallel()

	require.Equal(t, 20, agentdefinition.TemplateInsightAnalyst.StarterDailyRunLimit())
	for _, template := range agentdefinition.AllTemplates() {
		if template == agentdefinition.TemplateInsightAnalyst {
			continue
		}
		require.Zerof(t, template.StarterDailyRunLimit(), "%s", template.Label())
	}
}

func TestTemplates_DataAccessFollowsWhatEachDeskReads(t *testing.T) {
	t.Parallel()

	cases := map[agentdefinition.Template]agentdefinition.DataAccessCeiling{
		agentdefinition.TemplateLoadEntryCheck:     agentdefinition.DataAccessRestricted,
		agentdefinition.TemplateInsightAnalyst:     agentdefinition.DataAccessRestricted,
		agentdefinition.TemplateCashApplication:    agentdefinition.DataAccessRestricted,
		agentdefinition.TemplateServiceFailureDesk: agentdefinition.DataAccessInternal,
		agentdefinition.TemplateEDIDesk:            agentdefinition.DataAccessInternal,
		agentdefinition.TemplateBillingException:   agentdefinition.DataAccessInternal,
		agentdefinition.TemplateFormulaAssistant:   agentdefinition.DataAccessRestricted,
	}
	for template, want := range cases {
		require.Equalf(t, want, template.StarterDataAccess(), "%s", template.Label())
	}
}

func TestTemplates_TheFormulaAssistantIsAChatAgentThatOnlyProposes(t *testing.T) {
	t.Parallel()

	assistant := agentdefinition.TemplateFormulaAssistant
	require.Equal(t, agentdefinition.TriggerChat, assistant.StarterTrigger())
	require.Empty(t, assistant.StarterEvents())
	require.Equal(t, agent.TierPropose, assistant.StarterCeiling())
	require.NotEmpty(t, assistant.StarterTools())
}

func TestTemplates_StandingToolsAreTheTemplatesOwnTools(t *testing.T) {
	t.Parallel()

	for _, template := range agentdefinition.AllTemplates() {
		standing := template.StandingTools()
		if template != agentdefinition.TemplateFormulaAssistant {
			require.Emptyf(t, standing, "%s", template.Label())
			continue
		}
		require.Contains(t, standing, "formula_propose")
		require.Contains(t, standing, "formula_library_propose")
		require.Subset(t, template.StarterTools(), standing)
	}
}

/*
A template's instructions say who the agent is, what it puts first, the rules
it keeps and what is a person's to decide. The order of the work is on the
tools, as each one's recipe and prerequisites, where it is told with the tool.
Instructions that walked through the tools named ones a turn had not loaded,
went stale when a tool was renamed, and told an agent without a tool to call
it.
*/
func TestTemplates_InstructionsLeaveTheOrderOfWorkToTheTools(t *testing.T) {
	t.Parallel()

	for _, template := range agentdefinition.AllTemplates() {
		instructions := template.StarterInstructions()
		for _, tool := range template.StarterTools() {
			if !strings.Contains(tool, "_") {
				continue
			}
			require.NotContainsf(t, instructions, tool,
				"the %s instructions name %s; say it on the tool's recipe instead",
				template.Label(), tool)
		}
	}
}
