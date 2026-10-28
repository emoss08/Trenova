package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsableDefinition_AQuickQuestionHoldsEveryReadButExtensions(t *testing.T) {
	t.Parallel()

	queries := &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
		&agenttest.StubQueryTool{ToolName: "shipment_get"},
		&agenttest.StubQueryTool{ToolName: "customer_get"},
		&agenttest.StubQueryTool{ToolName: agentextension.ToolWebSearch},
	}}
	svc := newService(&scriptedCompletion{}, queries, &stubActionRegistry{})
	svc.queries = queries
	general := chatDefinition("Help", agentdefinition.TemplateGeneralAssistant)
	svc.definitions = &stubDefinitionList{definitions: []*agentdefinition.Definition{general}}
	actor := testActor()
	req := &assistantports.SendMessageRequest{TenantInfo: actor.TenantInfo()}

	asked, err := svc.usableDefinition(t.Context(), &conversation.Thread{
		AgentDefinitionID: general.ID,
		Origin:            conversation.ThreadOriginAsk,
	}, actor, req)
	require.NoError(t, err)
	assert.Equal(t, []string{"shipment_get", "customer_get"}, asked.ToolNames)

	desk, err := svc.usableDefinition(t.Context(), &conversation.Thread{
		AgentDefinitionID: general.ID,
		Origin:            conversation.ThreadOriginDesk,
	}, actor, req)
	require.NoError(t, err)
	assert.Empty(t, desk.ToolNames, "the general assistant keeps its own tools in the Desk")
	assert.Empty(t, general.ToolNames, "the stored agent is not changed")
}
