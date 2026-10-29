package adapterregistry

import (
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/anthropicadapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/openaiadapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// NewRegistry builds the registry over every protocol this system speaks.
func NewRegistry() *Registry {
	adapters := []modeladapter.Adapter{
		anthropicadapter.NewAnthropicAdapter(),
		openaiadapter.NewOpenAIResponsesAdapter(),
		openaiadapter.NewOpenAIChatAdapter(),
	}

	byKind := make(map[aiprovider.Kind]modeladapter.Adapter, len(adapters))
	embedders := make(map[aiprovider.Kind]modeladapter.Embedder, len(adapters))
	for _, adapter := range adapters {
		byKind[adapter.Kind()] = adapter
		if embedder, ok := adapter.(modeladapter.Embedder); ok &&
			adapter.Kind().SupportsEmbedding() {
			embedders[adapter.Kind()] = embedder
		}
	}

	return &Registry{byKind: byKind, embedders: embedders}
}

// Registry resolves an adapter for a provider's protocol.
type Registry struct {
	byKind    map[aiprovider.Kind]modeladapter.Adapter
	embedders map[aiprovider.Kind]modeladapter.Embedder
}

// Get returns the adapter for kind.
func (r *Registry) Get(kind aiprovider.Kind) (modeladapter.Adapter, error) {
	adapter, ok := r.byKind[kind]
	if !ok {
		return nil, fmt.Errorf("no adapter for provider kind %q", kind)
	}

	return adapter, nil
}

func (r *Registry) Embedder(kind aiprovider.Kind) (modeladapter.Embedder, error) {
	embedder, ok := r.embedders[kind]
	if !ok {
		return nil, errortypes.NewBusinessError(
			"The {0} protocol has no embedding endpoint", string(kind),
		).WithInternal(llmports.ErrNoProviderConfigured)
	}

	return embedder, nil
}
