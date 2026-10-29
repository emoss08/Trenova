package agentqualitycontrol

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*Control)(nil)
)

const (
	DefaultRunHourLocal        = 2
	DefaultMaxCasesPerAgent    = 50
	DefaultJudgeSampleRate     = 0.2
	DefaultRegressionThreshold = 0.10
	DefaultMinCases            = 10
	DefaultForceRerunDays      = 7

	MaxCasesPerAgentLimit = 500
	MaxMinCases           = 500
	MaxForceRerunDays     = 90
	MaxTimezoneChars      = 100
)

var (
	DefaultNightlyBudgetUSD = decimal.NewFromInt(5)
	DefaultMonthlyBudgetUSD = decimal.NewFromInt(50)
	MaxBudgetUSD            = decimal.NewFromInt(100000)
)

type Control struct {
	keel.Table `keel:"agent_quality_controls"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:aqc_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Enabled             bool            `json:"enabled"             keel:"enabled"`
	RunHourLocal        int             `json:"runHourLocal"        keel:"run_hour_local"`
	Timezone            string          `json:"timezone"            keel:"timezone,nullzero"`
	MaxCasesPerAgent    int             `json:"maxCasesPerAgent"    keel:"max_cases_per_agent"`
	NightlyBudgetUSD    decimal.Decimal `json:"nightlyBudgetUsd"    keel:"nightly_budget_usd"`
	MonthlyBudgetUSD    decimal.Decimal `json:"monthlyBudgetUsd"    keel:"monthly_budget_usd"`
	JudgeEnabled        bool            `json:"judgeEnabled"        keel:"judge_enabled"`
	JudgeSampleRate     float64         `json:"judgeSampleRate"     keel:"judge_sample_rate"`
	RegressionThreshold float64         `json:"regressionThreshold" keel:"regression_threshold"`
	MinCases            int             `json:"minCases"            keel:"min_cases"`
	ForceRerunDays      int             `json:"forceRerunDays"      keel:"force_rerun_days"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (c *Control) BeforeCreate(_ context.Context) error {
	if c.RunHourLocal == 0 {
		c.RunHourLocal = 2
	}
	if c.MaxCasesPerAgent == 0 {
		c.MaxCasesPerAgent = 50
	}
	if c.NightlyBudgetUSD.IsZero() {
		c.NightlyBudgetUSD = decimal.RequireFromString("5.00")
	}
	if c.MonthlyBudgetUSD.IsZero() {
		c.MonthlyBudgetUSD = decimal.RequireFromString("50.00")
	}
	if c.JudgeSampleRate == 0 {
		c.JudgeSampleRate = 0.2
	}
	if c.RegressionThreshold == 0 {
		c.RegressionThreshold = 0.10
	}
	if c.MinCases == 0 {
		c.MinCases = 10
	}
	if c.ForceRerunDays == 0 {
		c.ForceRerunDays = 7
	}

	return nil
}

func DefaultControl(orgID, buID pulid.ID) *Control {
	return &Control{
		OrganizationID:      orgID,
		BusinessUnitID:      buID,
		Enabled:             true,
		RunHourLocal:        DefaultRunHourLocal,
		MaxCasesPerAgent:    DefaultMaxCasesPerAgent,
		NightlyBudgetUSD:    DefaultNightlyBudgetUSD,
		MonthlyBudgetUSD:    DefaultMonthlyBudgetUSD,
		JudgeEnabled:        false,
		JudgeSampleRate:     DefaultJudgeSampleRate,
		RegressionThreshold: DefaultRegressionThreshold,
		MinCases:            DefaultMinCases,
		ForceRerunDays:      DefaultForceRerunDays,
	}
}

func (c *Control) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&c.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&c.RunHourLocal,
			validation.Min(0).Error("The hour is between 0 and 23"),
			validation.Max(23).Error("The hour is between 0 and 23"),
		),
		validation.Field(&c.Timezone,
			validation.Length(0, MaxTimezoneChars).Error("Timezone is at most 100 characters"),
			validation.By(domainvalidation.ValidateTimezone),
		),
		validation.Field(&c.MaxCasesPerAgent,
			validation.Min(1).Error("Run at least one case per agent"),
			validation.Max(MaxCasesPerAgentLimit).Error("Run at most 500 cases per agent"),
		),
		validation.Field(&c.JudgeSampleRate,
			validation.Min(0.0).Error("The share judged is between 0 and 1"),
			validation.Max(1.0).Error("The share judged is between 0 and 1"),
		),
		validation.Field(&c.RegressionThreshold,
			validation.Min(0.01).Error("The drop that counts as a regression is at least 0.01"),
			validation.Max(1.0).Error("The drop that counts as a regression is at most 1"),
		),
		validation.Field(&c.MinCases,
			validation.Min(1).Error("At least one case is needed to compare scores"),
			validation.Max(MaxMinCases).Error("At most 500 cases can be required"),
		),
		validation.Field(&c.ForceRerunDays,
			validation.Min(1).Error("Rerun an unchanged agent at least every 90 days"),
			validation.Max(MaxForceRerunDays).Error(
				"Rerun an unchanged agent at least every 90 days",
			),
		),
	))

	validateBudget(multiErr, "nightlyBudgetUsd", c.NightlyBudgetUSD)
	validateBudget(multiErr, "monthlyBudgetUsd", c.MonthlyBudgetUSD)
	if c.MonthlyBudgetUSD.LessThan(c.NightlyBudgetUSD) {
		multiErr.Add(
			"monthlyBudgetUsd",
			errortypes.ErrInvalid,
			"The monthly budget cannot be less than one night's budget",
		)
	}
}

