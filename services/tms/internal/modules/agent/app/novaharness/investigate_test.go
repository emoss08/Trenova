package novaharness_test

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/novaharness"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

// onWorker runs fn inside a Temporal activity, the only place the harness reads.
func onWorker(fn func(context.Context) error) error {
	var suite testsuite.WorkflowTestSuite
	worker := suite.NewTestActivityEnvironment()
	worker.RegisterActivityWithOptions(fn, activity.RegisterOptions{Name: "investigate"})
	_, err := worker.ExecuteActivity("investigate")

	return err
}

type investigating struct {
	*agenttest.ScriptedCompletion

	closingText string
	closing     []*llmports.StructuredCompletionRequest
}

func (i *investigating) CompleteStructured(
	_ context.Context,
	req *llmports.StructuredCompletionRequest,
) (*llmports.StructuredCompletionResult, error) {
	i.closing = append(i.closing, req)

	return &llmports.StructuredCompletionResult{Text: i.closingText, ModelIdentifier: "haiku"}, nil
}

func readingTask(reads ...llmports.NovaTool) *llmports.NovaTask {
	task := readTask()
	task.Context = llmports.DelimitedContext{Sections: []llmports.ContextSection{
		{Title: "The invoice", Trusted: true, Content: "Fuel $908.14"},
	}}
	task.Reads = reads
	task.MaxCalls = 3

	return task
}

func answering(arguments map[string]any) *llmports.ChatCompletionResult {
	return &llmports.ChatCompletionResult{
		ModelIdentifier: "sonnet",
		ToolCalls:       []llmports.ToolCall{{ID: "toolu_9", Name: "answer", Arguments: arguments}},
	}
}

type signature struct {
	SignedBy string `json:"signedBy"`
}

func TestRun_AnswersThroughTheAnswerToolInOneCallWhenNothingNeedsReading(t *testing.T) {
	t.Parallel()

	router := &investigating{ScriptedCompletion: &agenttest.ScriptedCompletion{
		Turns: []*llmports.ChatCompletionResult{answering(map[string]any{"signedBy": "Dana"})},
	}}
	harness := novaharness.New(novaharness.Params{Dispatcher: router, Router: router, Providers: serves(true)})
	read := &lookup{}

	var answer signature
	var result *llmports.StructuredCompletionResult
	err := onWorker(func(ctx context.Context) error {
		ran, runErr := harness.Run(ctx, readingTask(read), &answer)
		result = ran

		return runErr
	})
	require.NoError(t, err)

	assert.Equal(t, "Dana", answer.SignedBy)
	assert.Equal(t, "sonnet", result.ModelIdentifier)
	assert.JSONEq(t, `{"signedBy":"Dana"}`, result.Text)
	require.Len(t, router.Requests, 1)
	assert.Empty(t, router.closing)
	assert.Empty(t, read.calls)

	sent := router.Requests[0]
	require.Len(t, sent.Tools, 2)
	assert.Equal(t, "get_record", sent.Tools[0].Name)
	assert.Equal(t, "answer", sent.Tools[1].Name)
	assert.Equal(t, map[string]any{"type": "object"}, sent.Tools[1].Parameters, "the answer takes the task's schema")
	assert.True(t, strings.HasPrefix(sent.System, "You are Nova"))
	assert.Contains(t, sent.System, "Your task: read the signature.")
	assert.Contains(t, sent.System, "calling the answer tool once")
	require.Len(t, sent.Messages, 1)
	assert.Contains(t, sent.Messages[0].Content, "Fuel $908.14")
}

