package agentdefinition

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
)

const (
	SystemKeyImportAssistant  = "import_assistant"
	SystemKeyFormulaAssistant = "formula_assistant"
)

type pageAgent struct {
	key      string
	template agenttemplate.Template
}

var pageAgents = [...]pageAgent{
	{key: SystemKeyImportAssistant, template: agenttemplate.TemplateImportAssistant},
	{key: SystemKeyFormulaAssistant, template: agenttemplate.TemplateFormulaAssistant},
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

func PageAgentTemplate(key string) (agenttemplate.Template, bool) {
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
