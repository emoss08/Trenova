package novaharness_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/novaharness"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The router runs a request on the provider chosen for Nova, with Nova's model,
// only when it carries the Nova mark, so every request the harness sends must
// carry it: one left unmarked would run on whatever providers its task names.

// deferring is a router that keeps the background submissions and polls it
// was asked for.
type deferring struct {
	*agenttest.ScriptedCompletion

	submitted []*llmports.StructuredCompletionRequest
	polled    []*llmports.BackgroundPollRequest
}

func (d *deferring) SubmitBackground(
	_ context.Context,
	req *llmports.StructuredCompletionRequest,
) (*llmports.BackgroundSubmission, error) {
	d.submitted = append(d.submitted, req)

	return &llmports.BackgroundSubmission{Handle: "resp_1"}, nil
}

func (d *deferring) PollBackground(
	_ context.Context,
	req *llmports.BackgroundPollRequest,
) (*llmports.BackgroundOutcome, error) {
	d.polled = append(d.polled, req)

	return &llmports.BackgroundOutcome{State: llmports.BackgroundPending}, nil
}

// batching is a batch service that keeps every request it was asked.
type batching struct {
	submitted *llmports.BatchCompletionRequest
	status    *llmports.BatchStatusRequest
	canceled  *llmports.BatchCancelRequest
	results   *llmports.BatchResultsRequest
}

func (b *batching) SubmitBatch(
	_ context.Context,
	req *llmports.BatchCompletionRequest,
) (*llmports.BatchHandle, error) {
	b.submitted = req

	return &llmports.BatchHandle{BatchID: "msgbatch_1"}, nil
}

func (b *batching) BatchStatus(
	_ context.Context,
	req *llmports.BatchStatusRequest,
) (*llmports.BatchStatus, error) {
	b.status = req

	return &llmports.BatchStatus{}, nil
}

func (b *batching) CancelBatch(_ context.Context, req *llmports.BatchCancelRequest) error {
	b.canceled = req

	return nil
}

func (b *batching) BatchResults(
	_ context.Context,
	req *llmports.BatchResultsRequest,
) (map[string]llmports.BatchItemResult, error) {
	b.results = req

	return map[string]llmports.BatchItemResult{}, nil
}

func TestRun_MarksTheCallItSendsAsNovas(t *testing.T) {
	t.Parallel()

	completion := &structured{text: `{}`}
	harness := novaharness.New(novaharness.Params{Dispatcher: completion, Providers: serves(true)})

	_, err := harness.Run(t.Context(), readTask(), &map[string]any{})
	require.NoError(t, err)

	require.Len(t, completion.requests, 1)
	assert.True(t, completion.requests[0].Nova)
}

func TestRun_MarksEveryCallOfATaskThatReadsAsNovas(t *testing.T) {
	t.Parallel()

	router := &investigating{
		ScriptedCompletion: &agenttest.ScriptedCompletion{
			Turns: []*llmports.ChatCompletionResult{
				{ToolCalls: []llmports.ToolCall{{
					ID: "toolu_1", Name: "get_record", Arguments: map[string]any{"id": "doc_1"},
				}}},
				{ToolCalls: []llmports.ToolCall{{
					ID: "toolu_2", Name: "get_record", Arguments: map[string]any{"id": "doc_2"},
				}}},
			},
		},
		closingText: `{"signedBy":"Lee"}`,
	}
	harness := novaharness.New(novaharness.Params{Dispatcher: router, Router: router, Providers: serves(true)})

	var answer signature
	err := onWorker(func(ctx context.Context) error {
		_, runErr := harness.Run(ctx, readingTask(&lookup{}), &answer)

		return runErr
	})
	require.NoError(t, err)

	require.Len(t, router.Requests, 2, "the reads run out before an answer")
	for idx, sent := range router.Requests {
		assert.True(t, sent.Nova, "read call %d", idx)
	}
	require.Len(t, router.closing, 1, "the closing call answers with what was read")
	assert.True(t, router.closing[0].Nova)
}

