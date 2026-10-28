package agentdefinition

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmark"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

const (
	MaxNameLength        = 100
	MaxDescriptionLength = 500
	MaxInstructionsRunes = 20000
	MaxGuardrails        = 20
	MaxGuardrailRunes    = 300
	MaxTools             = 64
	maxSystemKeyLength   = 50
	MaxCronLength        = 100
	MinIntervalSeconds   = 60
	minDecisionTimeout   = 60
	maxDecisionTimeout   = 30 * 24 * 60 * 60
	minRunTimeoutSeconds = 60
	maxRunTimeoutSeconds = 3600
	maxToolCallsCeiling  = 64
	maxConcurrentRuns    = 10

	// MaxDelegates bounds the agents one agent may hand work to. Each is
	// described in the agent's prompt, so the list is paid for on every turn.
	MaxDelegates = 8

	DefaultDecisionTimeoutSeconds = 86400
	DefaultRunTimeoutSeconds      = 600
	DefaultMaxToolCalls           = 12
	DefaultCronTimezone           = "UTC"
)

type Definition struct {
	keel.Table `keel:"agent_definitions"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:agdef_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Name         string                 `json:"name"         keel:"name"`
	Description  string                 `json:"description"  keel:"description,nullzero"`
	Template     agenttemplate.Template `json:"template"     keel:"template,nullzero"`
	Instructions string                 `json:"instructions" keel:"instructions,nullzero"`
	Guardrails   []string               `json:"guardrails"   keel:"guardrails,nullzero"`

	ToolNames       []string                      `json:"toolNames"       keel:"tool_names,nullzero"`
	ToolTiers       map[string]agent.AutonomyTier `json:"toolTiers"       keel:"tool_tiers,jsonb,nullzero"`
	AutonomyCeiling agent.AutonomyTier            `json:"autonomyCeiling" keel:"autonomy_ceiling"`

	DataAccessCeiling DataAccessCeiling `json:"dataAccessCeiling" keel:"data_access_ceiling"`

	UnattendedRoleID pulid.ID `json:"unattendedRoleId" keel:"unattended_role_id,nullzero"`

	Enabled bool `json:"enabled"                keel:"enabled"`
	// DisabledAt and DisabledByID say when the agent was last turned off and
	// by whom; both are cleared when it is turned back on.
	DisabledAt             *int64    `json:"disabledAt,omitempty"   keel:"disabled_at"`
	DisabledByID           *pulid.ID `json:"disabledById,omitempty" keel:"disabled_by_id"`
	ShadowMode             bool      `json:"shadowMode"             keel:"shadow_mode"`
	DecisionTimeoutSeconds int       `json:"decisionTimeoutSeconds" keel:"decision_timeout_seconds"`

	TriggerMode       TriggerMode       `json:"triggerMode"       keel:"trigger_mode"`
	CronExpression    string            `json:"cronExpression"    keel:"cron_expression,nullzero"`
	CronTimezone      string            `json:"cronTimezone"      keel:"cron_timezone,nullzero"`
	EventKinds        []agent.EventKind `json:"eventKinds"        keel:"event_kinds,nullzero"`
	IntervalSeconds   int               `json:"intervalSeconds"   keel:"interval_seconds,nullzero"`
	EndsAt            *int64            `json:"endsAt"            keel:"ends_at"`
	MaxConcurrentRuns int               `json:"maxConcurrentRuns" keel:"max_concurrent_runs"`
	RunTimeoutSeconds int               `json:"runTimeoutSeconds" keel:"run_timeout_seconds"`
	MaxToolCalls      int               `json:"maxToolCalls"      keel:"max_tool_calls"`

	// MonthlyBudgetUSD caps what the agent's runs may cost in a calendar
	// month; nil is no cap. DailyRunLimit caps how many runs it may start in
	// a day and ToolDailyLimits how many times it may execute each tool in a
	// day; zero is no cap. SimulationMode keeps its writes from happening at
	// all: each is previewed and recorded as what it would have changed.
	MonthlyBudgetUSD *decimal.Decimal `json:"monthlyBudgetUsd" keel:"monthly_budget_usd"`
	DailyRunLimit    int              `json:"dailyRunLimit"    keel:"daily_run_limit"`
	// ToolDailyLimits is notnull because its column is: it was added as
	// NOT NULL DEFAULT '{}' while its tag said nullzero, and nullzero writes
	// SQL NULL for a nil map. Every insert of an agent with no per-tool cap —
	// which is every system agent — was therefore rejected by the constraint.
	// Its older sibling tool_tiers is nullable, which is why that one worked.
	ToolDailyLimits map[string]int `json:"toolDailyLimits"  keel:"tool_daily_limits,jsonb"`
	SimulationMode  bool           `json:"simulationMode"   keel:"simulation_mode"`

	MemoryTokenBudget *int `json:"memoryTokenBudget" keel:"memory_token_budget"`
	LearningOff       bool `json:"learningOff"       keel:"learning_off"`

	Icon   string `json:"icon"   keel:"icon,nullzero"`
	Accent string `json:"accent" keel:"accent,nullzero"`

	ContextProviders    []ContextProvider `json:"contextProviders"    keel:"context_providers,nullzero"`
	OutputMode          OutputMode        `json:"outputMode"          keel:"output_mode"`
	PreferredProviderID pulid.ID          `json:"preferredProviderId" keel:"preferred_provider_id,nullzero"`
	SystemKey           string            `json:"systemKey"           keel:"system_key,nullzero"`

	// DelegateIDs are the agents this one may hand a task to when a person is
	// talking to it: the ones it asks when the work needs tools it does not
	// hold. Each delegate works with its own tools, tiers, ceiling and budget,
	// as the same person, and never delegates further.
	DelegateIDs []pulid.ID `json:"delegateIds"    keel:"delegate_ids,nullzero"`
	// DelegateTopics names, per delegate id, the questions that go to that
	// agent ("Pay rates, people records"), so a person reading what an agent
	// does sees why it hands work on. A delegate without one is described by
	// its own description.
	DelegateTopics map[string]string `json:"delegateTopics" keel:"delegate_topics,jsonb,nullzero"`

	// DisabledToolNames are tools switched off from the agent's capabilities
	// page. Switching a tool off removes it from ToolNames, which is the
	// grant; it is listed here only so the page can still offer it back.
	DisabledToolNames []string `json:"disabledToolNames" keel:"disabled_tool_names,nullzero"`

	// MaxChangeItems is the most records one change by the agent may touch.
	// A bigger batch is refused with an instruction to split it, so each part
	// is proposed and approved on its own.
	MaxChangeItems int `json:"maxChangeItems" keel:"max_change_items"`

	// BusinessHoursOnly keeps the agent from making changes on its own
	// outside the window BusinessHoursStart to BusinessHoursEnd (minutes
	// after midnight) in BusinessHoursTimezone, or the organization's zone
	// when that is empty. Outside the window a change is held as a proposal
	// for a person instead of running.
	BusinessHoursOnly     bool   `json:"businessHoursOnly"     keel:"business_hours_only"`
	BusinessHoursStart    int    `json:"businessHoursStart"    keel:"business_hours_start"`
	BusinessHoursEnd      int    `json:"businessHoursEnd"      keel:"business_hours_end"`
	BusinessHoursTimezone string `json:"businessHoursTimezone" keel:"business_hours_timezone,nullzero"`

	// CreatedByID is the person who set the agent up; nil for the agents the
	// platform provisions and for those made before it was kept.
	CreatedByID *pulid.ID `json:"createdById,omitempty" keel:"created_by_id"`

	// AccessMode is who may use the agent among the people who may use the
	// assistant: everyone, or only the roles granted it. Only the agent
	// access service writes it.
	AccessMode AccessMode `json:"accessMode" keel:"access_mode"`

	LastRunAt *int64 `json:"lastRunAt" keel:"last_run_at"`
	NextRunAt *int64 `json:"nextRunAt" keel:"next_run_at"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
}