var budgetRule = domainvalidation.BudgetUSD(MaxBudgetUSD, "100,000")

func validateBudget(multiErr *errortypes.MultiError, field string, value decimal.Decimal) {
	if err := budgetRule.Validate(value); err != nil {
		multiErr.Add(field, errortypes.ErrInvalid, err.Error())
	}
}

type SuiteSettings struct {
	MaxCasesPerAgent    int             `json:"maxCasesPerAgent"`
	NightlyBudgetUSD    decimal.Decimal `json:"nightlyBudgetUsd"`
	MonthlyBudgetUSD    decimal.Decimal `json:"monthlyBudgetUsd"`
	JudgeEnabled        bool            `json:"judgeEnabled"`
	JudgeSampleRate     float64         `json:"judgeSampleRate"`
	RegressionThreshold float64         `json:"regressionThreshold"`
	MinCases            int             `json:"minCases"`
	ForceRerunDays      int             `json:"forceRerunDays"`
}

func (c *Control) Settings() SuiteSettings {
	return SuiteSettings{
		MaxCasesPerAgent:    c.MaxCasesPerAgent,
		NightlyBudgetUSD:    c.NightlyBudgetUSD,
		MonthlyBudgetUSD:    c.MonthlyBudgetUSD,
		JudgeEnabled:        c.JudgeEnabled,
		JudgeSampleRate:     c.JudgeSampleRate,
		RegressionThreshold: c.RegressionThreshold,
		MinCases:            c.MinCases,
		ForceRerunDays:      c.ForceRerunDays,
	}
}

const (
	suiteRunWorkflowPrefix   = "agent-suite-run:"
	evaluationWorkflowPrefix = "agent-evaluation:"
	sweepScheduleIDPrefix    = "agent-quality/"
)

func SuiteRunWorkflowID(id pulid.ID) string { return suiteRunWorkflowPrefix + id.String() }

func EvaluationWorkflowID(id pulid.ID) string { return evaluationWorkflowPrefix + id.String() }

func SweepScheduleID(orgID pulid.ID) string { return sweepScheduleIDPrefix + orgID.String() }

func SweepScheduleIDPrefix() string { return sweepScheduleIDPrefix }

func (c *Control) EffectiveTimezone(organizationTimezone string) string {
	if c.Timezone != "" {
		return c.Timezone
	}

	return timeutils.NormalizeTimezone(organizationTimezone)
}

func (c *Control) GetID() pulid.ID { return c.ID }

func (c *Control) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *Control) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *Control) GetTableName() string { return "agent_quality_controls" }
