package assistantservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsableDefinition_AQuickQuestionHoldsEveryReadButExtensions(t *testing.T) {
	t.Parallel()

	queries := &stubQueryRegistry{Tools: []agentports.AgentQueryTool{
		&agenttest.StubQueryTool{ToolName: "get_shipment"},
		&agenttest.StubQueryTool{ToolName: "get_customer"},
		&agenttest.StubQueryTool{ToolName: agentextension.ToolWebSearch},
	}}
	svc := newService(&scriptedCompletion{}, queries, &stubActionRegistry{})
	svc.queries = queries
	general := chatDefinition("Help", agentdefinition.TemplateGeneralAssistant)
	svc.definitions = &stubDefinitionList{definitions: []*agentdefinition.Definition{general}}
	actor := testActor()
	req := &agentports.SendMessageRequest{TenantInfo: actor.TenantInfo()}

	asked, err := svc.usableDefinition(t.Context(), &conversation.Thread{
		AgentDefinitionID: general.ID,
		Origin:            conversation.ThreadOriginAsk,
	}, actor, req)
	require.NoError(t, err)
	assert.Equal(t, []string{"get_shipment", "get_customer"}, asked.ToolNames)

	desk, err := svc.usableDefinition(t.Context(), &conversation.Thread{
		AgentDefinitionID: general.ID,
		Origin:            conversation.ThreadOriginDesk,
	}, actor, req)
	require.NoError(t, err)
	assert.Empty(t, desk.ToolNames, "the general assistant keeps its own tools in the Desk")
	assert.Empty(t, general.ToolNames, "the stored agent is not changed")
}
