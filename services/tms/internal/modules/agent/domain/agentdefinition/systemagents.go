package agentdefinition

import (
	"slices"

	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	SystemKeyImportAssistant  = "import_assistant"
	SystemKeyFormulaAssistant = "formula_assistant"
)

type pageAgent struct {
	key      string
	template Template
}

var pageAgents = [...]pageAgent{
	{key: SystemKeyImportAssistant, template: TemplateImportAssistant},
	{key: SystemKeyFormulaAssistant, template: TemplateFormulaAssistant},
}

func PageAgentKeys() []string {
	keys := make([]string, 0, len(pageAgents))
	for _, agent := range pageAgents {
		keys = append(keys, agent.key)
	}

	return keys
}

func IsPageAgentKey(key string) bool {
	return slices.Contains(PageAgentKeys(), key)
}

func PageAgentTemplate(key string) (Template, bool) {
	for _, agent := range pageAgents {
		if agent.key == key {
			return agent.template, true
		}
	}

	return "", false
}

func (d *Definition) IsPageAgent() bool {
	return IsPageAgentKey(d.SystemKey)
}

func NewPageAgent(key string, tenant pagination.TenantInfo, name string) (*Definition, bool) {
	template, ok := PageAgentTemplate(key)
	if !ok {
		return nil, false
	}

	definition := &Definition{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		Name:              name,
		Description:       template.Description(),
		Template:          template,
		Icon:              templateIcons[template],
		Instructions:      template.StarterInstructions(),
		ToolNames:         template.StarterTools(),
		AutonomyCeiling:   template.StarterCeiling(),
		DataAccessCeiling: template.StarterDataAccess(),
		TriggerMode:       TriggerChat,
		OutputMode:        OutputConversational,
		SystemKey:         key,
		AccessMode:        AccessEveryone,
		ContextProviders:  AllContextProviders(),
		Enabled:           true,
	}
	definition.ApplyDefaults()

	return definition, true
}
