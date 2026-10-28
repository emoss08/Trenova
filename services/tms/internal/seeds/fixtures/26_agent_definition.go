package fixtures

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentinstructions"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmark"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
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
		merged, toolsChanged := mergeToolNames(
			definition.ToolNames,
			agentpreset.StarterTools(definition.Template),
		)
		access, accessRaised := raisedDataAccess(
			definition.DataAccessCeiling,
			agentpreset.StarterDataAccess(definition.Template),
		)
		if !toolsChanged && !accessRaised {
			continue
		}

		d := agentdefinitiondb.Definitions
		definition.ToolNames = merged
		definition.DataAccessCeiling = access
		if err := d.Save(ctx, db, definition, d.ToolNames, d.DataAccessCeiling); err != nil {
			return fmt.Errorf("reconcile agent %s: %w", definition.Name, err)
		}
	}

	return nil
}

func raisedDataAccess(
	current, starter agentdefinition.DataAccessCeiling,
) (agentdefinition.DataAccessCeiling, bool) {
	if current.Sensitivity().Level() >= starter.Sensitivity().Level() {
		return current, false
	}

	return starter, true
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
			Icon:           agentmark.IconTruck,
			Accent:         agentmark.AccentIndigo,
			Description:    "Looks up shipments and drivers for the dispatch team.",
			Template:       agenttemplate.TemplateDispatchAssistant,
			Instructions: agentinstructions.StarterInstructions(
				agenttemplate.TemplateDispatchAssistant,
			) + "\n\n" +
				"We run mostly reefer freight out of the Los Angeles terminal. When a driver is " +
				"asked about, check hours of service before anything else.",
			Guardrails: []string{"Promise a delivery time to a customer", "Change a rate"},
			ToolNames: append(
				agentpreset.StarterTools(agenttemplate.TemplateDispatchAssistant),
				"dispatch_get_board",
				"worker_send_message",
			),
			AutonomyCeiling: agent.TierActWithApproval,
			DataAccessCeiling: agentpreset.StarterDataAccess(
				agenttemplate.TemplateDispatchAssistant,
			),
			TriggerMode: agentdefinition.TriggerChat,
			Enabled:     true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAgentBillingName,
			Icon:           agentmark.IconReceipt,
			Accent:         agentmark.AccentAmber,
			Description:    "Works blocked billing queue items and proposes what to do about them.",
			Template:       agenttemplate.TemplateBillingAssistant,
			Instructions: agentinstructions.StarterInstructions(
				agenttemplate.TemplateBillingAssistant,
			) + "\n\n" +
				"Never propose a rate change. If a rate looks wrong, flag it for a person and say " +
				"which document you would want to see.",
			ToolNames:       agentpreset.StarterTools(agenttemplate.TemplateBillingAssistant),
			AutonomyCeiling: agent.TierPropose,
			DataAccessCeiling: agentpreset.StarterDataAccess(
				agenttemplate.TemplateBillingAssistant,
			),
			TriggerMode: agentdefinition.TriggerChat,
			Enabled:     true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAgentSettlementsName,
			Icon:           agentmark.IconBanknote,
			Accent:         agentmark.AccentEmeral,
			Description:    agenttemplate.Description(agenttemplate.TemplateSettlementsClerk),
			Template:       agenttemplate.TemplateSettlementsClerk,
			Instructions: agentinstructions.StarterInstructions(
				agenttemplate.TemplateSettlementsClerk,
			),
			ToolNames: agentpreset.StarterTools(agenttemplate.TemplateSettlementsClerk),
			AutonomyCeiling: agentpreset.StarterCeiling(
				agenttemplate.TemplateSettlementsClerk,
			),
			DataAccessCeiling: agentpreset.StarterDataAccess(
				agenttemplate.TemplateSettlementsClerk,
			),
			TriggerMode: agentdefinition.TriggerChat,
			Enabled:     true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAgentReceivablesName,
			Icon:           agentmark.IconCoins,
			Accent:         agentmark.AccentSky,
			Description:    agenttemplate.Description(agenttemplate.TemplateReceivables),
			Template:       agenttemplate.TemplateReceivables,
			Instructions: agentinstructions.StarterInstructions(
				agenttemplate.TemplateReceivables,
			),
			ToolNames:         agentpreset.StarterTools(agenttemplate.TemplateReceivables),
			AutonomyCeiling:   agentpreset.StarterCeiling(agenttemplate.TemplateReceivables),
			DataAccessCeiling: agentpreset.StarterDataAccess(agenttemplate.TemplateReceivables),
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
			Name:           SeedAgentMasterDataName,
			Icon:           agentmark.IconFile,
			Accent:         agentmark.AccentSlate,
			Description:    agenttemplate.Description(agenttemplate.TemplateMasterDataSteward),
			Template:       agenttemplate.TemplateMasterDataSteward,
			Instructions: agentinstructions.StarterInstructions(
				agenttemplate.TemplateMasterDataSteward,
			),
			ToolNames: agentpreset.StarterTools(
				agenttemplate.TemplateMasterDataSteward,
			),
			AutonomyCeiling: agentpreset.StarterCeiling(
				agenttemplate.TemplateMasterDataSteward,
			),
			DataAccessCeiling: agentpreset.StarterDataAccess(
				agenttemplate.TemplateMasterDataSteward,
			),
			TriggerMode: agentdefinition.TriggerChat,
			Enabled:     true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAgentWorkforceName,
			Icon:           agentmark.IconClipboard,
			Accent:         agentmark.AccentViolet,
			Description: agenttemplate.Description(
				agenttemplate.TemplateWorkforceCoordinator,
			),
			Template: agenttemplate.TemplateWorkforceCoordinator,
			Instructions: agentinstructions.StarterInstructions(
				agenttemplate.TemplateWorkforceCoordinator,
			),
			ToolNames: agentpreset.StarterTools(
				agenttemplate.TemplateWorkforceCoordinator,
			),
			AutonomyCeiling: agentpreset.StarterCeiling(
				agenttemplate.TemplateWorkforceCoordinator,
			),
			DataAccessCeiling: agentpreset.StarterDataAccess(
				agenttemplate.TemplateWorkforceCoordinator,
			),
			TriggerMode: agentdefinition.TriggerChat,
			Enabled:     true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAgentFuelTaxName,
			Icon:           agentmark.IconGauge,
			Accent:         agentmark.AccentAmber,
			Description:    agenttemplate.Description(agenttemplate.TemplateFuelTaxClerk),
			Template:       agenttemplate.TemplateFuelTaxClerk,
			Instructions: agentinstructions.StarterInstructions(
				agenttemplate.TemplateFuelTaxClerk,
			),
			ToolNames:       agentpreset.StarterTools(agenttemplate.TemplateFuelTaxClerk),
			AutonomyCeiling: agentpreset.StarterCeiling(agenttemplate.TemplateFuelTaxClerk),
			DataAccessCeiling: agentpreset.StarterDataAccess(
				agenttemplate.TemplateFuelTaxClerk,
			),
			TriggerMode: agentdefinition.TriggerChat,
			Enabled:     true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAgentReportName,
			Icon:           agentmark.IconCompass,
			Accent:         agentmark.AccentIndigo,
			Description:    agenttemplate.Description(agenttemplate.TemplateReportAnalyst),
			Template:       agenttemplate.TemplateReportAnalyst,
			Instructions: agentinstructions.StarterInstructions(
				agenttemplate.TemplateReportAnalyst,
			),
			ToolNames:       agentpreset.StarterTools(agenttemplate.TemplateReportAnalyst),
			AutonomyCeiling: agentpreset.StarterCeiling(agenttemplate.TemplateReportAnalyst),
			DataAccessCeiling: agentpreset.StarterDataAccess(
				agenttemplate.TemplateReportAnalyst,
			),
			TriggerMode: agentdefinition.TriggerChat,
			Enabled:     true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           "Compliance desk",
			Icon:           agentmark.IconShield,
			Accent:         agentmark.AccentTeal,
			Description:    "Answers questions about driver qualification and what is expiring.",
			Template:       agenttemplate.TemplateComplianceAssistant,
			Instructions: agentinstructions.StarterInstructions(
				agenttemplate.TemplateComplianceAssistant,
			) + "\n\n" +
				"Medical cards and hazmat endorsements are the two we most often miss.",
			ToolNames: agentpreset.StarterTools(
				agenttemplate.TemplateComplianceAssistant,
			),
			AutonomyCeiling: agent.TierPropose,
			DataAccessCeiling: agentpreset.StarterDataAccess(
				agenttemplate.TemplateComplianceAssistant,
			),
			TriggerMode: agentdefinition.TriggerChat,
			Enabled:     true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           "Customer service",
			Icon:           agentmark.IconHeadset,
			Accent:         agentmark.AccentRose,
			Description:    "Shipment status for the customer-facing team. Off until reviewed.",
			Template:       agenttemplate.TemplateCustomerAssistant,
			Instructions: agentinstructions.StarterInstructions(
				agenttemplate.TemplateCustomerAssistant,
			),
			ToolNames: agentpreset.StarterTools(
				agenttemplate.TemplateCustomerAssistant,
			),
			AutonomyCeiling: agent.TierPropose,
			DataAccessCeiling: agentpreset.StarterDataAccess(
				agenttemplate.TemplateCustomerAssistant,
			),
			TriggerMode: agentdefinition.TriggerChat,
			Enabled:     false,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           "Help",
			Icon:           agentmark.IconSparkle,
			Accent:         agentmark.AccentSlate,
			Description: "Explains how to do things in Trenova and answers quick questions from " +
				"the command palette. Reads what the person may read; changes nothing.",
			Template: agenttemplate.TemplateGeneralAssistant,
			Instructions: agentinstructions.StarterInstructions(
				agenttemplate.TemplateGeneralAssistant,
			),
			AutonomyCeiling: agent.TierPropose,
			DataAccessCeiling: agentpreset.StarterDataAccess(
				agenttemplate.TemplateGeneralAssistant,
			),
			TriggerMode: agentdefinition.TriggerChat,
			Enabled:     true,

			MaxChangeItems: 500,

			BusinessHoursStart: 420,

			BusinessHoursEnd: 1080,

			AccessMode: agentdefinition.AccessEveryone,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           "Morning operations digest",
			Icon:           agentmark.IconGauge,
			Accent:         agentmark.AccentSky,
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
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAgentIntakeDeskName,
			Icon:           agentmark.IconInbox,
			Accent:         agentmark.AccentViolet,
			Description:    agenttemplate.Description(agenttemplate.TemplateIntakeDesk),
			Template:       agenttemplate.TemplateIntakeDesk,
			Instructions: agentinstructions.StarterInstructions(
				agenttemplate.TemplateIntakeDesk,
			),
			ToolNames:       agentpreset.StarterTools(agenttemplate.TemplateIntakeDesk),
			AutonomyCeiling: agentpreset.StarterCeiling(agenttemplate.TemplateIntakeDesk),
			TriggerMode:     agentpreset.StarterTrigger(agenttemplate.TemplateIntakeDesk),
			EventKinds:      agentpreset.StarterEvents(agenttemplate.TemplateIntakeDesk),
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
		watchtowerDesk(orgID, buID, agenttemplate.TemplateLoadMonitor,
			agentmark.IconRadar, agentmark.AccentSky),
		watchtowerDesk(orgID, buID, agenttemplate.TemplateDetentionDesk,
			agentmark.IconGauge, agentmark.AccentAmber),
		watchtowerDesk(orgID, buID, agenttemplate.TemplateCredentialDesk,
			agentmark.IconClipboard, agentmark.AccentTeal),
		watchtowerDesk(orgID, buID, agenttemplate.TemplateCarrierRiskDesk,
			agentmark.IconShield, agentmark.AccentRose),
		watchtowerDesk(orgID, buID, agenttemplate.TemplateEDIDesk,
			agentmark.IconFile, agentmark.AccentIndigo),
		watchtowerDesk(orgID, buID, agenttemplate.TemplateServiceFailureDesk,
			agentmark.IconHeadset, agentmark.AccentViolet),
		watchtowerDesk(orgID, buID, agenttemplate.TemplateCashApplication,
			agentmark.IconBanknote, agentmark.AccentEmeral),
	}
}

func watchtowerDesk(
	orgID, buID pulid.ID,
	template agenttemplate.Template,
	icon, accent string,
) *agentdefinition.Definition {
	return &agentdefinition.Definition{
		OrganizationID:  orgID,
		BusinessUnitID:  buID,
		Name:            agenttemplate.Label(template),
		Icon:            icon,
		Accent:          accent,
		Description:     agenttemplate.Description(template),
		Template:        template,
		Instructions:    agentinstructions.StarterInstructions(template),
		ToolNames:       agentpreset.StarterTools(template),
		AutonomyCeiling: agentpreset.StarterCeiling(template),
		TriggerMode:     agentpreset.StarterTrigger(template),
		EventKinds:      agentpreset.StarterEvents(template),
		CronExpression:  agentpreset.StarterCron(template),
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
