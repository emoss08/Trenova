package agenttestprompt

import (
	"strings"
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	TestPromptIDPrefix = "agtp_"
	// MaxTestPromptRunes bounds one prompt a person keeps for trying an agent.
	MaxTestPromptRunes = 2000
	// MaxTestPrompts is how many prompts an agent keeps; saving another
	// past it is refused rather than dropping one silently.
	MaxTestPrompts = 20
)

// TestPrompt is a message kept with an agent for trying it again after an
// edit, so a change can be checked against the same questions each time.
type TestPrompt struct {
	keel.Table `keel:"agent_test_prompts"`

	ID                pulid.ID  `json:"id"                keel:"id,pk,autoid:agtp_"`
	BusinessUnitID    pulid.ID  `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID  `json:"organizationId"    keel:"organization_id,pk,tenant"`
	AgentDefinitionID pulid.ID  `json:"agentDefinitionId" keel:"agent_definition_id"`
	Prompt            string    `json:"prompt"            keel:"prompt"`
	CreatedByID       *pulid.ID `json:"createdById"       keel:"created_by_id"`
	CreatedAt         int64     `json:"createdAt"         keel:"created_at,autocreate"`
}

// Validate trims the prompt and checks it is something to send.
func (p *TestPrompt) Validate(multiErr *errortypes.MultiError) {
	p.Prompt = strings.TrimSpace(p.Prompt)
	switch {
	case p.Prompt == "":
		multiErr.Add("prompt", errortypes.ErrRequired, "Enter a prompt to keep")
	case utf8.RuneCountInString(p.Prompt) > MaxTestPromptRunes:
		multiErr.Add(
			"prompt",
			errortypes.ErrInvalidLength,
			"A prompt cannot be longer than 2000 characters",
		)
	}
}
