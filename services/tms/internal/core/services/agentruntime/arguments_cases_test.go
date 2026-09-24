package agentruntime

import (
	"maps"
	"os"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const argumentCasesPath = "testdata/arguments/cases.yaml"

type argumentCase struct {
	Name   string         `yaml:"name"`
	Schema map[string]any `yaml:"schema"`
	Args   map[string]any `yaml:"args"`
	Want   map[string]any `yaml:"want"`
}

type malformedCase struct {
	Name  string `yaml:"name"`
	Error string `yaml:"error"`
}

type argumentCases struct {
	Aliases   []argumentCase  `yaml:"aliases"`
	Pipeline  []argumentCase  `yaml:"pipeline"`
	Malformed []malformedCase `yaml:"malformed"`
}

func loadArgumentCases(t *testing.T) argumentCases {
	t.Helper()

	raw, err := os.ReadFile(argumentCasesPath)
	require.NoError(t, err)

	var cases argumentCases
	require.NoError(t, yaml.Unmarshal(raw, &cases))
	require.NotEmpty(t, cases.Aliases)
	require.NotEmpty(t, cases.Pipeline)
	require.NotEmpty(t, cases.Malformed)

	return cases
}

func TestArgumentCases_Aliases(t *testing.T) {
	t.Parallel()

	for _, tc := range loadArgumentCases(t).Aliases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			sent := maps.Clone(tc.Args)
			got := aliasedArguments(tc.Schema, sent)

			assert.Equal(t, tc.Want, got)
			assert.Equal(t, tc.Args, sent, "the model's own call is never changed")
		})
	}
}

func TestArgumentCases_WhatAWriteReceives(t *testing.T) {
	t.Parallel()

	for _, tc := range loadArgumentCases(t).Pipeline {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.Want,
				declaredArguments(tc.Schema, aliasedArguments(tc.Schema, maps.Clone(tc.Args))))

			tool := &agentruntimetest.StubActionTool{
				ToolName: "record_case",
				Tier:     agent.TierAutoExecute,
				Schema:   tc.Schema,
			}
			completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
				toolTurn(tool.ToolName, maps.Clone(tc.Args)),
				textTurn("Done."),
			}}
			rt := newRuntime(completion, &stubQueryRegistry{},
				&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)

			result, err := rt.Run(t.Context(), &serviceports.RunRequest{
				Definition: autoDefinition(tool.ToolName),
				Actor:      testActor(),
				Input:      "Record it.",
			})
			require.NoError(t, err)

			require.Equal(t, 1, tool.Calls, "the write ran")
			assert.Equal(t, tc.Want, tool.LastParams.Params)
			require.Len(t, result.Actions, 1)
			assert.Equal(t, tc.Want, result.Actions[0].Arguments,
				"the recorded action carries what the tool took")
		})
	}
}

func TestArgumentCases_MalformedArgumentsNeverRun(t *testing.T) {
	t.Parallel()

	for _, tc := range loadArgumentCases(t).Malformed {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			tool := &agentruntimetest.StubActionTool{
				ToolName: "record_case",
				Tier:     agent.TierAutoExecute,
			}
			completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
				{
					ToolCalls: []serviceports.ToolCall{{
						ID:             "call_malformed",
						Name:           tool.ToolName,
						Arguments:      map[string]any{},
						ArgumentsError: tc.Error,
					}},
					ModelIdentifier: "test-model",
				},
				textTurn("I could not send it."),
			}}
			rt := newRuntime(completion, &stubQueryRegistry{},
				&stubActionRegistry{Tools: []serviceports.AgentTool{tool}}, nil)

			result, err := rt.Run(t.Context(), &serviceports.RunRequest{
				Definition: autoDefinition(tool.ToolName),
				Actor:      testActor(),
				Input:      "Record it.",
			})
			require.NoError(t, err)

			assert.Zero(t, tool.Calls, "arguments that did not parse are not arguments")
			assert.Empty(t, result.Actions)
			assert.Equal(t, 1, result.ToolCallsUsed, "a broken call still spends the budget")

			var refusal *conversation.Message
			for idx := range result.Messages {
				if result.Messages[idx].Role == conversation.RoleTool {
					refusal = &result.Messages[idx]
				}
			}
			require.NotNil(t, refusal)
			assert.True(t, refusal.ToolFailed)
			assert.Contains(t, refusal.Content, "not valid JSON")
			assert.Contains(t, refusal.Content, tc.Error)
		})
	}
}
