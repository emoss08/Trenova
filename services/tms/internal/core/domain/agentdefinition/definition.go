package agentdefinition

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	maxNameLength        = 100
	maxDescriptionLength = 500
	maxFocusLength       = 2000
	maxTools             = 32
)

// Definition is an organization's configuration of a Trenova agent template.
//
// There is deliberately no system-prompt field. An organization that could write
// one could write "you are a general coding assistant" and undo every boundary
// the product depends on. What it configures instead is composition: which
// template, which subset of that template's tools, how much autonomy, and a
// bounded focus note that is delivered to the model as data rather than as
// instruction.
type Definition struct {
	bun.BaseModel `bun:"table:agent_definitions,alias:agdef" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Name        string `json:"name"        bun:"name,type:VARCHAR(100),notnull"`
	Description string `json:"description" bun:"description,type:TEXT,nullzero"`
	Kind        Kind   `json:"kind"        bun:"kind,type:VARCHAR(50),notnull"`

	// Focus is the organization's own guidance — "we prioritise reefer loads",
	// "always check the detention policy first". It is never concatenated into the
	// system prompt; see BuildFocusSection.
	Focus string `json:"focus" bun:"focus,type:TEXT,nullzero"`

	// ToolNames is the subset of the kind's tools this agent may use. Empty means
	// the agent can only answer, not act.
	ToolNames []string `json:"toolNames" bun:"tool_names,type:TEXT[],array,nullzero"`

	// AutonomyCeiling caps every tool's autonomy. It can only lower a tool's own
	// tier, never raise it, so a configuration cannot promote a propose-only tool
	// into one that executes on its own.
	AutonomyCeiling agent.AutonomyTier `json:"autonomyCeiling" bun:"autonomy_ceiling,type:VARCHAR(50),notnull"`

	Enabled bool `json:"enabled" bun:"enabled,type:BOOLEAN,notnull"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (d *Definition) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("agdef_")
		}
		d.CreatedAt = now
		d.UpdatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}

func (d *Definition) GetID() pulid.ID { return d.ID }

func (d *Definition) GetTableName() string { return "agent_definitions" }

func (d *Definition) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "agdef",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "kind", Type: domaintypes.FieldTypeEnum},
		},
	}
}

// EffectiveTier applies the ceiling to a tool's own tier, returning whichever is
// more restrictive.
func (d *Definition) EffectiveTier(toolTier agent.AutonomyTier) agent.AutonomyTier {
	if tierRank(d.AutonomyCeiling) < tierRank(toolTier) {
		return d.AutonomyCeiling
	}

	return toolTier
}

func tierRank(tier agent.AutonomyTier) int {
	switch tier {
	case agent.TierPropose:
		return 0
	case agent.TierActWithApproval:
		return 1
	case agent.TierAutoExecute:
		return 2
	default:
		return 0
	}
}

// AllowsTool reports whether a tool name is among those configured.
func (d *Definition) AllowsTool(name string) bool {
	for _, tool := range d.ToolNames {
		if tool == name {
			return true
		}
	}

	return false
}

func (d *Definition) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(d,
		validation.Field(&d.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&d.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&d.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxNameLength).
				Error("Name cannot be longer than 100 characters"),
		),
		validation.Field(&d.Description,
			validation.Length(0, maxDescriptionLength).
				Error("Description cannot be longer than 500 characters"),
		),
		validation.Field(&d.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[Kind]("Kind is not an agent template this system offers"),
		),
		// Bounded because the focus note is sent on every turn, and because a very
		// long one is usually an attempt to write a system prompt in disguise.
		validation.Field(&d.Focus,
			validation.Length(0, maxFocusLength).
				Error("Focus cannot be longer than 2000 characters"),
		),
		validation.Field(&d.AutonomyCeiling,
			validation.Required.Error("Autonomy ceiling is required"),
			domainvalidation.ValidEnum[agent.AutonomyTier]("Autonomy ceiling is invalid"),
		),
	))

	d.validateTools(multiErr)
}

func (d *Definition) validateTools(multiErr *errortypes.MultiError) {
	if len(d.ToolNames) > maxTools {
		multiErr.Add(
			"toolNames",
			errortypes.ErrInvalid,
			"An agent cannot be given more than 32 tools",
		)
	}

	seen := make(map[string]struct{}, len(d.ToolNames))
	for idx, name := range d.ToolNames {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			multiErr.Add(
				fmt.Sprintf("toolNames[%d]", idx),
				errortypes.ErrInvalid,
				"Tool name cannot be empty",
			)
			continue
		}
		if _, duplicate := seen[trimmed]; duplicate {
			multiErr.Add(
				fmt.Sprintf("toolNames[%d]", idx),
				errortypes.ErrDuplicate,
				"Tool is listed more than once",
			)
		}
		seen[trimmed] = struct{}{}
	}

	if d.Kind.IsValid() && !d.Kind.MutatingAllowed() && len(d.ToolNames) > 0 {
		multiErr.Add(
			"toolNames",
			errortypes.ErrInvalid,
			"A general assistant answers questions only and cannot be given tools",
		)
	}
}
