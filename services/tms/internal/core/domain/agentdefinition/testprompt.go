package agentdefinition

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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
	bun.BaseModel `bun:"table:agent_test_prompts,alias:agtp" json:"-"`

	ID                pulid.ID  `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID  `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID  `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	AgentDefinitionID pulid.ID  `json:"agentDefinitionId" bun:"agent_definition_id,type:VARCHAR(100),notnull"`
	Prompt            string    `json:"prompt"            bun:"prompt,type:TEXT,notnull"`
	CreatedByID       *pulid.ID `json:"createdById"       bun:"created_by_id,type:VARCHAR(100),nullzero"`
	CreatedAt         int64     `json:"createdAt"         bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

// Validate trims the prompt and checks it is something to send.
func (p *TestPrompt) Validate(multiErr *errortypes.MultiError) {
	p.Prompt = strings.TrimSpace(p.Prompt)
	switch {
	case p.Prompt == "":
		multiErr.Add("prompt", errortypes.ErrRequired, "Enter a prompt to keep")
	case utf8.RuneCountInString(p.Prompt) > MaxTestPromptRunes:
		multiErr.Add("prompt", errortypes.ErrInvalidLength, "A prompt cannot be longer than 2000 characters")
	}
}

func (p *TestPrompt) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if p.ID.IsNil() {
			p.ID = pulid.MustNew(TestPromptIDPrefix)
		}
		p.CreatedAt = timeutils.NowUnix()
	}

	return nil
}
