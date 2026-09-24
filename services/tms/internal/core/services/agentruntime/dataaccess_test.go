package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
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
		tool := &agentruntimetest.StubQueryTool{ToolName: "get_invoice", Result: map[string]any{}}
		completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
			toolTurn("get_invoice", map[string]any{"invoiceId": "inv_1"}),
			textTurn("done"),
		}}
		rt := newRuntime(completion, &stubQueryRegistry{
			Tools: []serviceports.AgentQueryTool{tool},
		}, &stubActionRegistry{}, nil)

		definition := testDefinition("get_invoice")
		definition.DataAccessCeiling = setting
		_, err := rt.Run(t.Context(), &serviceports.RunRequest{
			Definition: definition,
			Actor:      testActor(),
			Input:      "what does the invoice say",
		})
		require.NoError(t, err)

		assert.Equal(t, want, tool.LastParams.DataAccessCeiling, string(setting))
	}
}
