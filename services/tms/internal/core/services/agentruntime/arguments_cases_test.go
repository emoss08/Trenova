package agentruntime

import (
	"maps"
	"os"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/pkg/errortypes"
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
	// Refused lists the paths the refusal must name; a case with any is
	// one the tool never receives.
	Refused []string `yaml:"refused"`
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

func toolMessages(result *serviceports.RunResult) []conversation.Message {
	var out []conversation.Message
	for idx := range result.Messages {
		if result.Messages[idx].Role == conversation.RoleTool {
			out = append(out, result.Messages[idx])
		}
	}

	return out
}

// requireRefusalNames asserts a refusal carries one "path: message" line
// for each path, and tells the model to fix the call.
func requireRefusalNames(t *testing.T, content string, paths []string) {
	t.Helper()

	assert.Contains(t, content, "do not fit the tool")
	assert.Contains(t, content, "Fix the call and send it again, changing only what is named.")
	for _, path := range paths {
		assert.Contains(t, content, "\n- "+path+": ", "the refusal names %s", path)
	}
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

func TestArgumentCases_WhatAToolReceives(t *testing.T) {
	t.Parallel()

	for _, tc := range loadArgumentCases(t).Pipeline {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			sent := maps.Clone(tc.Args)
			got, _, err := contractArguments(nil, "record_case", tc.Schema, sent)
			assert.Equal(t, tc.Args, sent, "the model's own call is never changed")
			if len(tc.Refused) > 0 {
				var multiErr *errortypes.MultiError
				require.ErrorAs(t, err, &multiErr)
				problems := strings.Join(argumentProblems(multiErr), "\n")
				for _, path := range tc.Refused {
					assert.Contains(t, problems, path+": ", "the refusal names %s", path)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.Want, got)
			}

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
			assert.Equal(t, 1, result.ToolCallsUsed, "a call spends the budget either way")

			if len(tc.Refused) > 0 {
				assert.Zero(t, tool.Calls, "a call that does not fit never reaches the tool")
				assert.Empty(t, result.Actions)
				refusals := toolMessages(result)
				require.Len(t, refusals, 1)
				assert.True(t, refusals[0].ToolFailed)
				requireRefusalNames(t, refusals[0].Content, tc.Refused)

				return
			}

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

			refusals := toolMessages(result)
			require.Len(t, refusals, 1)
			assert.True(t, refusals[0].ToolFailed)
			assert.Contains(t, refusals[0].Content, "not valid JSON")
			assert.Contains(t, refusals[0].Content, tc.Error)
		})
	}
}
