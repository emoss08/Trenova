package aiprovider

import (
	"fmt"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiplatform"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// NovaPlatforms are the platforms Nova runs through: those that name Claude
// models by their Claude API id, which is how Nova names its model. Bedrock
// prefixes ids of its own and Foundry calls a deployment the organization
// named, so Nova's model has no name Trenova can know there.
func NovaPlatforms() []aiplatform.Platform {
	return []aiplatform.Platform{
		aiplatform.PlatformDirect,
		aiplatform.PlatformClaudeAWS,
		aiplatform.PlatformClaudeGoogleCloud,
		aiplatform.PlatformVertexAI,
	}
}

// CanServeNova reports whether this provider can be the one Nova runs on, and
// why not when it cannot. Nova runs on Claude, so only an Anthropic Messages
// provider on a platform that names Claude models the way the Claude API does
// qualifies; its own model does not matter, since Nova calls it with Nova's.
func (p *Provider) CanServeNova() (servable bool, reason string) {
	switch {
	case p.Kind != KindAnthropicMessages:
		return false, "Nova runs on Claude, so its provider must be an Anthropic provider"
	case !slices.Contains(NovaPlatforms(), p.ResolvedPlatform()):
		return false, fmt.Sprintf(
			"Nova cannot run through %s; use the Claude API, Claude Platform on AWS or Google Cloud, or Vertex AI",
			p.ResolvedPlatform(),
		)
	default:
		return true, ""
	}
}

func (p *Provider) validateNova(multiErr *errortypes.MultiError) {
	if !p.Nova {
		return
	}
	if ok, reason := p.CanServeNova(); !ok {
		multiErr.Add("nova", errortypes.ErrInvalid, reason)
	}
}

func (Provider) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_ai_providers_org_name",
			Field:   "name",
			Message: "An AI provider with this name already exists",
		},
		{
			Name:    "idx_ai_providers_nova",
			Field:   "nova",
			Message: "Another provider already runs Nova in your organization",
		},
		{
			Name:    "ck_ai_providers_nova",
			Field:   "nova",
			Message: "Nova runs only on an Anthropic provider through the Claude API, Claude Platform or Vertex AI",
		},
	}
}
