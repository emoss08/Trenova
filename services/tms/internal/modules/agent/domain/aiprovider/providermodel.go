package aiprovider

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/aimodel"
)

// ConfiguredContextWindow is the window the provider is configured with, or
// zero when it is left to be read off the model id.
func (p *Provider) ConfiguredContextWindow() int {
	if p.ContextWindowTokens == nil || *p.ContextWindowTokens <= 0 {
		return 0
	}

	return *p.ContextWindowTokens
}

// ContextWindow is the context window of the provider's model: the one it is
// configured with, and otherwise the one its model id names.
func (p *Provider) ContextWindow() int {
	return aimodel.WindowOr(p.ConfiguredContextWindow(), p.Model)
}

// FileInput reports whether the provider's model reads files itself, and the
// bounds it reads them within. Only an Anthropic Messages provider serving a
// model this can read as Claude does: the same protocol also fronts models
// from other vendors, behind gateways and compatible endpoints, that refuse a
// document block, and a model that refuses one fails the whole request.
func (p *Provider) FileInput() (aimodel.FileInput, bool) {
	if p.Kind != KindAnthropicMessages {
		return aimodel.FileInput{}, false
	}
	claude, ok := aimodel.ParseClaudeModel(p.Model)
	if !ok {
		return aimodel.FileInput{}, false
	}

	return claude.FileInput(), true
}