func TestRun_ReadsThenAnswersWithTheReadInFrontOfTheModel(t *testing.T) {
	t.Parallel()

	router := &investigating{ScriptedCompletion: &agenttest.ScriptedCompletion{
		Turns: []*llmports.ChatCompletionResult{
			{ToolCalls: []llmports.ToolCall{{
				ID: "toolu_1", Name: "get_record", Arguments: map[string]any{"id": "doc_1"},
			}}},
			answering(map[string]any{"signedBy": "Lee"}),
		},
	}}
	harness := novaharness.New(novaharness.Params{Dispatcher: router, Router: router, Providers: serves(true)})
	read := &lookup{result: llmports.NovaToolResult{Content: "signed by Lee"}}

	var answer signature
	require.NoError(t, onWorker(func(ctx context.Context) error {
		_, err := harness.Run(ctx, readingTask(read), &answer)

		return err
	}))

	assert.Equal(t, "Lee", answer.SignedBy)
	assert.Equal(t, []map[string]any{{"id": "doc_1"}}, read.calls)
	require.Len(t, router.Requests, 2)
	second := router.Requests[1].Messages
	require.Len(t, second, 3, "the material, the read and its result")
	assert.Equal(t, llmports.RoleTool, second[2].Role)
	assert.Equal(t, "signed by Lee", second[2].Content)
	assert.Empty(t, router.closing)
}

func TestRun_AnswersWithWhatWasReadOnceTheCallsRunOut(t *testing.T) {
	t.Parallel()

	reading := &llmports.ChatCompletionResult{ToolCalls: []llmports.ToolCall{{
		ID: "toolu_1", Name: "get_record", Arguments: map[string]any{"id": "doc_1"},
	}}}
	router := &investigating{
		ScriptedCompletion: &agenttest.ScriptedCompletion{
			Turns: []*llmports.ChatCompletionResult{reading},
		},
		closingText: `{"signedBy":"Kim"}`,
	}
	harness := novaharness.New(novaharness.Params{Dispatcher: router, Router: router, Providers: serves(true)})
	read := &lookup{result: llmports.NovaToolResult{Content: "signed by Kim"}}
	task := readingTask(read)

	var answer signature
	err := onWorker(func(ctx context.Context) error {
		_, runErr := harness.Run(ctx, task, &answer)

		return runErr
	})
	require.NoError(t, err)

	assert.Equal(t, "Kim", answer.SignedBy)
	assert.Len(t, router.Requests, 2, "the last of three calls is the structured answer")
	require.Len(t, router.closing, 1)
	sections := router.closing[0].Context.Sections
	require.Len(t, sections, 3)
	assert.Equal(t, "The invoice", sections[0].Title)
	assert.Equal(t, `What you read with get_record {"id":"doc_1"}`, sections[1].Title)
	assert.Equal(t, "signed by Kim", sections[1].Content)
	assert.False(t, sections[1].Trusted, "what a read returned is data")
	assert.Len(t, task.Context.Sections, 1, "the task's own context is left as it was")
}

func TestRun_AnswersInOneStructuredCallWhenTheToolAnswerDoesNotDecode(t *testing.T) {
	t.Parallel()

	router := &investigating{
		ScriptedCompletion: &agenttest.ScriptedCompletion{
			Turns: []*llmports.ChatCompletionResult{answering(map[string]any{"signedBy": 7})},
		},
		closingText: `{"signedBy":"Dana"}`,
	}
	harness := novaharness.New(novaharness.Params{Dispatcher: router, Router: router, Providers: serves(true)})

	var answer signature
	err := onWorker(func(ctx context.Context) error {
		_, runErr := harness.Run(ctx, readingTask(&lookup{}), &answer)

		return runErr
	})
	require.NoError(t, err)

	assert.Equal(t, "Dana", answer.SignedBy)
	assert.Len(t, router.Requests, 1)
	assert.Len(t, router.closing, 1)
}

func TestRun_AnswersFromItsContextAloneOnARequestPath(t *testing.T) {
	t.Parallel()

	dispatcher := &structured{text: `{"signedBy":"Dana"}`}
	router := &investigating{ScriptedCompletion: &agenttest.ScriptedCompletion{}}
	harness := novaharness.New(novaharness.Params{
		Dispatcher: dispatcher,
		Router:     router,
		Providers:  serves(true),
	})

	var answer signature
	_, err := harness.Run(t.Context(), readingTask(&lookup{}), &answer)
	require.NoError(t, err)

	assert.Equal(t, "Dana", answer.SignedBy)
	assert.Len(t, dispatcher.requests, 1)
	assert.Empty(t, router.Requests)
}
