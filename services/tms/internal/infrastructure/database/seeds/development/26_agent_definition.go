package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds/base"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	SeedAgentDispatchName   = "Dispatch desk"
	SeedAgentBillingName    = "Billing exceptions"
	SeedAgentIntakeDeskName = "Inbox desk"
)

type AgentDefinitionSeed struct {
	seedhelpers.BaseSeed
}

func NewAgentDefinitionSeed() *AgentDefinitionSeed {
	seed := &AgentDefinitionSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"AgentDefinition",
		"2.1.0",
		"Seeds chat, event-driven and scheduled agents for the default organization",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedAdminAccount, seedhelpers.SeedSystemAgentDefinitions)
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
				Where(cols.SystemKey.IsNull()).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("count existing agents: %w", err)
			}
			if count > 0 {
				// Already seeded. Rather than doing nothing, bring the
				// templated agents up to their template's current tool list:
				// the catalog grows, and an agent seeded before a tool existed
				// otherwise stays unable to answer what it was created for. The
				// compliance agent shipped holding get_worker and search_worker
				// and could not look up an expiring medical card.
				if err = s.reconcileToolNames(ctx, tx, org.ID, org.BusinessUnitID); err != nil {
					return err
				}

				// A desk added to this list after a database was seeded is
				// inserted now, so a new agent shows up without a reset.
				return s.insertDefinitions(ctx, tx, sc, org.ID, org.BusinessUnitID, true)
			}

			if err = s.enableSystemAgents(ctx, tx, org.ID, org.BusinessUnitID); err != nil {
				return err
			}

			return s.insertDefinitions(ctx, tx, sc, org.ID, org.BusinessUnitID, false)
		},
	)
}

// insertDefinitions inserts the seeded agents, skipping any whose name the
// organization already has when onlyMissing is set.
func (s *AgentDefinitionSeed) insertDefinitions(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	orgID, buID pulid.ID,
	onlyMissing bool,
) error {
	present := make(map[string]struct{})
	if onlyMissing {
		cols := buncolgen.DefinitionColumns
		var names []string
		if err := tx.NewSelect().
			Model((*agentdefinition.Definition)(nil)).
			Column(cols.Name.Name).
			Where(cols.OrganizationID.Eq(), orgID).
			Where(cols.BusinessUnitID.Eq(), buID).
			Scan(ctx, &names); err != nil {
			return fmt.Errorf("list existing agents: %w", err)
		}
		for _, name := range names {
			present[name] = struct{}{}
		}
	}

	now := timeutils.NowUnix()
	for _, definition := range s.definitions(orgID, buID) {
		if _, ok := present[definition.Name]; ok {
			continue
		}

		definition.ApplyDefaults()
		if definition.TriggerMode == agentdefinition.TriggerScheduled ||
			definition.TriggerMode == agentdefinition.TriggerContinuous {
			next, err := definition.ComputeNextRun(now)
			if err != nil {
				return fmt.Errorf("schedule agent %s: %w", definition.Name, err)
			}
			definition.NextRunAt = &next
		}
		if _, err := tx.NewInsert().Model(definition).Exec(ctx); err != nil {
			return fmt.Errorf("insert agent %s: %w", definition.Name, err)
		}
		if err := sc.TrackCreated(ctx, "agent_definitions", definition.ID, s.Name()); err != nil {
			return err
		}
	}

	return nil
}

