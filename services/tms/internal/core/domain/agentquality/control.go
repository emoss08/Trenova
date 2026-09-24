package agentquality

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Control)(nil)
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
	bun.BaseModel `bun:"table:agent_quality_controls,alias:aqc" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	Enabled             bool            `json:"enabled"             bun:"enabled,type:BOOLEAN,notnull,default:true"`
	RunHourLocal        int             `json:"runHourLocal"        bun:"run_hour_local,type:SMALLINT,notnull,default:2"`
	Timezone            string          `json:"timezone"            bun:"timezone,type:VARCHAR(100),nullzero"`
	MaxCasesPerAgent    int             `json:"maxCasesPerAgent"    bun:"max_cases_per_agent,type:INTEGER,notnull,default:50"`
	NightlyBudgetUSD    decimal.Decimal `json:"nightlyBudgetUsd"    bun:"nightly_budget_usd,type:NUMERIC(14,2),notnull,default:5.00"`
	MonthlyBudgetUSD    decimal.Decimal `json:"monthlyBudgetUsd"    bun:"monthly_budget_usd,type:NUMERIC(14,2),notnull,default:50.00"`
	JudgeEnabled        bool            `json:"judgeEnabled"        bun:"judge_enabled,type:BOOLEAN,notnull,default:false"`
	JudgeSampleRate     float64         `json:"judgeSampleRate"     bun:"judge_sample_rate,type:DOUBLE PRECISION,notnull,default:0.2"`
	RegressionThreshold float64         `json:"regressionThreshold" bun:"regression_threshold,type:DOUBLE PRECISION,notnull,default:0.10"`
	MinCases            int             `json:"minCases"            bun:"min_cases,type:INTEGER,notnull,default:10"`
	ForceRerunDays      int             `json:"forceRerunDays"      bun:"force_rerun_days,type:INTEGER,notnull,default:7"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
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

func (c *Control) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("aqc_")
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
