package report

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*ReportSchedule)(nil)
	_ validationframework.TenantedEntity = (*ReportSchedule)(nil)
)

const (
	MaxScheduleEmailRecipients = 20
	MaxScheduleNotifyUsers     = 20
)

type ScheduleDelivery struct {
	EmailRecipients []string `json:"emailRecipients,omitempty"`
	EmailAttach     bool     `json:"emailAttach,omitempty"`
	// EmailInline renders the first rows of the result as a table in the email
	// body, so a recipient reads the answer without opening an attachment.
	EmailInline   bool       `json:"emailInline,omitempty"`
	NotifyUserIDs []pulid.ID `json:"notifyUserIds,omitempty"`
}

func (d *ScheduleDelivery) WantsInlineDigest() bool {
	return d != nil && d.EmailInline && len(d.EmailRecipients) > 0
}

func (d *ScheduleDelivery) HasEmail() bool {
	return d != nil && len(d.EmailRecipients) > 0
}

func (d *ScheduleDelivery) HasNotify() bool {
	return d != nil && len(d.NotifyUserIDs) > 0
}

func (d *ScheduleDelivery) Validate(multiErr *errortypes.MultiError) {
	if d == nil {
		return
	}
	if len(d.EmailRecipients) > MaxScheduleEmailRecipients {
		multiErr.Add("emailRecipients", errortypes.ErrInvalid, fmt.Sprintf(
			"A schedule supports at most %d email recipients", MaxScheduleEmailRecipients,
		))
	}
	for i, recipient := range d.EmailRecipients {
		if err := is.EmailFormat.Validate(recipient); err != nil {
			multiErr.Add(
				fmt.Sprintf("emailRecipients[%d]", i),
				errortypes.ErrInvalid,
				fmt.Sprintf("%q is not a valid email address", recipient),
			)
		}
	}
	if len(d.NotifyUserIDs) > MaxScheduleNotifyUsers {
		multiErr.Add("notifyUserIds", errortypes.ErrInvalid, fmt.Sprintf(
			"A schedule supports at most %d in-app recipients", MaxScheduleNotifyUsers,
		))
	}
	for i, userID := range d.NotifyUserIDs {
		if userID.IsNil() {
			multiErr.Add(
				fmt.Sprintf("notifyUserIds[%d]", i),
				errortypes.ErrInvalid,
				"In-app recipient is invalid",
			)
		}
	}
}

// ScheduleAlert turns a schedule from "send this every morning" into "tell me
// only when something needs attention". The condition is evaluated against the
// number of rows the run returned, which is what an exception report is: a
// query that should normally come back empty, and matters exactly when it does
// not.
//
// It is deliberately not a threshold on a measure — that needs the row values,
// which the run pipeline streams to storage without retaining. Measure
// thresholds are a separate piece of work.
type ScheduleAlert struct {
	Operator dbtype.Operator `json:"operator"`
	// Threshold is compared against the run's row count.
	Threshold int64 `json:"threshold"`
	// ColumnID switches the alert from counting rows to reading one measure's
	// grand total. "Any late shipments at all" is a row count; "revenue fell
	// below target" is a number no row count can express.
	ColumnID string `json:"columnId,omitempty"`
	// Value is the threshold for a measure alert. It is separate from
	// Threshold because a total is rarely a whole number — a row count always
	// is, and rounding a currency comparison would fire on the wrong side.
	Value *float64 `json:"value,omitempty"`
	// SuppressWhileFiring stops a daily exception report from mailing the same
	// alert every morning until someone fixes the underlying problem. The next
	// delivery waits until the condition clears and trips again.
	SuppressWhileFiring bool `json:"suppressWhileFiring,omitempty"`
}

// TargetsMeasure reports whether the alert reads a measure total rather than
// the row count, which is what decides if the run has to capture one.
func (a *ScheduleAlert) TargetsMeasure() bool {
	return a != nil && a.ColumnID != ""
}

func (a *ScheduleAlert) ValueOrZero() float64 {
	if a == nil || a.Value == nil {
		return 0
	}
	return *a.Value
}

// MatchesValue compares a measure total against the alert threshold. A missing
// total is not zero — the measure simply did not appear — so it never fires.
func (a *ScheduleAlert) MatchesValue(total *float64) bool {
	if a == nil {
		return true
	}
	if total == nil {
		return false
	}
	return a.compare(*total, a.ValueOrZero())
}

// Matches reports whether a run's row count satisfies the alert condition.
func (a *ScheduleAlert) Matches(rowCount int64) bool {
	if a == nil {
		return true
	}

	return a.compare(float64(rowCount), float64(a.Threshold))
}

// compare is the single comparison both alert kinds run through, so a row
// count and a measure total can never disagree about what ">=" means.
func (a *ScheduleAlert) compare(actual, threshold float64) bool {
	//nolint:exhaustive // non-comparison operators are rejected by Validate
	switch a.Operator {
	case dbtype.OpGreaterThan:
		return actual > threshold
	case dbtype.OpGreaterThanOrEqual:
		return actual >= threshold
	case dbtype.OpLessThan:
		return actual < threshold
	case dbtype.OpLessThanOrEqual:
		return actual <= threshold
	case dbtype.OpEqual:
		return actual == threshold
	case dbtype.OpNotEqual:
		return actual != threshold
	default:
		return false
	}
}

