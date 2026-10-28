package agentpreset

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentinstructions"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/pkg/pagination"
)

func NewPageAgent(
	key string,
	tenant pagination.TenantInfo,
	name string,
) (*agentdefinition.Definition, bool) {
	template, ok := agentdefinition.PageAgentTemplate(key)
	if !ok {
		return nil, false
	}

	definition := &agentdefinition.Definition{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		Name:              name,
		Description:       agenttemplate.Description(template),
		Template:          template,
		Icon:              agentdefinition.TemplateIcon(template),
		Instructions:      agentinstructions.StarterInstructions(template),
		ToolNames:         StarterTools(template),
		AutonomyCeiling:   StarterCeiling(template),
		DataAccessCeiling: StarterDataAccess(template),
		TriggerMode:       agentdefinition.TriggerChat,
		OutputMode:        agentdefinition.OutputConversational,
		SystemKey:         key,
		AccessMode:        agentdefinition.AccessEveryone,
		ContextProviders:  agentdefinition.AllContextProviders(),
		Enabled:           true,
	}
	definition.ApplyDefaults()

	return definition, true
}