const IDPrefix = "agdef_"

func (d *Definition) BeforeCreate(ctx context.Context) error {
	if d.DataAccessCeiling == "" {
		d.DataAccessCeiling = DataAccessInternal
	}
	if d.MaxChangeItems == 0 {
		d.MaxChangeItems = 500
	}
	if d.BusinessHoursStart == 0 {
		d.BusinessHoursStart = 420
	}
	if d.BusinessHoursEnd == 0 {
		d.BusinessHoursEnd = 1080
	}
	if d.AccessMode == "" {
		d.AccessMode = AccessEveryone
	}
	d.ApplyDefaults()

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
			{Name: "description", Type: domaintypes.FieldTypeText},
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
	if d.DataAccessCeiling == "" {
		d.DataAccessCeiling = DataAccessInternal
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
	// No per-tool cap is an empty map, not an absent one, because the column
	// cannot hold an absent one.
	if d.ToolDailyLimits == nil {
		d.ToolDailyLimits = map[string]int{}
	}
	if d.AccessMode == "" {
		d.AccessMode = AccessEveryone
	}
	d.applyCapabilityDefaults()
}

func (d *Definition) EffectiveTier(tool string, toolTier agent.AutonomyTier) agent.AutonomyTier {
	requested := toolTier
	if override, ok := d.ToolTiers[tool]; ok && override.IsValid() {
		requested = override
	}
	if requested == "" {
		requested = agent.TierPropose
	}

	if requested.Above(d.AutonomyCeiling) {
		return d.AutonomyCeiling
	}

	return requested
}

// SetsToolTier reports whether the agent's own settings name a tier for the
// tool, rather than leaving it to the tool's default.
func (d *Definition) SetsToolTier(tool string) bool {
	override, ok := d.ToolTiers[tool]

	return ok && override.IsValid()
}

// WithinCeiling reports whether an agent may hold a tool at the tier.
func (d *Definition) WithinCeiling(tier agent.AutonomyTier) bool {
	return !tier.Above(d.AutonomyCeiling)
}

func (d *Definition) AllowsTool(name string) bool {
	return IsCoreTool(name) || slices.Contains(d.ToolNames, name)
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
		if d.IntervalSeconds < MinIntervalSeconds {
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
			validation.Length(1, MaxNameLength).
				Error("Name cannot be longer than 100 characters"),
		),
		validation.Field(&d.Description,
			validation.Length(0, MaxDescriptionLength).
				Error("Description cannot be longer than 500 characters"),
		),
		validation.Field(&d.Template,
			validation.When(
				d.Template != "",
				domainvalidation.ValidEnum[agenttemplate.Template](
					"Template is not one this system offers",
				),
			),
		),
		validation.Field(&d.Instructions,
			validation.RuneLength(0, MaxInstructionsRunes).
				Error("Instructions cannot be longer than 20000 characters"),
		),
		validation.Field(&d.AutonomyCeiling,
			validation.Required.Error("Autonomy ceiling is required"),
			domainvalidation.ValidEnum[agent.AutonomyTier]("Autonomy ceiling is invalid"),
		),
		validation.Field(
			&d.DataAccessCeiling,
			domainvalidation.ValidEnum[DataAccessCeiling](
				"Data access must be Internal or Restricted",
			),
		),
		validation.Field(&d.TriggerMode,
			validation.Required.Error("Trigger mode is required"),
			domainvalidation.ValidEnum[TriggerMode]("Trigger mode is invalid"),
		),
		validation.Field(&d.OutputMode,
			validation.Required.Error("Output mode is required"),
			domainvalidation.ValidEnum[OutputMode]("Output mode is invalid"),
		),
		validation.Field(
			&d.DecisionTimeoutSeconds,
			validation.Min(minDecisionTimeout).
				Error("Decision timeout must be at least one minute"),
			validation.Max(maxDecisionTimeout).Error("Decision timeout cannot exceed 30 days"),
		),
		validation.Field(&d.RunTimeoutSeconds,
			validation.Min(minRunTimeoutSeconds).Error("Run timeout must be at least one minute"),
			validation.Max(maxRunTimeoutSeconds).Error("Run timeout cannot exceed one hour"),
		),
		validation.Field(
			&d.MaxToolCalls,
			validation.Min(1).Error("An agent needs at least one tool call per run"),
			validation.Max(maxToolCallsCeiling).
				Error("An agent cannot make more than 64 tool calls per run"),
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
				if d.Icon != "" && !agentmark.IsKnownIcon(d.Icon) {
					return errIconUnknown
				}

				return nil
			}),
		),
		validation.Field(&d.Accent,
			validation.By(func(any) error {
				if d.Accent != "" && !agentmark.IsKnownAccent(d.Accent) {
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
	d.validateBudget(multiErr)
	d.validateMemoryBudget(multiErr)
	d.validateDelegates(multiErr)
	d.validateAccess(multiErr)
	d.validateCapabilities(multiErr)
}

// Delegates reports whether the agent may hand work to another agent at all:
// it names at least one, and it is an agent a person talks to. A run nobody
// is watching never delegates.
func (d *Definition) Delegates() bool {
	return len(d.DelegateIDs) > 0 && !d.IsBackground()
}

// MayDelegateTo reports whether the agent's allowlist names the agent.
func (d *Definition) MayDelegateTo(id pulid.ID) bool {
	return id.IsNotNil() && slices.Contains(d.DelegateIDs, id)
}

// DelegateRefusal says why the agent may not hand a task to delegate, in
// words the agent can pass on, or "" when it may. Whether the person may use
// the delegate, and whether it is in the same tenant, are the caller's to
// check: a delegate is read within the tenant, and a permission needs the
// person.
func (d *Definition) DelegateRefusal(delegate *Definition) string {
	if delegate != nil && !d.MayDelegateTo(delegate.ID) {
		return delegate.Name + " is not one of the agents you may hand work to."
	}

	return d.TaskRefusal(delegate)
}

// TaskRefusal is DelegateRefusal without the allowlist: why delegate may not
// take a task from inside this agent's conversation when the person, not the
// agent, chose it. The person's own access to the delegate is still the
// caller's to check.
func (d *Definition) TaskRefusal(delegate *Definition) string {
	switch {
	case delegate == nil:
		return "That agent no longer exists."
	case delegate.ID == d.ID:
		return "You cannot hand a task to yourself."
	case !delegate.Enabled:
		return delegate.Name + " is disabled, so it cannot take tasks. An administrator " +
			"can enable it in AI Control."
	case delegate.IsBackground():
		return delegate.Name + " runs on its own and cannot be handed a task."
	default:
		return ""
	}
}

// validateDelegates checks what the allowlist can say about itself. Whether
// each delegate exists in the tenant, is enabled and can be talked to needs
// the database, so the service checks that.
func (d *Definition) validateDelegates(multiErr *errortypes.MultiError) {
	if len(d.DelegateIDs) == 0 {
		return
	}
	if d.IsBackground() {
		multiErr.Add(
			"delegateIds",
			errortypes.ErrInvalid,
			"Only an agent people talk to can hand work to other agents",
		)
	}
	if len(d.DelegateIDs) > MaxDelegates {
		multiErr.Add(
			"delegateIds",
			errortypes.ErrInvalid,
			"An agent can hand work to at most 8 other agents",
		)
	}

	seen := make(map[pulid.ID]struct{}, len(d.DelegateIDs))
	for idx, id := range d.DelegateIDs {
		field := fmt.Sprintf("delegateIds[%d]", idx)
		switch {
		case id.IsNil():
			multiErr.Add(field, errortypes.ErrInvalid, "Agent cannot be empty")
			continue
		case d.ID.IsNotNil() && id == d.ID:
			multiErr.Add(field, errortypes.ErrInvalid, "An agent cannot hand work to itself")
		}
		if _, duplicate := seen[id]; duplicate {
			multiErr.Add(field, errortypes.ErrDuplicate, "Agent is listed more than once")
		}
		seen[id] = struct{}{}
	}
}

const maxDailyRunLimit = 10000

func (d *Definition) validateBudget(multiErr *errortypes.MultiError) {
	if d.MonthlyBudgetUSD != nil && d.MonthlyBudgetUSD.IsNegative() {
		multiErr.Add("monthlyBudgetUsd", errortypes.ErrInvalid, "A budget cannot be negative")
	}
	if d.DailyRunLimit < 0 || d.DailyRunLimit > maxDailyRunLimit {
		multiErr.Add(
			"dailyRunLimit",
			errortypes.ErrInvalid,
			"Runs per day must be between 0 and 10000; 0 means no limit",
		)
	}

	held := make(map[string]struct{}, len(d.ToolNames))
	for _, name := range d.ToolNames {
		held[strings.TrimSpace(name)] = struct{}{}
	}
	for tool, limit := range d.ToolDailyLimits {
		field := "toolDailyLimits." + tool
		if _, ok := held[tool]; !ok && !IsCoreTool(tool) {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				fmt.Sprintf("%q is not one of this agent's tools", tool),
			)
			continue
		}
		if limit < 0 || limit > maxDailyRunLimit {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				"A daily tool limit must be between 0 and 10000",
			)
		}
	}
}

// ToolDailyLimit is the cap on a tool's executions per day, or zero for none.
func (d *Definition) ToolDailyLimit(tool string) int {
	if d.ToolDailyLimits == nil {
		return 0
	}

	return d.ToolDailyLimits[tool]
}

func (d *Definition) validateGuardrails(multiErr *errortypes.MultiError) {
	if len(d.Guardrails) > MaxGuardrails {
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
		if len([]rune(trimmed)) > MaxGuardrailRunes {
			multiErr.Add(
				field,
				errortypes.ErrInvalid,
				"A guardrail cannot be longer than 300 characters",
			)
		}
	}
}

func (d *Definition) validateTools(multiErr *errortypes.MultiError) {
	if len(d.ToolNames) > MaxTools {
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
		if _, enabled := seen[tool]; !enabled && !IsCoreTool(tool) {
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
		case len(expression) > MaxCronLength:
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
		if d.IntervalSeconds < MinIntervalSeconds {
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

// NoteEnabledChange records who turned the agent off and when, or clears it
// once the agent is on again.
func (d *Definition) NoteEnabledChange(wasEnabled bool, by pulid.ID, at int64) {
	switch {
	case wasEnabled && !d.Enabled:
		d.DisabledAt = &at
		if by.IsNil() {
			d.DisabledByID = nil
		} else {
			d.DisabledByID = &by
		}
	case d.Enabled:
		d.DisabledAt = nil
		d.DisabledByID = nil
	}
}