// reconcileToolNames adds any tool a definition's template has gained.
//
// Strictly additive: a tool already present keeps its position, and one a
// developer removed by hand comes back, which is the accepted cost of a seeder
// whose job is to reflect the current templates. It never removes, so a tool
// picked by hand survives, and it runs only here — in the development seeds —
// so no organization's configured agent is touched by it.
func (s *AgentDefinitionSeed) reconcileToolNames(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) error {
	cols := buncolgen.DefinitionColumns

	var existing []*agentdefinition.Definition
	if err := tx.NewSelect().
		Model(&existing).
		Where(cols.OrganizationID.Eq(), orgID).
		Where(cols.BusinessUnitID.Eq(), buID).
		Where(cols.Template.IsNotNull()).
		Scan(ctx); err != nil {
		return fmt.Errorf("load agents to reconcile: %w", err)
	}

	for _, definition := range existing {
		merged, changed := mergeToolNames(definition.ToolNames, definition.Template.StarterTools())
		if !changed {
			continue
		}

		if _, err := tx.NewUpdate().
			Model((*agentdefinition.Definition)(nil)).
			Set(cols.ToolNames.Set(), dbhelper.TextArray(merged)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Where(cols.ID.Eq(), definition.ID).
			Where(cols.OrganizationID.Eq(), orgID).
			Where(cols.BusinessUnitID.Eq(), buID).
			Exec(ctx); err != nil {
			return fmt.Errorf("reconcile agent %s: %w", definition.Name, err)
		}
	}

	return nil
}

// mergeToolNames appends the starters a definition is missing, preserving the
// order it already had so a developer's arrangement is not reshuffled.
func mergeToolNames(current, starters []string) ([]string, bool) {
	held := make(map[string]struct{}, len(current))
	for _, tool := range current {
		held[tool] = struct{}{}
	}

	merged := current
	changed := false
	for _, tool := range starters {
		if _, ok := held[tool]; ok {
			continue
		}
		merged = append(merged, tool)
		held[tool] = struct{}{}
		changed = true
	}

	return merged, changed
}

func (s *AgentDefinitionSeed) definitions(orgID, buID pulid.ID) []*agentdefinition.Definition {
	return []*agentdefinition.Definition{
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAgentDispatchName,
			Icon:           agentdefinition.IconTruck,
			Accent:         agentdefinition.AccentIndigo,
			Description:    "Looks up shipments and drivers for the dispatch team.",
			Template:       agentdefinition.TemplateDispatchAssistant,
			Instructions: agentdefinition.TemplateDispatchAssistant.StarterInstructions() + "\n\n" +
				"We run mostly reefer freight out of the Los Angeles terminal. When a driver is " +
				"asked about, check hours of service before anything else.",
			Guardrails:      []string{"Promise a delivery time to a customer", "Change a rate"},
			ToolNames:       agentdefinition.TemplateDispatchAssistant.StarterTools(),
			AutonomyCeiling: agent.TierActWithApproval,
			TriggerMode:     agentdefinition.TriggerChat,
			Enabled:         true,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAgentBillingName,
			Icon:           agentdefinition.IconReceipt,
			Accent:         agentdefinition.AccentAmber,
			Description:    "Works blocked billing queue items and proposes what to do about them.",
			Template:       agentdefinition.TemplateBillingAssistant,
			Instructions: agentdefinition.TemplateBillingAssistant.StarterInstructions() + "\n\n" +
				"Never propose a rate change. If a rate looks wrong, flag it for a person and say " +
				"which document you would want to see.",
			ToolNames:       agentdefinition.TemplateBillingAssistant.StarterTools(),
			AutonomyCeiling: agent.TierPropose,
			TriggerMode:     agentdefinition.TriggerChat,
			Enabled:         true,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           "Compliance desk",
			Icon:           agentdefinition.IconShield,
			Accent:         agentdefinition.AccentTeal,
			Description:    "Answers questions about driver qualification and what is expiring.",
			Template:       agentdefinition.TemplateComplianceAssistant,
			Instructions: agentdefinition.TemplateComplianceAssistant.StarterInstructions() + "\n\n" +
				"Medical cards and hazmat endorsements are the two we most often miss.",
			ToolNames:       agentdefinition.TemplateComplianceAssistant.StarterTools(),
			AutonomyCeiling: agent.TierPropose,
			TriggerMode:     agentdefinition.TriggerChat,
			Enabled:         true,
		},
		{
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Name:            "Customer service",
			Icon:            agentdefinition.IconHeadset,
			Accent:          agentdefinition.AccentRose,
			Description:     "Shipment status for the customer-facing team. Off until reviewed.",
			Template:        agentdefinition.TemplateCustomerAssistant,
			Instructions:    agentdefinition.TemplateCustomerAssistant.StarterInstructions(),
			ToolNames:       agentdefinition.TemplateCustomerAssistant.StarterTools(),
			AutonomyCeiling: agent.TierPropose,
			TriggerMode:     agentdefinition.TriggerChat,
			Enabled:         false,
		},
		{
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Name:            "Help",
			Icon:            agentdefinition.IconSparkle,
			Accent:          agentdefinition.AccentSlate,
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
			Name:           "Morning operations digest",
			Icon:           agentdefinition.IconGauge,
			Accent:         agentdefinition.AccentSky,
			Description:    "A weekday summary of what is stuck, late or unassigned, ready before the desk opens.",
			Instructions: "Each weekday morning, review shipments that are late, moves without a driver, " +
				"and billing items that are blocked. Write a short digest grouped by urgency with the " +
				"pro numbers and names a dispatcher needs to act.",
			// No template, so this one names its own tools — and has to be read
			// whenever the catalog grows, which is exactly the maintenance the
			// templated definitions above no longer need.
			ToolNames: []string{
				"list_shipments",
				"search_shipments",
				"list_workers",
				"list_time_off",
				"list_expiring_credentials",
			},
			AutonomyCeiling: agent.TierPropose,
			TriggerMode:     agentdefinition.TriggerScheduled,
			CronExpression:  "0 6 * * 1-5",
			CronTimezone:    "America/Los_Angeles",
			OutputMode:      agentdefinition.OutputReport,
			Enabled:         true,
		},
		{
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Name:            SeedAgentIntakeDeskName,
			Icon:            agentdefinition.IconInbox,
			Accent:          agentdefinition.AccentViolet,
			Description:     agentdefinition.TemplateIntakeDesk.Description(),
			Template:        agentdefinition.TemplateIntakeDesk,
			Instructions:    agentdefinition.TemplateIntakeDesk.StarterInstructions(),
			ToolNames:       agentdefinition.TemplateIntakeDesk.StarterTools(),
			AutonomyCeiling: agentdefinition.TemplateIntakeDesk.StarterCeiling(),
			TriggerMode:     agentdefinition.TemplateIntakeDesk.StarterTrigger(),
			EventKinds:      agentdefinition.TemplateIntakeDesk.StarterEvents(),
			OutputMode:      agentdefinition.OutputReport,
			// Shadowed: in development the desk proposes against the seeded
			// inbox without sending real mail until somebody turns it loose.
			ShadowMode: true,
			ContextProviders: []agentdefinition.ContextProvider{
				agentdefinition.ContextOrganization,
				agentdefinition.ContextClock,
				agentdefinition.ContextTools,
			},
			Enabled: true,
		},
	}
}

