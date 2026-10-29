package agentpreset_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentinstructions"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
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
func starterDefinition(template agenttemplate.Template) *agentdefinition.Definition {
	return &agentdefinition.Definition{
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		Name:            agenttemplate.Label(template),
		Description:     agenttemplate.Description(template),
		Template:        template,
		Instructions:    agentinstructions.StarterInstructions(template),
		ToolNames:       agentpreset.StarterTools(template),
		AutonomyCeiling: agentpreset.StarterCeiling(template),
		TriggerMode:     agentpreset.StarterTrigger(template),
		EventKinds:      agentpreset.StarterEvents(template),
		CronExpression:  agentpreset.StarterCron(template),
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

	for _, template := range agenttemplate.AllTemplates() {
		t.Run(string(template), func(t *testing.T) {
			t.Parallel()

			definition := starterDefinition(template)
			definition.ApplyDefaults()

			multiErr := errortypes.NewMultiError()
			definition.Validate(multiErr)
			require.Falsef(t, multiErr.HasErrors(),
				"the %s starter does not validate: %v", agenttemplate.Label(template), multiErr,
			)
		})
	}
}

// A trigger and the thing that drives it are set by two different switches, so
// they can disagree without anything failing to compile.
func TestTemplates_TriggersCarryWhatDrivesThem(t *testing.T) {
	t.Parallel()

	for _, template := range agenttemplate.AllTemplates() {
		t.Run(string(template), func(t *testing.T) {
			t.Parallel()

			switch agentpreset.StarterTrigger(template) {
			case agentdefinition.TriggerEvent:
				require.NotEmptyf(t, agentpreset.StarterEvents(template),
					"%s is event-triggered with no events", agenttemplate.Label(template),
				)
				for _, kind := range agentpreset.StarterEvents(template) {
					require.Truef(t, kind.IsValid(),
						"%s waits on %q, which nothing raises", agenttemplate.Label(template), kind,
					)
				}
				require.Emptyf(t, agentpreset.StarterCron(template),
					"%s is event-triggered and also carries a cron", agenttemplate.Label(template),
				)
			case agentdefinition.TriggerScheduled:
				require.NotEmptyf(t, agentpreset.StarterCron(template),
					"%s is scheduled with no cron", agenttemplate.Label(template),
				)
			case agentdefinition.TriggerChat, agentdefinition.TriggerContinuous:
				require.Emptyf(t, agentpreset.StarterEvents(template),
					"%s is not event-triggered but lists events", agenttemplate.Label(template),
				)
			}
		})
	}
}

// The desks that reach outside the company start where a person still sees
// every message before it goes.
func TestTemplates_OutwardFacingDesksStartAtPropose(t *testing.T) {
	t.Parallel()

	for _, template := range []agenttemplate.Template{
		agenttemplate.TemplateCustomerUpdateDesk,
		agenttemplate.TemplateCarrierRiskDesk,
		agenttemplate.TemplateLoadEntryCheck,
		agenttemplate.TemplateInsightAnalyst,
		agenttemplate.TemplateEDIDesk,
	} {
		require.Equal(
			t,
			agent.TierPropose,
			agentpreset.StarterCeiling(template),
			agenttemplate.Label(template),
		)
	}
}

func TestTemplates_TheNewDesksWaitOnTheirOwnEvents(t *testing.T) {
	t.Parallel()

	cases := map[agenttemplate.Template][]agenttrigger.EventKind{
		agenttemplate.TemplateDetentionDesk: {
			agenttrigger.EventDetentionOccurrenceOpened,
			agenttrigger.EventDetentionNoticeDue,
		},
		agenttemplate.TemplateCredentialDesk: {
			agenttrigger.EventWorkerCredentialExpiring,
		},
		agenttemplate.TemplateCustomerUpdateDesk: {
			agenttrigger.EventShipmentMoveArrived,
			agenttrigger.EventShipmentMoveDeparted,
		},
		agenttemplate.TemplateCarrierRiskDesk: {
			agenttrigger.EventCarrierIntelEventOpened,
		},
		agenttemplate.TemplateDispatchAssignment: {
			agenttrigger.EventShipmentMoveCoverageAtRisk,
			agenttrigger.EventShipmentMoveUnassigned,
		},
		agenttemplate.TemplateIntakeDesk: {
			agenttrigger.EventInboundMessageClassified,
			agenttrigger.EventEDITenderReceived,
		},
		agenttemplate.TemplateBillingException: {
			agenttrigger.EventBillingQueueItemException,
			agenttrigger.EventBillingQueueItemOnHold,
			agenttrigger.EventAccountingConnectionDegraded,
		},
		agenttemplate.TemplateLoadEntryCheck: {
			agenttrigger.EventShipmentCreated,
		},
		agenttemplate.TemplateServiceFailureDesk: {
			agenttrigger.EventServiceFailureDetected,
		},
		agenttemplate.TemplateInsightAnalyst: {
			agenttrigger.EventInsightDetected,
		},
		agenttemplate.TemplateEDIDesk: {
			agenttrigger.EventEDIFileQuarantined,
		},
	}

	for template, kinds := range cases {
		require.ElementsMatch(
			t,
			kinds,
			agentpreset.StarterEvents(template),
			agenttemplate.Label(template),
		)
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

	desk := agenttemplate.TemplateIntakeDesk
	require.Equal(t, agent.TierAutoExecute, agentpreset.StarterCeiling(desk))
	require.Equal(t, agentdefinition.TriggerEvent, agentpreset.StarterTrigger(desk))

	tools := agentpreset.StarterTools(desk)
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

	heard := make(map[agenttrigger.EventKind][]agenttemplate.Template)
	for _, template := range agenttemplate.AllTemplates() {
		for _, kind := range agentpreset.StarterEvents(template) {
			heard[kind] = append(heard[kind], template)
		}
	}

	for _, event := range agenttrigger.KnownEvents() {
		require.NotEmptyf(t, heard[event.Kind],
			"nothing starts from %q, so a hand-off from the watchtower suggests no agent for it",
			event.Kind,
		)
	}
}

func TestTemplates_TheNewDesksHoldWhatTheirWorkNeedsAndNothingThatLeaves(t *testing.T) {
	t.Parallel()

	cases := []struct {
		template agenttemplate.Template
		needs    []string
		never    []string
	}{
		{
			template: agenttemplate.TemplateLoadEntryCheck,
			needs: []string{
				"shipment_get", "shipment_list", "rate_explain", "rate_quote",
				"masterdata_list_hold_reasons", "shipment_add_comment", "shipment_place_hold",
			},
			never: []string{"customer_send_email", "shipment_update", "shipment_cancel"},
		},
		{
			template: agenttemplate.TemplateServiceFailureDesk,
			needs: []string{
				"exception_list_service_failures", "exception_get_service_failure",
				"exception_list_service_failure_reason_codes", "exception_resolve_service_failure",
				"customer_get_update_preferences", "customer_send_email",
			},
			never: []string{"receivable_post_customer_payment", "worker_send_message"},
		},
		{
			template: agenttemplate.TemplateCarrierRiskDesk,
			needs: []string{
				"carrier_list_intelligence_events", "carrier_get_intelligence_event",
				"carrier_acknowledge_intelligence_event", "carrier_resolve_intelligence_event", "carrier_get",
				"carrier_set_status",
			},
			never: []string{"customer_send_email"},
		},
		{
			template: agenttemplate.TemplateComplianceAssistant,
			needs: []string{
				"carrier_list_intelligence_events", "carrier_list", "carrier_get",
				"carrier_record_intelligence_review", "worker_list_credentials",
			},
		},
		{
			template: agenttemplate.TemplateInsightAnalyst,
			needs:    []string{"insight_get", "insight_list", "report_preview", "insight_dismiss"},
			never:    []string{"customer_send_email", "report_create", "report_update"},
		},
		{
			template: agenttemplate.TemplateDetentionDesk,
			needs: []string{
				"detention_get_occurrence", "inbox_get_inbound_message", "inbox_list_inbound_messages",
				"detention_resolve_dispute",
			},
			never: []string{
				"inbox_send_reply", "inbox_block_sender", "inbox_create_load",
			},
		},
		{
			template: agenttemplate.TemplateEDIDesk,
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
		tools := agentpreset.StarterTools(tc.template)
		for _, tool := range tc.needs {
			require.Containsf(t, tools, tool, "%s needs %s", agenttemplate.Label(tc.template), tool)
		}
		for _, tool := range tc.never {
			require.NotContainsf(
				t,
				tools,
				tool,
				"%s must not hold %s",
				agenttemplate.Label(tc.template),
				tool,
			)
		}
	}
}

func TestTemplates_TheLoadMonitorReadsWhetherPartnersAcknowledgeWhatWasSent(t *testing.T) {
	t.Parallel()

	require.Contains(
		t,
		agentpreset.StarterTools(agenttemplate.TemplateLoadMonitor),
		"edi_get_ack_health",
	)
}

func TestTemplates_TheInsightAnalystIsCappedAtTwentyRunsADay(t *testing.T) {
	t.Parallel()

	require.Equal(t, 20, agentpreset.StarterDailyRunLimit(agenttemplate.TemplateInsightAnalyst))
	for _, template := range agenttemplate.AllTemplates() {
		if template == agenttemplate.TemplateInsightAnalyst {
			continue
		}
		require.Zerof(
			t,
			agentpreset.StarterDailyRunLimit(template),
			"%s",
			agenttemplate.Label(template),
		)
	}
}

func TestTemplates_DataAccessFollowsWhatEachDeskReads(t *testing.T) {
	t.Parallel()

	cases := map[agenttemplate.Template]agentdefinition.DataAccessCeiling{
		agenttemplate.TemplateLoadEntryCheck:     agentdefinition.DataAccessRestricted,
		agenttemplate.TemplateInsightAnalyst:     agentdefinition.DataAccessRestricted,
		agenttemplate.TemplateCashApplication:    agentdefinition.DataAccessRestricted,
		agenttemplate.TemplateServiceFailureDesk: agentdefinition.DataAccessInternal,
		agenttemplate.TemplateEDIDesk:            agentdefinition.DataAccessInternal,
		agenttemplate.TemplateBillingException:   agentdefinition.DataAccessInternal,
		agenttemplate.TemplateFormulaAssistant:   agentdefinition.DataAccessRestricted,
		agenttemplate.TemplateReportAnalyst:      agentdefinition.DataAccessRestricted,
	}
	for template, want := range cases {
		require.Equalf(
			t,
			want,
			agentpreset.StarterDataAccess(template),
			"%s",
			agenttemplate.Label(template),
		)
	}
}

func TestTemplates_TheFormulaAssistantIsAChatAgentThatOnlyProposes(t *testing.T) {
	t.Parallel()

	assistant := agenttemplate.TemplateFormulaAssistant
	require.Equal(t, agentdefinition.TriggerChat, agentpreset.StarterTrigger(assistant))
	require.Empty(t, agentpreset.StarterEvents(assistant))
	require.Equal(t, agent.TierPropose, agentpreset.StarterCeiling(assistant))
	require.NotEmpty(t, agentpreset.StarterTools(assistant))
}

func TestTemplates_StandingToolsAreTheTemplatesOwnTools(t *testing.T) {
	t.Parallel()

	for _, template := range agenttemplate.AllTemplates() {
		standing := agentpreset.StandingTools(template)
		if template != agenttemplate.TemplateFormulaAssistant {
			require.Emptyf(t, standing, "%s", agenttemplate.Label(template))
			continue
		}
		require.Contains(t, standing, "formula_propose")
		require.Contains(t, standing, "formula_library_propose")
		require.Subset(t, agentpreset.StarterTools(template), standing)
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

	for _, template := range agenttemplate.AllTemplates() {
		instructions := agentinstructions.StarterInstructions(template)
		for _, tool := range agentpreset.StarterTools(template) {
			if !strings.Contains(tool, "_") {
				continue
			}
			require.NotContainsf(t, instructions, tool,
				"the %s instructions name %s; say it on the tool's recipe instead",
				agenttemplate.Label(template), tool)
		}
	}
}
