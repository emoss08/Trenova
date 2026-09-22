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
		},
		agentdefinition.TemplateIntakeDesk: {
			agent.EventInboundMessageClassified,
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

It holds nothing that creates a load or money. A tender's attachment already
wakes shipment intake through the document it became; a second desk holding
create_shipment would put two proposals for the same load in front of a person.
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
	} {
		require.Containsf(t, tools, tool, "the intake desk needs %s", tool)
	}
	for _, tool := range []string{"create_shipment", "post_customer_payment", "tender_move_to_carriers"} {
		require.NotContainsf(t, tools, tool, "the intake desk must not hold %s", tool)
	}
}
