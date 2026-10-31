package novaharness_test

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/novaharness"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type structured struct {
	text     string
	requests []*llmports.StructuredCompletionRequest
}

func (s *structured) CompleteStructured(
	_ context.Context,
	req *llmports.StructuredCompletionRequest,
) (*llmports.StructuredCompletionResult, error) {
	s.requests = append(s.requests, req)

	return &llmports.StructuredCompletionResult{Text: s.text, ModelIdentifier: "haiku"}, nil
}

type serves bool

func (s serves) CanServe(context.Context, aitask.Task, pagination.TenantInfo) (bool, error) {
	return bool(s), nil
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func readTask() *llmports.NovaTask {
	return &llmports.NovaTask{
		Name:         aitask.TaskPaperworkReading,
		Feature:      aiusage.FeaturePaperworkReading,
		Instructions: "Your task: read the signature.",
		SchemaName:   "signature",
		Schema:       map[string]any{"type": "object"},
		TenantInfo:   tenant(),
		UserID:       pulid.MustNew("usr_"),
	}
}

func TestRun_PutsNovasIdentityBeforeTheTasksInstructions(t *testing.T) {
	t.Parallel()

	completion := &structured{text: `{"signedBy":"Dana"}`}
	harness := novaharness.New(novaharness.Params{Dispatcher: completion, Providers: serves(true)})
	task := readTask()

	var answer struct {
		SignedBy string `json:"signedBy"`
	}
	result, err := harness.Run(t.Context(), task, &answer)
	require.NoError(t, err)

	assert.Equal(t, "Dana", answer.SignedBy)
	assert.Equal(t, "haiku", result.ModelIdentifier)
	require.Len(t, completion.requests, 1)
	sent := completion.requests[0]
	assert.True(t, strings.HasPrefix(sent.System, "You are Nova"), sent.System)
	assert.True(t, strings.HasSuffix(sent.System, "\n\nYour task: read the signature."))
	assert.Equal(t, aitask.TaskPaperworkReading, sent.Task)
	assert.Equal(t, aiusageports.AIUsageAttribution{
		UserID:  task.UserID,
		Feature: aiusage.FeaturePaperworkReading,
	}, sent.Attribution)
}

func TestRun_BillsTheAttributionATaskNames(t *testing.T) {
	t.Parallel()

	completion := &structured{text: `{}`}
	harness := novaharness.New(novaharness.Params{Dispatcher: completion, Providers: serves(true)})
	task := readTask()
	task.Attribution = &aiusageports.AIUsageAttribution{
		Feature: aiusage.FeatureFactoringRemittanceExtract,
		Subject: aiusage.Subject{Type: aiusage.SubjectTypeDocument, ID: "doc_1"},
	}

	_, err := harness.Run(t.Context(), task, &map[string]any{})
	require.NoError(t, err)

	assert.Equal(t, *task.Attribution, completion.requests[0].Attribution)
}

func TestRun_ReturnsTheResultWithErrOutsideSchemaWhenTheAnswerIsNotItsShape(t *testing.T) {
	t.Parallel()

	harness := novaharness.New(novaharness.Params{
		Dispatcher: &structured{text: "not json"},
		Providers:  serves(true),
	})

	var answer map[string]any
	result, err := harness.Run(t.Context(), readTask(), &answer)

	require.ErrorIs(t, err, llmports.ErrOutsideSchema)
	require.NotNil(t, result, "the call is still returned so its cost can be recorded")
	assert.Equal(t, "not json", result.Text)
}

func TestRun_RefusesATaskNoProviderServes(t *testing.T) {
	t.Parallel()

	completion := &structured{text: `{}`}
	harness := novaharness.New(novaharness.Params{Dispatcher: completion, Providers: serves(false)})

	_, err := harness.Run(t.Context(), readTask(), &map[string]any{})

	require.ErrorIs(t, err, llmports.ErrNoProviderConfigured)
	assert.Empty(t, completion.requests)
}

func TestRun_UsesTheRouterWhenNoDispatcherIsWired(t *testing.T) {
	t.Parallel()

	router := &agenttest.ScriptedCompletion{StructuredText: `{"ok":true}`}
	harness := novaharness.New(novaharness.Params{Router: router, Providers: serves(true)})

	var answer struct {
		OK bool `json:"ok"`
	}
	_, err := harness.Run(t.Context(), readTask(), &answer)
	require.NoError(t, err)
	assert.True(t, answer.OK)
}

type batches struct {
	llmports.BatchCompletionService

	submitted *llmports.BatchCompletionRequest
}

func (b *batches) SubmitBatch(
	_ context.Context,
	req *llmports.BatchCompletionRequest,
) (*llmports.BatchHandle, error) {
	b.submitted = req

	return &llmports.BatchHandle{BatchID: "msgbatch_1"}, nil
}

func TestSubmitTasks_SendsEveryTaskWithNovasIdentity(t *testing.T) {
	t.Parallel()

	service := &batches{}
	harness := novaharness.New(novaharness.Params{Batches: service})
	first, second := readTask(), readTask()
	second.Instructions = "Your task: read the seal."

	handle, err := harness.SubmitTasks(t.Context(), &llmports.NovaBatch{
		TenantInfo: first.TenantInfo,
		Task:       aitask.TaskPaperworkReading,
		Items: []llmports.NovaBatchItem{
			{CustomID: "a", Task: first},
			{CustomID: "b", Task: second},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "msgbatch_1", handle.BatchID)
	require.Len(t, service.submitted.Items, 2)
	assert.Equal(t, "b", service.submitted.Items[1].CustomID)
	assert.True(t, strings.HasPrefix(service.submitted.Items[1].Request.System, "You are Nova"))
	assert.True(t, strings.HasSuffix(service.submitted.Items[1].Request.System, "Your task: read the seal."))
}

func TestSubmitTasks_IsUnsupportedWithoutABatchService(t *testing.T) {
	t.Parallel()

	_, err := novaharness.New(novaharness.Params{}).SubmitTasks(t.Context(), &llmports.NovaBatch{})

	require.ErrorIs(t, err, llmports.ErrBatchUnsupported)
}

type lookup struct {
	result llmports.NovaToolResult
	calls  []map[string]any
}

func (l *lookup) Spec() kernelports.ToolSpec {
	return kernelports.ToolSpec{Name: "get_record", Parameters: map[string]any{"type": "object"}}
}

func (l *lookup) Call(_ context.Context, arguments map[string]any) llmports.NovaToolResult {
	l.calls = append(l.calls, arguments)

	return l.result
}

type heard struct {
	deltas   []string
	messages []string
	started  []string
	finished []llmports.NovaToolResult
}

func (h *heard) Delta(text string) { h.deltas = append(h.deltas, text) }

func (h *heard) Message(content string, _ []llmports.ToolCall, _ string) {
	h.messages = append(h.messages, content)
}

func (h *heard) ToolStarted(call llmports.ToolCall) { h.started = append(h.started, call.ID) }

func (h *heard) ToolFinished(_ llmports.ToolCall, result llmports.NovaToolResult) {
	h.finished = append(h.finished, result)
}

func paletteTalk(tools ...llmports.NovaTool) *llmports.NovaConversation {
	return &llmports.NovaConversation{
		Name:         aitask.TaskPaletteAnswer,
		Feature:      aiusage.FeaturePaletteAnswer,
		Instructions: "Your task: answer a quick question.",
		Question:     "Who is on S-104?",
		Tools:        tools,
		MaxCalls:     3,
		TenantInfo:   tenant(),
		UserID:       pulid.MustNew("usr_"),
	}
}

func TestConverse_ReadsThenAnswersStreamingEachStep(t *testing.T) {
	t.Parallel()

	shipment := agentpagecontext.EntityRef{Type: "shipment", ID: "shp_1", Label: "S-104"}
	tool := &lookup{result: llmports.NovaToolResult{
		Content: "driver: Dana",
		Records: []agentpagecontext.EntityRef{shipment, shipment},
	}}
	router := &agenttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		{
			Text: "Looking it up.",
			ToolCalls: []llmports.ToolCall{{
				ID: "call_0", SynthesizedID: true, Name: "get_record",
				Arguments: map[string]any{"id": "shp_1"},
			}},
			ModelIdentifier: "haiku",
		},
		{Text: "Dana is driving S-104.", ModelIdentifier: "haiku"},
	}}
	harness := novaharness.New(novaharness.Params{Router: router, Providers: serves(true)})
	events := &heard{}

	answer, err := harness.Converse(t.Context(), paletteTalk(tool), events)
	require.NoError(t, err)

	assert.Equal(t, "Dana is driving S-104.", answer.Text)
	assert.Equal(t, "haiku", answer.Model)
	assert.Equal(t, []agentpagecontext.EntityRef{shipment}, answer.Records, "each record once")
	assert.Equal(t, []map[string]any{{"id": "shp_1"}}, tool.calls)
	assert.Equal(t, []string{"Looking it up."}, events.messages)
	require.Len(t, events.started, 1)
	assert.NotEqual(t, "call_0", events.started[0], "a synthesized call id is replaced")
	assert.Equal(t, "Dana is driving S-104.", strings.Join(events.deltas[len(events.deltas)-2:], ""))

	require.Len(t, router.Requests, 2)
	first, second := router.Requests[0], router.Requests[1]
	assert.Equal(t, aitask.TaskPaletteAnswer, first.Task)
	assert.Equal(t, aiusage.FeaturePaletteAnswer, first.Attribution.Feature)
	assert.True(t, strings.HasPrefix(first.System, "You are Nova"))
	assert.False(t, first.ToolChoiceNone)
	require.Len(t, second.Messages, 3, "the question, the call and its result")
	assert.Equal(t, llmports.RoleTool, second.Messages[2].Role)
	assert.Equal(t, "driver: Dana", second.Messages[2].Content)
}

func TestConverse_TellsTheLastCallToAnswerWithWhatItHas(t *testing.T) {
	t.Parallel()

	reading := &llmports.ChatCompletionResult{
		Text:      "still reading",
		ToolCalls: []llmports.ToolCall{{ID: "toolu_1", Name: "get_record"}},
	}
	router := &agenttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		reading,
		{Text: "", ToolCalls: []llmports.ToolCall{{ID: "toolu_2", Name: "get_record"}}},
		{Text: "Here is what I found."},
	}}
	harness := novaharness.New(novaharness.Params{Router: router, Providers: serves(true)})

	answer, err := harness.Converse(t.Context(), paletteTalk(&lookup{}), &heard{})
	require.NoError(t, err)

	require.Len(t, router.Requests, 3)
	assert.False(t, router.Requests[1].ToolChoiceNone)
	assert.True(t, router.Requests[2].ToolChoiceNone)
	assert.Equal(t, "Here is what I found.", answer.Text)
}

