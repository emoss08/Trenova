package aiprovider

import (
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	MinContextWindowTokens  = 2048
	MaxContextWindowTokens  = 2_000_000
	hostedContextWindow     = 128_000
	selfHostedContextWindow = 32_768
)

func (k Kind) DefaultContextWindow() int {
	switch k {
	case KindAnthropicMessages, KindOpenAIResponses:
		return hostedContextWindow
	case KindOpenAIChat, KindOllama:
		return selfHostedContextWindow
	default:
		return selfHostedContextWindow
	}
}

func (p *Provider) ResolvedContextWindow() int {
	if p.ContextWindowTokens != nil && *p.ContextWindowTokens > 0 {
		return *p.ContextWindowTokens
	}

	return p.Kind.DefaultContextWindow()
}

func (p *Provider) ConfiguredContextWindow() int {
	if p.ContextWindowTokens == nil {
		return 0
	}

	return *p.ContextWindowTokens
}

func (p *Provider) validateContextWindow(multiErr *errortypes.MultiError) {
	if p.ContextWindowTokens == nil {
		return
	}

	window := *p.ContextWindowTokens
	switch {
	case window < MinContextWindowTokens:
		multiErr.Add("contextWindowTokens", errortypes.ErrInvalid,
			"Context window must be at least 2048 tokens")
	case window > MaxContextWindowTokens:
		multiErr.Add("contextWindowTokens", errortypes.ErrInvalid,
			"Context window cannot exceed 2000000 tokens")
	case window <= p.ResolvedMaxTokens():
		multiErr.Add("contextWindowTokens", errortypes.ErrInvalid,
			"Context window must be larger than max tokens, since the reply is written inside it")
	}
}
