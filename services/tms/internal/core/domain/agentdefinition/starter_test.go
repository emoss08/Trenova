package agentdefinition_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
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
holding create_shipment would put two proposals for the same load in front of a
person. A tender that arrives over EDI never becomes a document, so the desk
answers it, and accepting one is always a proposal a person decides.
*/
func TestTemplates_TheIntakeDeskMayEarnAutonomyTheMailboxGrants(t *testing.T) {
	t.Parallel()

	desk := agentdefinition.TemplateIntakeDesk
	require.Equal(t, agent.TierAutoExecute, desk.StarterCeiling())
	require.Equal(t, agentdefinition.TriggerEvent, desk.StarterTrigger())

	tools := desk.StarterTools()
	for _, tool := range []string{
		"get_inbound_message",
		"link_inbound_message",
		"mark_inbound_message",
		"reply_to_inbound_message",
		"attach_document_to_shipment",
		"get_edi_transfer",
		"accept_edi_tender",
		"decline_edi_tender",
	} {
		require.Containsf(t, tools, tool, "the intake desk needs %s", tool)
	}
	for _, tool := range []string{
		"create_shipment", "post_customer_payment", "tender_move_to_carriers",
		"send_edi_tender", "replay_edi_message",
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
				"get_shipment", "search_shipments", "explain_rate", "quote_shipment",
				"list_hold_reasons", "add_shipment_comment", "place_shipment_hold",
			},
			never: []string{"email_customer", "update_shipment", "cancel_shipment"},
		},
		{
			template: agentdefinition.TemplateServiceFailureDesk,
			needs: []string{
				"list_service_failures", "get_service_failure",
				"list_service_failure_reason_codes", "resolve_service_failure",
				"get_customer_update_preferences", "email_customer",
			},
			never: []string{"post_customer_payment", "notify_driver"},
		},
		{
			template: agentdefinition.TemplateInsightAnalyst,
			needs:    []string{"get_insight", "list_insights", "preview_report", "dismiss_insight"},
			never:    []string{"email_customer", "create_report", "update_report"},
		},
		{
			template: agentdefinition.TemplateEDIDesk,
			needs: []string{
				"list_edi_inbound_files", "get_edi_inbound_file", "list_edi_transfers",
				"get_edi_partner", "reprocess_edi_inbound_files",
			},
			never: []string{
				"create_shipment", "email_customer", "reply_to_inbound_message",
				"accept_edi_tender", "retry_edi_message_delivery", "replay_edi_message",
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

func TestTemplates_TheFormulaAssistantIsAChatAgentThatOnlyReads(t *testing.T) {
	t.Parallel()

	assistant := agentdefinition.TemplateFormulaAssistant
	require.Equal(t, agentdefinition.TriggerChat, assistant.StarterTrigger())
	require.Empty(t, assistant.StarterEvents())
	require.Equal(t, agent.TierPropose, assistant.StarterCeiling())
	require.NotEmpty(t, assistant.StarterTools())
}
