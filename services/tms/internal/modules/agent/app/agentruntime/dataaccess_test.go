package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_HandsQueryToolsTheAgentsDataAccess(t *testing.T) {
	t.Parallel()

	cases := map[agentdefinition.DataAccessCeiling]permission.FieldSensitivity{
		"":                                   permission.SensitivityInternal,
		agentdefinition.DataAccessInternal:   permission.SensitivityInternal,
		agentdefinition.DataAccessRestricted: permission.SensitivityRestricted,
	}
	for setting, want := range cases {
		tool := &agenttest.StubQueryTool{ToolName: "invoice_get", Result: map[string]any{}}
		completion := &scriptedCompletion{Turns: []*agentports.ChatCompletionResult{
			toolTurn("invoice_get", map[string]any{"invoiceId": "inv_1"}),
			textTurn("done"),
		}}
		rt := newRuntime(completion, &stubQueryRegistry{
			Tools: []agentports.AgentQueryTool{tool},
		}, &stubActionRegistry{}, nil)

		definition := testDefinition("invoice_get")
		definition.DataAccessCeiling = setting
		_, err := rt.Run(t.Context(), &agentports.RunRequest{
			Definition: definition,
			Actor:      testActor(),
			Input:      "what does the invoice say",
		})
		require.NoError(t, err)

		assert.Equal(t, want, tool.LastParams.DataAccessCeiling, string(setting))
	}
}
