// Package novamodel is the model Nova runs on. An organization chooses which
// of its Anthropic providers Nova uses (the credential, the platform, the
// limits and the monthly cap); the model is Nova's own, so Nova's prompts are
// written for one model and read the same in every organization. The Desk is
// different: there an organization picks its model.
package novamodel

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/aipricing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
)

// ID is the model every Nova task runs on.
const ID = "claude-haiku-5-5"

// Provider is p as Nova calls it: p's endpoint, credential, limits and health,
// with Nova's model, the reasoning Nova's prompts are tuned for and that
// model's list price, in place of what the organization chose for its own
// work on p. The advisor is off: it would bill another model for Nova's
// calls.
func Provider(p *aiprovider.Provider) *aiprovider.Provider {
	pinned := *p
	pinned.Model = ID
	pinned.ThinkingStyle = aireasoning.ThinkingStyleAuto
	pinned.ReasoningEffort = aireasoning.ReasoningOff
	pinned.ContextWindowTokens = nil
	pinned.AdvisorEnabled = false
	pinned.AdvisorModel = ""
	pinned.AdvisorRefusal = nil
	pinned.ExtraBody = nil
	pinned.InputCostPerMillion = nil
	pinned.OutputCostPerMillion = nil
	pinned.CacheReadCostPerMillion = nil
	pinned.CacheWriteCostPerMillion = nil
	if card, ok := aipricing.ListPrice(pinned.Kind, ID); ok {
		input, output := card.InputCostPerMillion, card.OutputCostPerMillion
		pinned.InputCostPerMillion = &input
		pinned.OutputCostPerMillion = &output
	}

	return &pinned
}