func (s *AgentDefinitionSeed) enableSystemAgents(
	ctx context.Context,
	tx bun.Tx,
	orgID, buID pulid.ID,
) error {
	cols := buncolgen.DefinitionColumns
	now := timeutils.NowUnix()
	for _, definition := range base.SystemAgentDefinitions(orgID, buID) {
		query := tx.NewUpdate().
			Model((*agentdefinition.Definition)(nil)).
			Set(cols.Enabled.Set(), true).
			Set(cols.ShadowMode.Set(), true).
			Set(cols.UpdatedAt.Set(), now).
			Where(cols.OrganizationID.Eq(), orgID).
			Where(cols.BusinessUnitID.Eq(), buID).
			Where(cols.SystemKey.Eq(), definition.SystemKey)
		if definition.TriggerMode == agentdefinition.TriggerScheduled {
			next, err := definition.ComputeNextRun(now)
			if err != nil {
				return fmt.Errorf("schedule system agent %s: %w", definition.SystemKey, err)
			}
			query = query.Set(cols.NextRunAt.Set(), next).
				Set(cols.CronTimezone.Set(), "America/Los_Angeles")
		}
		if _, err := query.Exec(ctx); err != nil {
			return fmt.Errorf("enable system agent %s: %w", definition.SystemKey, err)
		}
	}

	return nil
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
