package novaharness

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Dispatcher llmports.StructuredCompleter
	Router     llmports.CompletionService
	Batches    llmports.BatchCompletionService
	// Providers says whether the tenant has a provider chosen for Nova. The
	// completion router provides it in every binary, so it is required: a
	// missing one is a wiring mistake, not a Nova that is off.
	Providers llmports.ProviderAvailability
}

// Harness runs Nova's background tasks: one structured call with Nova's
// identity, the task's instructions and its files, and no conversation or chat
// guards around it. Briefings, reviews and document reading run here; the Desk
// keeps its own runtime.
//
// Every call the harness makes is marked Nova, so the router runs it on the
// provider the organization chose for Nova with Nova's own model
// (novamodel.ID), whatever providers its task names. The task names only what
// the usage was for.
//
// A task asked for on a request path runs as a worker job through the
// dispatcher. A task asked for inside an activity is already on a worker and
// calls the router itself, so it never waits on a second workflow for a slot
// on its own queue.
type Harness struct {
	dispatcher llmports.StructuredCompleter
	router     llmports.CompletionService
	batches    llmports.BatchCompletionService
	providers  llmports.ProviderAvailability
}

var (
	_ llmports.TaskRunner   = (*Harness)(nil)
	_ llmports.TaskDeferrer = (*Harness)(nil)
	_ llmports.TaskBatcher  = (*Harness)(nil)
)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Harness {
	return &Harness{
		dispatcher: p.Dispatcher,
		router:     p.Router,
		batches:    p.Batches,
		providers:  p.Providers,
	}
}

// Available reports whether Nova can run for the tenant: AI is on and the
// organization has an enabled provider chosen for Nova.
func (h *Harness) Available(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error) {
	return h.providers.NovaReady(ctx, tenantInfo)
}

// Run does the task and decodes the answer into out.
func (h *Harness) Run(
	ctx context.Context,
	task *llmports.NovaTask,
	out any,
) (*llmports.StructuredCompletionResult, error) {
	available, err := h.Available(ctx, task.TenantInfo)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, llmports.ErrNoProviderConfigured
	}
	if len(task.Reads) > 0 && h.onWorker(ctx) {
		return h.investigate(ctx, task, out)
	}

	result, err := h.completer(ctx).CompleteStructured(ctx, request(task))
	if err != nil {
		return nil, fmt.Errorf("nova %s: %w", task.Name, err)
	}

	return result, llmports.DecodeAnswer(task.Name, result.Text, out)
}

func (h *Harness) completer(ctx context.Context) llmports.StructuredCompleter {
	if h.onWorker(ctx) {
		return h.router
	}

	return h.dispatcher
}

// onWorker reports a call made inside a Temporal activity, which calls the
// router itself; anywhere else the call goes to a worker through the
// dispatcher.
func (h *Harness) onWorker(ctx context.Context) bool {
	return activity.IsActivity(ctx)
}

func request(task *llmports.NovaTask) *llmports.StructuredCompletionRequest {
	return &llmports.StructuredCompletionRequest{
		TenantInfo:   task.TenantInfo,
		Task:         task.Name,
		System:       identity + "\n\n" + task.Instructions,
		Context:      task.Context,
		Files:        task.Files,
		OutputSchema: task.Schema,
		SchemaName:   task.SchemaName,
		MaxTokens:    task.MaxTokens,
		Reasoning:    task.Reasoning,
		Attribution:  attribution(task),
		Nova:         true,
	}
}

func attribution(task *llmports.NovaTask) aiusageports.AIUsageAttribution {
	if task.Attribution != nil {
		return *task.Attribution
	}

	return aiusageports.AIUsageAttribution{UserID: task.UserID, Feature: task.Feature}
}
