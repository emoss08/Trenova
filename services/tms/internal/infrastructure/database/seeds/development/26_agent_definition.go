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
	"github.com/uptrace/bun"
)

// Names the conversation seed looks agents up by.
const (
	SeedAgentDispatchName = "Dispatch desk"
	SeedAgentBillingName  = "Billing exceptions"
)

type AgentDefinitionSeed struct {
	seedhelpers.BaseSeed
}

// AgentDefinitionSeed configures one agent per template so the assistant's
// picker, the Agent Control page and the conversation seed all have something
// to show. Tool names are the registered read and write tools each template
// may reach; the general assistant gets none, because it may not.
//
// Depends on:
//   - AdminAccount: the default organization the agents belong to
func NewAgentDefinitionSeed() *AgentDefinitionSeed {
	seed := &AgentDefinitionSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"AgentDefinition",
		"1.0.0",
		"Seeds one configured assistant agent per template",
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

			for _, definition := range s.definitions(org.ID, org.BusinessUnitID) {
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
			Kind:           agentdefinition.KindDispatchAssistant,
			Focus: "We run mostly reefer freight out of the Los Angeles terminal. " +
				"When a driver is asked about, check hours of service before anything else.",
			ToolNames:       []string{"get_shipment", "search_shipments", "get_worker", "search_workers"},
			AutonomyCeiling: agent.TierActWithApproval,
			Enabled:         true,
		},
		{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Name:           SeedAgentBillingName,
			Description:    "Works blocked billing queue items and proposes what to do about them.",
			Kind:           agentdefinition.KindBillingAssistant,
			Focus: "Never propose a rate change. If a rate looks wrong, flag it for a person " +
				"and say which document you would want to see.",
			ToolNames:       []string{"get_shipment", "search_shipments", "flag_for_manual_review", "request_missing_docs"},
			AutonomyCeiling: agent.TierPropose,
			Enabled:         true,
		},
		{
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Name:            "Compliance desk",
			Description:     "Answers questions about driver qualification and what is expiring.",
			Kind:            agentdefinition.KindComplianceAssistant,
			Focus:           "Medical cards and hazmat endorsements are the two we most often miss.",
			ToolNames:       []string{"get_worker", "search_workers"},
			AutonomyCeiling: agent.TierPropose,
			Enabled:         true,
		},
		{
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Name:            "Customer service",
			Description:     "Shipment status for the customer-facing team. Off until reviewed.",
			Kind:            agentdefinition.KindCustomerAssistant,
			ToolNames:       []string{"get_shipment", "search_shipments"},
			AutonomyCeiling: agent.TierPropose,
			Enabled:         false,
		},
		{
			OrganizationID:  orgID,
			BusinessUnitID:  buID,
			Name:            "Help",
			Description:     "Explains how to do things in Trenova. Cannot read or change records.",
			Kind:            agentdefinition.KindGeneralAssistant,
			AutonomyCeiling: agent.TierPropose,
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