func TestConverse_MarksEveryCallAsNovas(t *testing.T) {
	t.Parallel()

	router := &agenttest.ScriptedCompletion{
		Turns: []*llmports.ChatCompletionResult{{Text: "Two loads are late."}},
	}
	harness := novaharness.New(novaharness.Params{Dispatcher: router, Router: router, Providers: serves(true)})

	answer, err := harness.Converse(t.Context(), &llmports.NovaConversation{
		Name:         aitask.TaskPaletteAnswer,
		Instructions: "Your task: answer the question.",
		Question:     "Which loads are late?",
		TenantInfo:   tenant(),
		UserID:       pulid.MustNew("usr_"),
	}, &heard{})
	require.NoError(t, err)

	assert.Equal(t, "Two loads are late.", answer.Text)
	require.Len(t, router.Requests, 1)
	assert.True(t, router.Requests[0].Nova)
}

func TestConverse_RefusesWhileNoProviderIsChosenForNova(t *testing.T) {
	t.Parallel()

	router := &agenttest.ScriptedCompletion{
		Turns: []*llmports.ChatCompletionResult{{Text: "unused"}},
	}
	harness := novaharness.New(novaharness.Params{Dispatcher: router, Router: router, Providers: serves(false)})

	_, err := harness.Converse(t.Context(), &llmports.NovaConversation{
		Name:       aitask.TaskPaletteAnswer,
		Question:   "Which loads are late?",
		TenantInfo: tenant(),
	}, &heard{})

	require.ErrorIs(t, err, llmports.ErrNoProviderConfigured)
	assert.Empty(t, router.Requests)
}

func TestDeferAndCollect_MarkTheSubmissionAndItsPollAsNovas(t *testing.T) {
	t.Parallel()

	router := &deferring{ScriptedCompletion: &agenttest.ScriptedCompletion{}}
	harness := novaharness.New(novaharness.Params{Dispatcher: router, Router: router, Providers: serves(true)})

	_, err := harness.Defer(t.Context(), readTask())
	require.NoError(t, err)
	poll := &llmports.BackgroundPollRequest{
		TenantInfo: tenant(),
		ProviderID: pulid.MustNew("aiprv_"),
		Handle:     "resp_1",
		Task:       aitask.TaskDocumentExtraction,
	}
	_, err = harness.Collect(t.Context(), poll)
	require.NoError(t, err)

	require.Len(t, router.submitted, 1)
	assert.True(t, router.submitted[0].Nova)
	require.Len(t, router.polled, 1)
	assert.True(t, router.polled[0].Nova)
	assert.Equal(t, "resp_1", router.polled[0].Handle)
	assert.False(t, poll.Nova, "the caller's request is not changed")
}

func TestBatches_MarkEveryRequestAsNovas(t *testing.T) {
	t.Parallel()

	service := &batching{}
	harness := novaharness.New(novaharness.Params{Batches: service})
	task := readTask()

	_, err := harness.SubmitTasks(t.Context(), &llmports.NovaBatch{
		TenantInfo: task.TenantInfo,
		Task:       aitask.TaskBillingDiagnosis,
		Items:      []llmports.NovaBatchItem{{CustomID: "a", Task: task}},
	})
	require.NoError(t, err)
	status := &llmports.BatchStatusRequest{BatchID: "msgbatch_1"}
	_, err = harness.BatchStatus(t.Context(), status)
	require.NoError(t, err)
	cancel := &llmports.BatchCancelRequest{BatchID: "msgbatch_1"}
	require.NoError(t, harness.CancelBatch(t.Context(), cancel))
	results := &llmports.BatchResultsRequest{BatchID: "msgbatch_1"}
	_, err = harness.BatchResults(t.Context(), results)
	require.NoError(t, err)

	assert.True(t, service.submitted.Nova)
	require.Len(t, service.submitted.Items, 1)
	assert.True(t, service.submitted.Items[0].Request.Nova)
	assert.True(t, service.status.Nova)
	assert.Equal(t, "msgbatch_1", service.status.BatchID)
	assert.True(t, service.canceled.Nova)
	assert.True(t, service.results.Nova)
	assert.False(
		t,
		status.Nova || cancel.Nova || results.Nova,
		"the caller's requests are not changed",
	)
}