func (a *ScheduleAlert) Validate(multiErr *errortypes.MultiError) {
	if a == nil {
		return
	}
	if !alertOperatorIsValid(a.Operator) {
		multiErr.Add("alert.operator", errortypes.ErrInvalid,
			"An alert compares the row count with one of: >, >=, <, <=, =, !=")
	}
	if a.TargetsMeasure() {
		if a.Value == nil {
			multiErr.Add("alert.value", errortypes.ErrRequired,
				"A measure alert needs a threshold value")
		} else if !floatutils.IsFinite(*a.Value) {
			multiErr.Add("alert.value", errortypes.ErrInvalid,
				"An alert threshold must be a finite number")
		}
		return
	}
	if a.Threshold < 0 {
		multiErr.Add("alert.threshold", errortypes.ErrInvalid,
			"A row-count threshold cannot be negative")
	}
}

func alertOperatorIsValid(op dbtype.Operator) bool {
	//nolint:exhaustive // only comparison operators are legal on a row count
	switch op {
	case dbtype.OpGreaterThan, dbtype.OpGreaterThanOrEqual, dbtype.OpLessThan,
		dbtype.OpLessThanOrEqual, dbtype.OpEqual, dbtype.OpNotEqual:
		return true
	default:
		return false
	}
}

type ReportSchedule struct {
	bun.BaseModel `bun:"table:report_schedules,alias:rsch" json:"-"`

	ID                  pulid.ID          `json:"id"                  bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID      pulid.ID          `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID          `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	DefinitionID        pulid.ID          `json:"definitionId"        bun:"definition_id,type:VARCHAR(100),notnull"`
	CronExpression      string            `json:"cronExpression"      bun:"cron_expression,type:VARCHAR(100),notnull"`
	Timezone            string            `json:"timezone"            bun:"timezone,type:VARCHAR(64),notnull"`
	Formats             []string          `json:"formats"             bun:"formats,type:TEXT[],array,notnull"`
	Delivery            *ScheduleDelivery `json:"delivery"            bun:"delivery,type:JSONB,nullzero"`
	Alert               *ScheduleAlert    `json:"alert"               bun:"alert,type:JSONB,nullzero"`
	AlertFiring         bool              `json:"alertFiring"         bun:"alert_firing,type:BOOLEAN,notnull,default:false"`
	Enabled             bool              `json:"enabled"             bun:"enabled,type:BOOLEAN,notnull,default:true"`
	RunAsID             pulid.ID          `json:"runAsId"             bun:"run_as_id,type:VARCHAR(100),notnull"`
	LastRunID           pulid.ID          `json:"lastRunId"           bun:"last_run_id,type:VARCHAR(100),nullzero"`
	NextRunAt           int64             `json:"nextRunAt"           bun:"next_run_at,type:BIGINT,nullzero"`
	ConsecutiveFailures int               `json:"consecutiveFailures" bun:"consecutive_failures,type:INTEGER,notnull,default:0"`
	Version             int64             `json:"version"             bun:"version,type:BIGINT"`
	CreatedAt           int64             `json:"createdAt"           bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64             `json:"updatedAt"           bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization     *tenant.Organization `json:"organization,omitempty"     bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit     *tenant.BusinessUnit `json:"businessUnit,omitempty"     bun:"rel:belongs-to,join:business_unit_id=id"`
	RunAs            *tenant.User         `json:"runAs,omitempty"            bun:"rel:belongs-to,join:run_as_id=id"`
	ReportDefinition *ReportDefinition    `json:"reportDefinition,omitempty" bun:"rel:belongs-to,join:definition_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (rs *ReportSchedule) Validate(multiErr *errortypes.MultiError) {
	if rs.CronExpression == "" {
		multiErr.Add("cronExpression", errortypes.ErrRequired, "Cron expression is required")
	}
	if rs.Timezone == "" {
		multiErr.Add("timezone", errortypes.ErrRequired, "Timezone is required")
	}
	if len(rs.Formats) == 0 {
		multiErr.Add("formats", errortypes.ErrRequired, "At least one format is required")
	}
	for i, format := range rs.Formats {
		if !Format(format).IsValid() {
			multiErr.Add(fmt.Sprintf("formats[%d]", i), errortypes.ErrInvalid, "Format is invalid")
		}
	}
	if rs.RunAsID.IsNil() {
		multiErr.Add("runAsId", errortypes.ErrRequired, "Run-as user is required")
	}
	rs.Delivery.Validate(multiErr)
	rs.Alert.Validate(multiErr)
}

func (rs *ReportSchedule) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rs.ID.IsNil() {
			rs.ID = pulid.MustNew("rsch_")
		}
		rs.CreatedAt = now
	case *bun.UpdateQuery:
		rs.UpdatedAt = now
	}

	return nil
}

func (rs *ReportSchedule) GetID() pulid.ID { return rs.ID }

func (rs *ReportSchedule) GetOrganizationID() pulid.ID { return rs.OrganizationID }

func (rs *ReportSchedule) GetBusinessUnitID() pulid.ID { return rs.BusinessUnitID }

func (rs *ReportSchedule) GetTableName() string { return "report_schedules" }