func TestConverse_RefusesAToolTheTaskWasNotGiven(t *testing.T) {
	t.Parallel()

	router := &agenttest.ScriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		{ToolCalls: []llmports.ToolCall{{ID: "toolu_1", Name: "shipment_cancel"}}},
		{Text: "I can't do that from here."},
	}}
	harness := novaharness.New(novaharness.Params{Router: router, Providers: serves(true)})
	events := &heard{}

	answer, err := harness.Converse(t.Context(), paletteTalk(&lookup{}), events)
	require.NoError(t, err)

	require.Len(t, events.finished, 1)
	assert.True(t, events.finished[0].Failed)
	assert.Contains(t, events.finished[0].Content, "There is no tool named shipment_cancel")
	assert.Equal(t, "I can't do that from here.", answer.Text)
}

func TestConverse_NeedsTheRouterAndAProvider(t *testing.T) {
	t.Parallel()

	_, err := novaharness.New(novaharness.Params{Dispatcher: &structured{}, Providers: serves(true)}).
		Converse(t.Context(), paletteTalk(), &heard{})
	require.ErrorIs(t, err, llmports.ErrNoProviderConfigured)

	_, err = novaharness.New(novaharness.Params{
		Router:    &agenttest.ScriptedCompletion{},
		Providers: serves(false),
	}).Converse(t.Context(), paletteTalk(), &heard{})
	require.ErrorIs(t, err, llmports.ErrNoProviderConfigured)
}
