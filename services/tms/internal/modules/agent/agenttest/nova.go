package agenttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/novaharness"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
)

// NovaRunner is the harness over a scripted completion, with a Nova provider
// chosen. A nil completion gives a nil runner: Nova is not configured.
func NovaRunner(completion llmports.StructuredCompleter) llmports.TaskRunner {
	if completion == nil {
		return nil
	}

	return novaharness.New(novaharness.Params{Dispatcher: completion, Providers: novaChosen{}})
}

type novaChosen struct{}

func (novaChosen) NovaReady(context.Context, pagination.TenantInfo) (bool, error) {
	return true, nil
}

// NovaOnRouter is the harness calling a scripted router itself, as it does on a
// worker, with a Nova provider chosen.
func NovaOnRouter(router llmports.CompletionService) *novaharness.Harness {
	return novaharness.New(novaharness.Params{Router: router, Providers: novaChosen{}})
}
