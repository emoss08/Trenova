package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	SeedAgentDispatchName   = "Dispatch desk"
	SeedAgentBillingName    = "Billing exceptions"
	SeedAgentIntakeDeskName = "Inbox desk"

	SeedAgentSettlementsName = "Settlements clerk"
	SeedAgentReceivablesName = "Receivables"

	SeedAgentMasterDataName = "Master data steward"
	SeedAgentWorkforceName  = "Workforce coordinator"
	SeedAgentFuelTaxName    = "Fuel and IFTA clerk"
	SeedAgentReportName     = "Report analyst"
)

type AgentDefinitionSeed struct {
}

var AgentDefinition = Seeds.Add(seed.Seed{
	Name:        "agent_definition",
	Kind:        seed.Fixture,
	Description: "Seeds chat, event-driven and scheduled agents for the default organization",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&AgentDefinitionSeed{}).Run,
})

func (s *AgentDefinitionSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	count, err := agentdefinitiondb.Definitions.AllTenants().Where(
		agentdefinitiondb.Definitions.OrganizationID.Eq(org.ID),
		agentdefinitiondb.Definitions.BusinessUnitID.Eq(org.BusinessUnitID),
		agentdefinitiondb.Definitions.SystemKey.IsNull(),
	).Count(ctx, db)
	if err != nil {
		return fmt.Errorf("count existing agents: %w", err)
	}
	if count > 0 {
		// Already seeded. Rather than doing nothing, bring the
		// templated agents up to their template's current tool list:
		// the catalog grows, and an agent seeded before a tool existed
		// otherwise stays unable to answer what it was created for. The
		// compliance agent shipped holding worker_get and the worker name
		// search and could not look up an expiring medical card.
		if err = s.reconcileToolNames(ctx, db, org.ID, org.BusinessUnitID); err != nil {
			return err
		}

		// A desk added to this list after a database was seeded is
		// inserted now, so a new agent shows up without a reset.
		return s.insertDefinitions(ctx, db, sr, org.ID, org.BusinessUnitID, true)
	}

	if err = s.enableSystemAgents(ctx, db, org.ID, org.BusinessUnitID); err != nil {
		return err
	}

	return s.insertDefinitions(ctx, db, sr, org.ID, org.BusinessUnitID, false)
}

