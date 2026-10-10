package aiprovider

import (
	"regexp"
	"strings"
)

// DefaultContextWindow is the window assumed for a model this cannot place.
// It is what the open-weight models most deployments serve take today, and
// erring small only compacts a conversation sooner than it had to.
const DefaultContextWindow = 128_000

// contextWindows are the windows of the model families a provider's model id
// names, most specific first. Each pattern is matched against the id in lower
// case, so it holds for the forms an id takes behind a gateway, on Bedrock or
// Vertex, or as a dated snapshot.
//
// Where a family's input and output share a window, the figure is what is
// left for input, because the input is what a conversation fills.
var contextWindows = []struct {
	pattern *regexp.Regexp
	tokens  int
}{
	{regexp.MustCompile(`claude`), 200_000},
	{regexp.MustCompile(`gpt-4\.1`), 1_047_576},
	{regexp.MustCompile(`gpt-5`), 272_000},
	{regexp.MustCompile(`gpt-4o|gpt-4-turbo`), 128_000},
	{regexp.MustCompile(`(^|[/.:])o[134](-|$)`), 200_000},
	{regexp.MustCompile(`gemini`), 1_048_576},
	{regexp.MustCompile(`llama-?4`), 1_000_000},
	{regexp.MustCompile(`llama|deepseek|nemotron|command-r`), 128_000},
	{regexp.MustCompile(`qwen|mistral|mixtral`), 32_768},
}

// ContextWindowFor is the context window, in tokens, of the model a provider
// is configured with. Model ids are free text, so a model this cannot place
// gets DefaultContextWindow.
func ContextWindowFor(model string) int {
	if tokens, ok := KnownContextWindow(model); ok {
		return tokens
	}

	return DefaultContextWindow
}

// KnownContextWindow is the window of a model family this can place by its
// id, and false for any other id.
func KnownContextWindow(model string) (int, bool) {
	id := strings.ToLower(strings.TrimSpace(model))
	if id == "" {
		return 0, false
	}
	for _, window := range contextWindows {
		if window.pattern.MatchString(id) {
			return window.tokens, true
		}
	}

	return 0, false
}

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
	return WindowOr(p.ConfiguredContextWindow(), p.Model)
}

// WindowOr is configured when it is set, and otherwise the window of model.
// A configured window wins because the operator saw the server; the id is
// only a guess at what is behind it.
func WindowOr(configured int, model string) int {
	if configured > 0 {
		return configured
	}

	return ContextWindowFor(model)
}
