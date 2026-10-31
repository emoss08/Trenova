package agenttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/novaharness"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
)

// NovaRunner is the harness over a scripted completion, with a provider for
// every task. A nil completion gives a nil runner: Nova is not configured.
func NovaRunner(completion llmports.StructuredCompleter) llmports.TaskRunner {
	if completion == nil {
		return nil
	}

	return novaharness.New(novaharness.Params{Dispatcher: completion, Providers: everyTask{}})
}

type everyTask struct{}

func (everyTask) CanServe(context.Context, aitask.Task, pagination.TenantInfo) (bool, error) {
	return true, nil
}

// NovaOnRouter is the harness calling a scripted router itself, as it does on a
// worker, with a provider for every task.
func NovaOnRouter(router llmports.CompletionService) *novaharness.Harness {
	return novaharness.New(novaharness.Params{Router: router, Providers: everyTask{}})
}
