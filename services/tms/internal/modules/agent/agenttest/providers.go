package agenttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
)

// StubProviders lists the same providers for every task.
type StubProviders struct {
	aiproviderports.AIProviderRepository

	Providers []*aiprovider.Provider
}

func (r *StubProviders) ListForTask(
	context.Context,
	aiproviderports.ListAIProvidersForTaskRequest,
) ([]*aiprovider.Provider, error) {
	return r.Providers, nil
}
