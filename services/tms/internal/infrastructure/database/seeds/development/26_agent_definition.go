package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	SeedAgentDispatchName = "Dispatch desk"
	SeedAgentBillingName  = "Billing exceptions"

	SystemKeyBillingException   = "billing_exception"
	SystemKeyDispatchAssignment = "dispatch_assignment"
)

type AgentDefinitionSeed struct {
	seedhelpers.BaseSeed
}

// AgentDefinitionSeed configures a spread of agents so the assistant's picker,
// the AI Control page, the scheduler sweep and the conversation seed all have
// something to show: chat agents built from templates, the two system agents
// the platform's own events fire, and a scheduled report.
//
// Depends on:
//   - AdminAccount: the default organization the agents belong to
func NewAgentDefinitionSeed() *AgentDefinitionSeed {
	seed := &AgentDefinitionSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"AgentDefinition",
		"2.0.0",
		"Seeds chat, event-driven and scheduled agents for the default organization",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedAdminAccount)
	return seed
}

func (s *AgentDefinitionSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}

			cols := buncolgen.DefinitionColumns
			count, err := tx.NewSelect().
				Model((*agentdefinition.Definition)(nil)).
				Where(cols.OrganizationID.Eq(), org.ID).
				Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("count existing agents: %w", err)
			}
			if count > 0 {
				return nil
			}

			now := timeutils.NowUnix()
			for _, definition := range s.definitions(org.ID, org.BusinessUnitID) {
				definition.ApplyDefaults()
				if definition.TriggerMode == agentdefinition.TriggerScheduled ||
					definition.TriggerMode == agentdefinition.TriggerContinuous {
					next, nErr := definition.ComputeNextRun(now)
					if nErr != nil {
						return fmt.Errorf("schedule agent %s: %w", definition.Name, nErr)
					}
					definition.NextRunAt = &next
				}
				if _, err = tx.NewInsert().Model(definition).Exec(ctx); err != nil {
					return fmt.Errorf("insert agent %s: %w", definition.Name, err)
				}
				if err = sc.TrackCreated(ctx, "agent_definitions", definition.ID, s.Name()); err != nil {
					return err
				}
			}

			return nil
		},
	)
}

