package aiprovider

import (
	"net/url"
	"strings"
)

type vendorHost struct {
	suffix string
	vendor string
}

var vendorHosts = []vendorHost{
	{suffix: "anthropic.com", vendor: "anthropic"},
	{suffix: "openai.com", vendor: "openai"},
	{suffix: "openai.azure.com", vendor: "openai"},
	{suffix: "googleapis.com", vendor: "gemini"},
	{suffix: "ai.google.dev", vendor: "gemini"},
	{suffix: "groq.com", vendor: "groq"},
	{suffix: "mistral.ai", vendor: "mistral"},
	{suffix: "openrouter.ai", vendor: "openrouter"},
	{suffix: "together.xyz", vendor: "together"},
	{suffix: "together.ai", vendor: "together"},
	{suffix: "fireworks.ai", vendor: "fireworks"},
	{suffix: "deepseek.com", vendor: "deepseek"},
	{suffix: "amazonaws.com", vendor: "bedrock"},
	{suffix: "huggingface.co", vendor: "huggingface"},
}

func (p *Provider) Vendor() string {
	base := strings.TrimSpace(p.BaseURL)
	if base == "" {
		base = p.Kind.DefaultBaseURL()
	}
	if parsed, err := url.Parse(base); err == nil {
		host := strings.ToLower(parsed.Hostname())
		for _, candidate := range vendorHosts {
			if host == candidate.suffix || strings.HasSuffix(host, "."+candidate.suffix) {
				return candidate.vendor
			}
		}
	}

	switch p.Kind {
	case KindAnthropicMessages:
		return "anthropic"
	case KindOpenAIResponses:
		return "openai"
	case KindOllama:
		return "ollama"
	case KindOpenAIChat:
		return ""
	default:
		return ""
	}
}

func (p *Provider) FailedLastTest() bool {
	return p.LastTest != nil && !p.LastTest.Success
}