// insertDefinitions inserts the seeded agents, skipping any whose name the
// organization already has when onlyMissing is set.
func (s *AgentDefinitionSeed) insertDefinitions(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	orgID, buID pulid.ID,
	onlyMissing bool,
) error {
	present := make(map[string]struct{})
	if onlyMissing {
		d := agentdefinitiondb.Definitions
		rows, err := d.AllTenants().Pick(d.Name).Where(
			d.OrganizationID.Eq(orgID),
			d.BusinessUnitID.Eq(buID),
		).All(ctx, db)
		if err != nil {
			return fmt.Errorf("list existing agents: %w", err)
		}
		for _, row := range rows {
			present[row.Name] = struct{}{}
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
		if err := agentdefinitiondb.Definitions.Create(ctx, db, definition); err != nil {
			return fmt.Errorf("insert agent %s: %w", definition.Name, err)
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
	db *keel.DB,
	orgID, buID pulid.ID,
) error {
	existing, err := agentdefinitiondb.Definitions.AllTenants().Where(
		agentdefinitiondb.Definitions.OrganizationID.Eq(orgID),
		agentdefinitiondb.Definitions.BusinessUnitID.Eq(buID),
		agentdefinitiondb.Definitions.Template.IsNotNull(),
	).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load agents to reconcile: %w", err)
	}

	for _, definition := range existing {
		merged, changed := mergeToolNames(definition.ToolNames, definition.Template.StarterTools())
		if !changed {
			continue
		}

		d := agentdefinitiondb.Definitions
		definition.ToolNames = merged
		if err := d.Save(ctx, db, definition, d.ToolNames); err != nil {
			return fmt.Errorf("reconcile agent %s: %w", definition.Name, err)
		}
	}

	return nil
}

// mergeToolNames appends the starters a definition is missing, preserving the
// order it already had so a developer's arrangement is not reshuffled. It stops
// at the tool cap: nothing is removed, so a template that moved tools elsewhere
// and gained others would otherwise grow an agent past what it may hold.
func mergeToolNames(current, starters []string) ([]string, bool) {
	held := make(map[string]struct{}, len(current))
	for _, tool := range current {
		held[tool] = struct{}{}
	}

	merged := current
	changed := false
	for _, tool := range starters {
		if len(merged) >= agentdefinition.MaxTools {
			break
		}
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
			Guardrails: []string{"Promise a delivery time to a customer", "Change a rate"},
			ToolNames: append(
				agentdefinition.TemplateDispatchAssistant.StarterTools(),
				"dispatch_get_board",
				"worker_send_message",
			),
			AutonomyCeiling:   agent.TierActWithApproval,
			DataAccessCeiling: agentdefinition.TemplateDispatchAssistant.StarterDataAccess(),
			TriggerMode:       agentdefinition.TriggerChat,
			Enabled:           true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
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
			ToolNames:         agentdefinition.TemplateBillingAssistant.StarterTools(),
			AutonomyCeiling:   agent.TierPropose,
			DataAccessCeiling: agentdefinition.TemplateBillingAssistant.StarterDataAccess(),
			TriggerMode:       agentdefinition.TriggerChat,
			Enabled:           true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			Name:              SeedAgentSettlementsName,
			Icon:              agentdefinition.IconBanknote,
			Accent:            agentdefinition.AccentEmeral,
			Description:       agentdefinition.TemplateSettlementsClerk.Description(),
			Template:          agentdefinition.TemplateSettlementsClerk,
			Instructions:      agentdefinition.TemplateSettlementsClerk.StarterInstructions(),
			ToolNames:         agentdefinition.TemplateSettlementsClerk.StarterTools(),
			AutonomyCeiling:   agentdefinition.TemplateSettlementsClerk.StarterCeiling(),
			DataAccessCeiling: agentdefinition.TemplateSettlementsClerk.StarterDataAccess(),
			TriggerMode:       agentdefinition.TriggerChat,
			Enabled:           true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			Name:              SeedAgentReceivablesName,
			Icon:              agentdefinition.IconCoins,
			Accent:            agentdefinition.AccentSky,
			Description:       agentdefinition.TemplateReceivables.Description(),
			Template:          agentdefinition.TemplateReceivables,
			Instructions:      agentdefinition.TemplateReceivables.StarterInstructions(),
			ToolNames:         agentdefinition.TemplateReceivables.StarterTools(),
			AutonomyCeiling:   agentdefinition.TemplateReceivables.StarterCeiling(),
			DataAccessCeiling: agentdefinition.TemplateReceivables.StarterDataAccess(),
			TriggerMode:       agentdefinition.TriggerChat,
			Enabled:           true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			Name:              SeedAgentMasterDataName,
			Icon:              agentdefinition.IconFile,
			Accent:            agentdefinition.AccentSlate,
			Description:       agentdefinition.TemplateMasterDataSteward.Description(),
			Template:          agentdefinition.TemplateMasterDataSteward,
			Instructions:      agentdefinition.TemplateMasterDataSteward.StarterInstructions(),
			ToolNames:         agentdefinition.TemplateMasterDataSteward.StarterTools(),
			AutonomyCeiling:   agentdefinition.TemplateMasterDataSteward.StarterCeiling(),
			DataAccessCeiling: agentdefinition.TemplateMasterDataSteward.StarterDataAccess(),
			TriggerMode:       agentdefinition.TriggerChat,
			Enabled:           true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			Name:              SeedAgentWorkforceName,
			Icon:              agentdefinition.IconClipboard,
			Accent:            agentdefinition.AccentViolet,
			Description:       agentdefinition.TemplateWorkforceCoordinator.Description(),
			Template:          agentdefinition.TemplateWorkforceCoordinator,
			Instructions:      agentdefinition.TemplateWorkforceCoordinator.StarterInstructions(),
			ToolNames:         agentdefinition.TemplateWorkforceCoordinator.StarterTools(),
			AutonomyCeiling:   agentdefinition.TemplateWorkforceCoordinator.StarterCeiling(),
			DataAccessCeiling: agentdefinition.TemplateWorkforceCoordinator.StarterDataAccess(),
			TriggerMode:       agentdefinition.TriggerChat,
			Enabled:           true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			Name:              SeedAgentFuelTaxName,
			Icon:              agentdefinition.IconGauge,
			Accent:            agentdefinition.AccentAmber,
			Description:       agentdefinition.TemplateFuelTaxClerk.Description(),
			Template:          agentdefinition.TemplateFuelTaxClerk,
			Instructions:      agentdefinition.TemplateFuelTaxClerk.StarterInstructions(),
			ToolNames:         agentdefinition.TemplateFuelTaxClerk.StarterTools(),
			AutonomyCeiling:   agentdefinition.TemplateFuelTaxClerk.StarterCeiling(),
			DataAccessCeiling: agentdefinition.TemplateFuelTaxClerk.StarterDataAccess(),
			TriggerMode:       agentdefinition.TriggerChat,
			Enabled:           true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			Name:              SeedAgentReportName,
			Icon:              agentdefinition.IconCompass,
			Accent:            agentdefinition.AccentIndigo,
			Description:       agentdefinition.TemplateReportAnalyst.Description(),
			Template:          agentdefinition.TemplateReportAnalyst,
			Instructions:      agentdefinition.TemplateReportAnalyst.StarterInstructions(),
			ToolNames:         agentdefinition.TemplateReportAnalyst.StarterTools(),
			AutonomyCeiling:   agentdefinition.TemplateReportAnalyst.StarterCeiling(),
			DataAccessCeiling: agentdefinition.TemplateReportAnalyst.StarterDataAccess(),
			TriggerMode:       agentdefinition.TriggerChat,
			Enabled:           true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
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
			ToolNames:         agentdefinition.TemplateComplianceAssistant.StarterTools(),
			AutonomyCeiling:   agent.TierPropose,
			DataAccessCeiling: agentdefinition.TemplateComplianceAssistant.StarterDataAccess(),
			TriggerMode:       agentdefinition.TriggerChat,
			Enabled:           true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			Name:              "Customer service",
			Icon:              agentdefinition.IconHeadset,
			Accent:            agentdefinition.AccentRose,
			Description:       "Shipment status for the customer-facing team. Off until reviewed.",
			Template:          agentdefinition.TemplateCustomerAssistant,
			Instructions:      agentdefinition.TemplateCustomerAssistant.StarterInstructions(),
			ToolNames:         agentdefinition.TemplateCustomerAssistant.StarterTools(),
			AutonomyCeiling:   agent.TierPropose,
			DataAccessCeiling: agentdefinition.TemplateCustomerAssistant.StarterDataAccess(),
			TriggerMode:       agentdefinition.TriggerChat,
			Enabled:           false,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           "Help",
			Icon:           agentdefinition.IconSparkle,
			Accent:         agentdefinition.AccentSlate,
			Description: "Explains how to do things in Trenova and answers quick questions from " +
				"the command palette. Reads what the person may read; changes nothing.",
			Template:          agentdefinition.TemplateGeneralAssistant,
			Instructions:      agentdefinition.TemplateGeneralAssistant.StarterInstructions(),
			AutonomyCeiling:   agent.TierPropose,
			DataAccessCeiling: agentdefinition.TemplateGeneralAssistant.StarterDataAccess(),
			TriggerMode:       agentdefinition.TriggerChat,
			Enabled:           true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
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
				"shipment_list",
				"worker_list",
				"worker_list_time_off",
				"worker_list_credentials",
			},
			AutonomyCeiling: agent.TierPropose,
			TriggerMode:     agentdefinition.TriggerScheduled,
			CronExpression:  "0 6 * * 1-5",
			CronTimezone:    "America/Los_Angeles",
			OutputMode:      agentdefinition.OutputReport,
			Enabled:         true,

			DataAccessCeiling: agentdefinition.DataAccessInternal,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
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
				// Corrections people make to its proposals reach it only
				// through memory; without it a fix was made again every run.
				agentdefinition.ContextMemory,
			},
			Enabled: true,

			DataAccessCeiling: agentdefinition.DataAccessInternal,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		watchtowerDesk(orgID, buID, agentdefinition.TemplateLoadMonitor,
			agentdefinition.IconRadar, agentdefinition.AccentSky),
		watchtowerDesk(orgID, buID, agentdefinition.TemplateDetentionDesk,
			agentdefinition.IconGauge, agentdefinition.AccentAmber),
		watchtowerDesk(orgID, buID, agentdefinition.TemplateCredentialDesk,
			agentdefinition.IconClipboard, agentdefinition.AccentTeal),
		watchtowerDesk(orgID, buID, agentdefinition.TemplateCarrierRiskDesk,
			agentdefinition.IconShield, agentdefinition.AccentRose),
		watchtowerDesk(orgID, buID, agentdefinition.TemplateEDIDesk,
			agentdefinition.IconFile, agentdefinition.AccentIndigo),
		watchtowerDesk(orgID, buID, agentdefinition.TemplateServiceFailureDesk,
			agentdefinition.IconHeadset, agentdefinition.AccentViolet),
		watchtowerDesk(orgID, buID, agentdefinition.TemplateCashApplication,
			agentdefinition.IconBanknote, agentdefinition.AccentEmeral),
	}
}

