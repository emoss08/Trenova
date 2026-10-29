package assistanttest

import (
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

type (
	ScriptedCompletion = agenttest.ScriptedCompletion
	StubQueryRegistry  = agenttest.StubQueryRegistry
	StubActionRegistry = agenttest.StubActionRegistry
)

func Actor() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func Definition(tools ...string) *agentdefinition.Definition {
	d := &agentdefinition.Definition{
		Name:            "Dispatch helper",
		Template:        agenttemplate.TemplateDispatchAssistant,
		Instructions:    "Help the dispatch desk.",
		AutonomyCeiling: agent.TierPropose,
		ToolNames:       tools,
		Enabled:         true,
	}
	d.ApplyDefaults()

	return d
}

func TextTurn(text string) *llmports.ChatCompletionResult {
	return &llmports.ChatCompletionResult{Text: text, ModelIdentifier: "test-model"}
}