func (s *AgentDefinitionSeed) definitions(orgID, buID pulid.ID) []*agentdefinition.Definition {
	return []*agentdefinition.Definition{
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAgentDispatchName,
			Description:    "Looks up shipments and drivers for the dispatch team.",
			Template:       agentdefinition.TemplateDispatchAssistant,
			Instructions: agentdefinition.TemplateDispatchAssistant.StarterInstructions() + "\n\n" +
				"We run mostly reefer freight out of the Los Angeles terminal. When a driver is " +
				"asked about, check hours of service before anything else.",
			Guardrails:      []string{"Promise a delivery time to a customer", "Change a rate"},
			ToolNames:       []string{"get_shipment", "search_shipments", "get_worker", "search_workers"},
			AutonomyCeiling: agent.TierActWithApproval,
			TriggerMode:     agentdefinition.TriggerChat,
			Enabled:         true,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAgentBillingName,
			Description:    "Works blocked billing queue items and proposes what to do about them.",
			Template:       agentdefinition.TemplateBillingAssistant,
			Instructions: agentdefinition.TemplateBillingAssistant.StarterInstructions() + "\n\n" +
				"Never propose a rate change. If a rate looks wrong, flag it for a person and say " +
				"which document you would want to see.",
			ToolNames: []string{
				"get_shipment",
				"search_shipments",
				"flag_for_manual_review",
				"request_missing_docs",
			},
			AutonomyCeiling: agent.TierPropose,
			TriggerMode:     agentdefinition.TriggerChat,
			Enabled:         true,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           "Compliance desk",
			Description:    "Answers questions about driver qualification and what is expiring.",
			Template:       agentdefinition.TemplateComplianceAssistant,
			Instructions: agentdefinition.TemplateComplianceAssistant.StarterInstructions() + "\n\n" +
				"Medical cards and hazmat endorsements are the two we most often miss.",
			ToolNames:       []string{"get_worker", "search_workers"},
			AutonomyCeiling: agent.TierPropose,
			TriggerMode:     agentdefinition.TriggerChat,
			Enabled:         true,
		},
		{
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Name:            "Customer service",
			Description:     "Shipment status for the customer-facing team. Off until reviewed.",
			Template:        agentdefinition.TemplateCustomerAssistant,
			Instructions:    agentdefinition.TemplateCustomerAssistant.StarterInstructions(),
			ToolNames:       []string{"get_shipment", "search_shipments"},
			AutonomyCeiling: agent.TierPropose,
			TriggerMode:     agentdefinition.TriggerChat,
			Enabled:         false,
		},
		{
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Name:            "Help",
			Description:     "Explains how to do things in Trenova. Cannot read or change records.",
			Template:        agentdefinition.TemplateGeneralAssistant,
			Instructions:    agentdefinition.TemplateGeneralAssistant.StarterInstructions(),
			AutonomyCeiling: agent.TierPropose,
			TriggerMode:     agentdefinition.TriggerChat,
			Enabled:         true,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           "Billing exception agent",
			Description:    "Diagnoses each billing item that becomes blocked and proposes how to clear it.",
			Template:       agentdefinition.TemplateBillingException,
			Instructions:   agentdefinition.TemplateBillingException.StarterInstructions(),
			ToolNames: []string{
				"get_shipment",
				"search_shipments",
				"transition_item_to_in_review",
				"correct_charge_code",
				"request_missing_docs",
				"attach_document_to_bqi",
				"flag_for_manual_review",
			},
			AutonomyCeiling:  agent.TierActWithApproval,
			TriggerMode:      agentdefinition.TriggerEvent,
			EventKinds:       []agent.EventKind{agent.EventBillingQueueItemException},
			OutputMode:       agentdefinition.OutputReport,
			ShadowMode:       true,
			SystemKey:        SystemKeyBillingException,
			ContextProviders: []agentdefinition.ContextProvider{agentdefinition.ContextOrganization, agentdefinition.ContextClock, agentdefinition.ContextTools},
			Enabled:          true,
		},
		{
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Name:            "Dispatch coverage agent",
			Description:     "Every half hour, reviews moves with no driver and proposes assignments.",
			Template:        agentdefinition.TemplateDispatchAssignment,
			Instructions:    agentdefinition.TemplateDispatchAssignment.StarterInstructions(),
			ToolNames:       []string{"get_shipment", "search_shipments", "get_worker", "search_workers"},
			AutonomyCeiling: agent.TierActWithApproval,
			TriggerMode:     agentdefinition.TriggerScheduled,
			CronExpression:  agentdefinition.TemplateDispatchAssignment.StarterCron(),
			CronTimezone:    "America/Los_Angeles",
			OutputMode:      agentdefinition.OutputReport,
			SystemKey:       SystemKeyDispatchAssignment,
			Enabled:         true,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           "Morning operations digest",
			Description:    "A weekday summary of what is stuck, late or unassigned, ready before the desk opens.",
			Instructions: "Each weekday morning, review shipments that are late, moves without a driver, " +
				"and billing items that are blocked. Write a short digest grouped by urgency with the " +
				"pro numbers and names a dispatcher needs to act.",
			ToolNames:       []string{"search_shipments", "search_workers"},
			AutonomyCeiling: agent.TierPropose,
			TriggerMode:     agentdefinition.TriggerScheduled,
			CronExpression:  "0 6 * * 1-5",
			CronTimezone:    "America/Los_Angeles",
			OutputMode:      agentdefinition.OutputReport,
			Enabled:         true,
		},
	}
}

func (s *AgentDefinitionSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *AgentDefinitionSeed) CanRollback() bool {
	return true
}