func watchtowerDesk(
	orgID, buID pulid.ID,
	template agentdefinition.Template,
	icon, accent string,
) *agentdefinition.Definition {
	return &agentdefinition.Definition{
		OrganizationID:  orgID,
		BusinessUnitID:  buID,
		Name:            template.Label(),
		Icon:            icon,
		Accent:          accent,
		Description:     template.Description(),
		Template:        template,
		Instructions:    template.StarterInstructions(),
		ToolNames:       template.StarterTools(),
		AutonomyCeiling: template.StarterCeiling(),
		TriggerMode:     template.StarterTrigger(),
		EventKinds:      template.StarterEvents(),
		CronExpression:  template.StarterCron(),
		CronTimezone:    "America/Los_Angeles",
		OutputMode:      agentdefinition.OutputReport,
		ShadowMode:      true,
		ContextProviders: []agentdefinition.ContextProvider{
			agentdefinition.ContextOrganization,
			agentdefinition.ContextClock,
			agentdefinition.ContextTools,
			agentdefinition.ContextMemory,
		},
		Enabled:            true,
		DataAccessCeiling:  agentdefinition.DataAccessInternal,
		MaxChangeItems:     500,
		BusinessHoursStart: 420,
		BusinessHoursEnd:   1080,
		AccessMode:         agentdefinition.AccessEveryone,
	}
}

func (s *AgentDefinitionSeed) enableSystemAgents(
	ctx context.Context,
	db *keel.DB,
	orgID, buID pulid.ID,
) error {
	d := agentdefinitiondb.Definitions
	now := timeutils.NowUnix()
	for _, definition := range tenantbootstrap.SystemAgentDefinitions(orgID, buID) {
		sets := []keel.Assignment{d.Enabled.To(true), d.ShadowMode.To(true)}
		if definition.TriggerMode == agentdefinition.TriggerScheduled {
			next, err := definition.ComputeNextRun(now)
			if err != nil {
				return fmt.Errorf("schedule system agent %s: %w", definition.SystemKey, err)
			}
			sets = append(sets, d.NextRunAt.To(next), d.CronTimezone.To("America/Los_Angeles"))
		}
		if _, err := d.AllTenants().Where(
			d.OrganizationID.Eq(orgID),
			d.BusinessUnitID.Eq(buID),
			d.SystemKey.Eq(definition.SystemKey),
		).Update(ctx, db, sets...); err != nil {
			return fmt.Errorf("enable system agent %s: %w", definition.SystemKey, err)
		}
	}

	return nil
}
