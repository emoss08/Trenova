package agentdefinition

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	maxNameLength        = 100
	maxDescriptionLength = 500
	maxInstructionsRunes = 20000
	maxGuardrails        = 20
	maxGuardrailRunes    = 300
	maxTools             = 64
	maxSystemKeyLength   = 50
	maxCronLength        = 100
	minIntervalSeconds   = 60
	minDecisionTimeout   = 60
	maxDecisionTimeout   = 30 * 24 * 60 * 60
	minRunTimeoutSeconds = 60
	maxRunTimeoutSeconds = 3600
	maxToolCallsCeiling  = 64
	maxConcurrentRuns    = 10

	DefaultDecisionTimeoutSeconds = 86400
	DefaultRunTimeoutSeconds      = 600
	DefaultMaxToolCalls           = 12
	DefaultCronTimezone           = "UTC"
)

type Definition struct {
	bun.BaseModel `bun:"table:agent_definitions,alias:agdef" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Name         string   `json:"name"         bun:"name,type:VARCHAR(100),notnull"`
	Description  string   `json:"description"  bun:"description,type:TEXT,nullzero"`
	Template     Template `json:"template"     bun:"template,type:VARCHAR(50),nullzero"`
	Instructions string   `json:"instructions" bun:"instructions,type:TEXT,nullzero"`
	Guardrails   []string `json:"guardrails"   bun:"guardrails,type:TEXT[],array,nullzero"`

	ToolNames       []string                      `json:"toolNames"       bun:"tool_names,type:TEXT[],array,nullzero"`
	ToolTiers       map[string]agent.AutonomyTier `json:"toolTiers"       bun:"tool_tiers,type:JSONB,nullzero"`
	AutonomyCeiling agent.AutonomyTier            `json:"autonomyCeiling" bun:"autonomy_ceiling,type:VARCHAR(50),notnull"`

	Enabled                bool `json:"enabled"                bun:"enabled,type:BOOLEAN,notnull"`
	ShadowMode             bool `json:"shadowMode"             bun:"shadow_mode,type:BOOLEAN,notnull"`
	DecisionTimeoutSeconds int  `json:"decisionTimeoutSeconds" bun:"decision_timeout_seconds,type:INTEGER,notnull"`

	TriggerMode       TriggerMode       `json:"triggerMode"       bun:"trigger_mode,type:VARCHAR(20),notnull"`
	CronExpression    string            `json:"cronExpression"    bun:"cron_expression,type:VARCHAR(100),nullzero"`
	CronTimezone      string            `json:"cronTimezone"      bun:"cron_timezone,type:VARCHAR(100),nullzero"`
	EventKinds        []agent.EventKind `json:"eventKinds"        bun:"event_kinds,type:TEXT[],array,nullzero"`
	IntervalSeconds   int               `json:"intervalSeconds"   bun:"interval_seconds,type:INTEGER,nullzero"`
	EndsAt            *int64            `json:"endsAt"            bun:"ends_at,type:BIGINT,nullzero"`
	MaxConcurrentRuns int               `json:"maxConcurrentRuns" bun:"max_concurrent_runs,type:INTEGER,notnull"`
	RunTimeoutSeconds int               `json:"runTimeoutSeconds" bun:"run_timeout_seconds,type:INTEGER,notnull"`
	MaxToolCalls      int               `json:"maxToolCalls"      bun:"max_tool_calls,type:INTEGER,notnull"`

	Icon   string `json:"icon"   bun:"icon,type:VARCHAR(40),nullzero"`
	Accent string `json:"accent" bun:"accent,type:VARCHAR(20),nullzero"`

	ContextProviders    []ContextProvider `json:"contextProviders"    bun:"context_providers,type:TEXT[],array,nullzero"`
	OutputMode          OutputMode        `json:"outputMode"          bun:"output_mode,type:VARCHAR(20),notnull"`
	PreferredProviderID pulid.ID          `json:"preferredProviderId" bun:"preferred_provider_id,type:VARCHAR(100),nullzero"`
	SystemKey           string            `json:"systemKey"           bun:"system_key,type:VARCHAR(50),nullzero"`

	LastRunAt *int64 `json:"lastRunAt" bun:"last_run_at,type:BIGINT,nullzero"`
	NextRunAt *int64 `json:"nextRunAt" bun:"next_run_at,type:BIGINT,nullzero"`

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
		d.ApplyDefaults()
		d.CreatedAt = now
		d.UpdatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}

func (d *Definition) GetID() pulid.ID { return d.ID }

func (d *Definition) GetCreatedAt() int64 { return d.CreatedAt }

func (d *Definition) GetTableName() string { return "agent_definitions" }

func (d *Definition) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "agdef",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "template", Type: domaintypes.FieldTypeEnum},
			{Name: "trigger_mode", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (d *Definition) ApplyDefaults() {
	if d.TriggerMode == "" {
		d.TriggerMode = TriggerChat
	}
	if d.OutputMode == "" {
		d.OutputMode = OutputConversational
	}
	if d.AutonomyCeiling == "" {
		d.AutonomyCeiling = agent.TierPropose
	}
	if d.DecisionTimeoutSeconds == 0 {
		d.DecisionTimeoutSeconds = DefaultDecisionTimeoutSeconds
	}
	if d.RunTimeoutSeconds == 0 {
		d.RunTimeoutSeconds = DefaultRunTimeoutSeconds
	}
	if d.MaxToolCalls == 0 {
		d.MaxToolCalls = DefaultMaxToolCalls
	}
	if d.MaxConcurrentRuns == 0 {
		d.MaxConcurrentRuns = 1
	}
	if strings.TrimSpace(d.CronTimezone) == "" {
		d.CronTimezone = DefaultCronTimezone
	}
}

func (d *Definition) EffectiveTier(tool string, toolTier agent.AutonomyTier) agent.AutonomyTier {
	requested := toolTier
	if override, ok := d.ToolTiers[tool]; ok && override.IsValid() {
		requested = override
	}
	if requested == "" {
		requested = agent.TierPropose
	}

	if tierRank(d.AutonomyCeiling) < tierRank(requested) {
		return d.AutonomyCeiling
	}

	return requested
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

func (d *Definition) AllowsTool(name string) bool {
	for _, tool := range d.ToolNames {
		if tool == name {
			return true
		}
	}

	return false
}

func (d *Definition) IsSystem() bool {
	return strings.TrimSpace(d.SystemKey) != ""
}

func (d *Definition) EffectiveShadow(organizationShadow bool) bool {
	return organizationShadow || d.ShadowMode
}

func (d *Definition) IsBackground() bool {
	switch d.TriggerMode {
	case TriggerScheduled, TriggerContinuous, TriggerEvent:
		return true
	default:
		return false
	}
}

func (d *Definition) IsDue(now int64) bool {
	if !d.Enabled || d.NextRunAt == nil {
		return false
	}
	if d.TriggerMode != TriggerScheduled && d.TriggerMode != TriggerContinuous {
		return false
	}
	if d.EndsAt != nil && *d.EndsAt <= now {
		return false
	}

	return *d.NextRunAt <= now
}

func (d *Definition) ComputeNextRun(now int64) (int64, error) {
	switch d.TriggerMode {
	case TriggerScheduled:
		timezone := d.CronTimezone
		if strings.TrimSpace(timezone) == "" {
			timezone = DefaultCronTimezone
		}

		return cronutils.NextRun(d.CronExpression, timezone, now)
	case TriggerContinuous:
		if d.IntervalSeconds < minIntervalSeconds {
			return 0, fmt.Errorf("interval of %d seconds is below the minimum", d.IntervalSeconds)
		}

		return now + int64(d.IntervalSeconds), nil
	default:
		return 0, fmt.Errorf("a %s agent has no schedule", d.TriggerMode)
	}
}

func (d *Definition) HasContextProvider(provider ContextProvider) bool {
	if len(d.ContextProviders) == 0 {
		return true
	}
	for _, configured := range d.ContextProviders {
		if configured == provider {
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
		validation.Field(&d.Template,
			validation.When(d.Template != "",
				domainvalidation.ValidEnum[Template]("Template is not one this system offers"),
			),
		),
		validation.Field(&d.Instructions,
			validation.RuneLength(0, maxInstructionsRunes).
				Error("Instructions cannot be longer than 20000 characters"),
		),
		validation.Field(&d.AutonomyCeiling,
			validation.Required.Error("Autonomy ceiling is required"),
			domainvalidation.ValidEnum[agent.AutonomyTier]("Autonomy ceiling is invalid"),
		),
		validation.Field(&d.TriggerMode,
			validation.Required.Error("Trigger mode is required"),
			domainvalidation.ValidEnum[TriggerMode]("Trigger mode is invalid"),
		),
		validation.Field(&d.OutputMode,
			validation.Required.Error("Output mode is required"),
			domainvalidation.ValidEnum[OutputMode]("Output mode is invalid"),
		),
		validation.Field(&d.DecisionTimeoutSeconds,
			validation.Min(minDecisionTimeout).Error("Decision timeout must be at least one minute"),
			validation.Max(maxDecisionTimeout).Error("Decision timeout cannot exceed 30 days"),
		),
		validation.Field(&d.RunTimeoutSeconds,
			validation.Min(minRunTimeoutSeconds).Error("Run timeout must be at least one minute"),
			validation.Max(maxRunTimeoutSeconds).Error("Run timeout cannot exceed one hour"),
		),
		validation.Field(&d.MaxToolCalls,
			validation.Min(1).Error("An agent needs at least one tool call per run"),
			validation.Max(maxToolCallsCeiling).Error("An agent cannot make more than 64 tool calls per run"),
		),
		validation.Field(&d.MaxConcurrentRuns,
			validation.Min(1).Error("At least one concurrent run is required"),
			validation.Max(maxConcurrentRuns).Error("At most 10 concurrent runs are allowed"),
		),
		validation.Field(&d.SystemKey,
			validation.Length(0, maxSystemKeyLength).
				Error("System key cannot be longer than 50 characters"),
		),
		validation.Field(&d.Icon,
			validation.By(func(any) error {
				if d.Icon != "" && !IsKnownIcon(d.Icon) {
					return errIconUnknown
				}

				return nil
			}),
		),
		validation.Field(&d.Accent,
			validation.By(func(any) error {
				if d.Accent != "" && !IsKnownAccent(d.Accent) {
					return errAccentUnknown
				}

				return nil
			}),
		),
	))

	d.validateGuardrails(multiErr)
	d.validateTools(multiErr)
	d.validateTrigger(multiErr)
	d.validateContextProviders(multiErr)
}

func (d *Definition) validateGuardrails(multiErr *errortypes.MultiError) {
	if len(d.Guardrails) > maxGuardrails {
		multiErr.Add(
			"guardrails",
			errortypes.ErrInvalid,
			"An agent cannot have more than 20 guardrails",
		)
	}

	for idx, rule := range d.Guardrails {
		field := fmt.Sprintf("guardrails[%d]", idx)
		trimmed := strings.TrimSpace(rule)
		if trimmed == "" {
			multiErr.Add(field, errortypes.ErrInvalid, "A guardrail cannot be empty")
			continue
		}
		if len([]rune(trimmed)) > maxGuardrailRunes {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				"A guardrail cannot be longer than 300 characters",
			)
		}
	}
}

func (d *Definition) validateTools(multiErr *errortypes.MultiError) {
	if len(d.ToolNames) > maxTools {
		multiErr.Add(
			"toolNames",
			errortypes.ErrInvalid,
			"An agent cannot be given more than 64 tools",
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

	for tool, tier := range d.ToolTiers {
		field := "toolTiers." + tool
		if _, enabled := seen[tool]; !enabled {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				fmt.Sprintf("%q is not one of this agent's tools", tool),
			)
			continue
		}
		if !tier.IsValid() {
			multiErr.Add(field, errortypes.ErrInvalid, "Autonomy tier is invalid")
		}
	}
}

func (d *Definition) validateTrigger(multiErr *errortypes.MultiError) {
	switch d.TriggerMode {
	case TriggerScheduled:
		expression := strings.TrimSpace(d.CronExpression)
		switch {
		case expression == "":
			multiErr.Add(
				"cronExpression",
				errortypes.ErrRequired,
				"A scheduled agent needs a cron expression",
			)
		case len(expression) > maxCronLength:
			multiErr.Add(
				"cronExpression",
				errortypes.ErrInvalid,
				"Cron expression cannot be longer than 100 characters",
			)
		case cronutils.Validate(expression) != nil:
			multiErr.Add(
				"cronExpression",
				errortypes.ErrInvalid,
				"Cron expression must have five fields: minute, hour, day of month, month, day of week",
			)
		}

		if timezone := strings.TrimSpace(d.CronTimezone); timezone != "" {
			if _, err := time.LoadLocation(timezone); err != nil {
				multiErr.Add(
					"cronTimezone",
					errortypes.ErrInvalid,
					"Timezone must be a valid IANA name such as America/Chicago",
				)
			}
		}
	case TriggerEvent:
		if len(d.EventKinds) == 0 {
			multiErr.Add(
				"eventKinds",
				errortypes.ErrRequired,
				"An event-triggered agent needs at least one event",
			)
		}
		for idx, kind := range d.EventKinds {
			if !kind.IsValid() {
				multiErr.Add(
					fmt.Sprintf("eventKinds[%d]", idx),
					errortypes.ErrInvalid,
					fmt.Sprintf("%q is not an event this system raises", kind),
				)
			}
		}
	case TriggerContinuous:
		if d.IntervalSeconds < minIntervalSeconds {
			multiErr.Add(
				"intervalSeconds",
				errortypes.ErrInvalid,
				"A continuous agent must wait at least one minute between runs",
			)
		}
	case TriggerChat:
	}

	if d.EndsAt != nil && *d.EndsAt <= 0 {
		multiErr.Add("endsAt", errortypes.ErrInvalid, "End time is invalid")
	}
}

func (d *Definition) validateContextProviders(multiErr *errortypes.MultiError) {
	for idx, provider := range d.ContextProviders {
		if !provider.IsValid() {
			multiErr.Add(
				fmt.Sprintf("contextProviders[%d]", idx),
				errortypes.ErrInvalid,
				fmt.Sprintf("%q is not a context this system can provide", provider),
			)
		}
	}
}
