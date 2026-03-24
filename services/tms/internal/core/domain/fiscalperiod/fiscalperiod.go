package fiscalperiod

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*FiscalPeriod)(nil)
	_ validationframework.TenantedEntity = (*FiscalPeriod)(nil)
	_ domaintypes.PostgresSearchable     = (*FiscalPeriod)(nil)
)

// ---------------------------------------------------------------
// Fiscal Period
// ---------------------------------------------------------------

type FiscalPeriod struct {
	bun.BaseModel `bun:"table:fiscal_periods,alias:fp" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	FiscalYearID   pulid.ID `json:"fiscalYearId"   bun:"fiscal_year_id,type:VARCHAR(100),notnull"`

	// ---------------------------------------------------------------
	// Core Identity
	// ---------------------------------------------------------------

	// PeriodNumber is the ordinal position within the fiscal year (1–12 for
	// monthly, 1–4 for quarterly, 1–13 for weekly/4-4-5, 13+ for adjusting).
	// Adjusting periods (Period 13/14) use numbers above the regular count.
	PeriodNumber int `json:"periodNumber" bun:"period_number,type:INT,notnull"`

	// PeriodType indicates what kind of period this is. Most carriers use
	// Month. Week is for 4-4-5 calendars. Adjusting is for Period 13
	// year-end adjustment periods.
	PeriodType PeriodType `json:"periodType" bun:"period_type,type:period_type_enum,notnull,default:'Month'"`

	// Status tracks the period's position in its lifecycle.
	// Inactive → Open → Locked → Closed → PermanentlyClosed.
	Status Status `json:"status" bun:"status,type:period_status_enum,notnull,default:'Inactive'"`

	// Name is the display name (e.g. "January 2026", "P1-2026", "Q1 2026",
	// "Adjusting Period 2026").
	Name string `json:"name" bun:"name,type:VARCHAR(100),notnull"`

	// StartDate is the first day of the period as a Unix epoch timestamp.
	// For adjusting periods, this matches the start date of the final
	// operating period — they overlap intentionally.
	StartDate int64 `json:"startDate" bun:"start_date,type:BIGINT,notnull"`

	// EndDate is the last day of the period as a Unix epoch timestamp.
	// For adjusting periods, this matches the end date of the final
	// operating period.
	EndDate int64 `json:"endDate" bun:"end_date,type:BIGINT,notnull"`

	// ---------------------------------------------------------------
	// Period Controls
	// ---------------------------------------------------------------

	// IsAdjusting flags this period as an adjusting period (Period 13/14).
	// Adjusting periods overlap the date range of the last operating period.
	// When running reports, you typically want to include or exclude adjusting
	// periods — this flag makes that filter trivial.
	//
	// Oracle's gl_period_statuses table has an adjustment_period_flag column
	// that serves exactly this purpose. D365 identifies Period 13 as a
	// "Closing" transaction type.
	IsAdjusting bool `json:"isAdjusting" bun:"is_adjusting,type:BOOLEAN,default:false"`

	// AllowAdjustingEntries controls whether adjusting journal entries can
	// be posted to this specific period. During year-end close, the controller
	// enables this on Period 12 and/or the adjusting period while keeping it
	// disabled on periods 1–11.
	//
	// This field is subordinate to the fiscal year's AllowAdjustingEntries
	// master switch. If the year-level flag is false, this field is ignored
	// and no adjusting entries can be posted regardless.
	//
	// The workflow:
	// 1. Controller locks months 1–11 throughout the year as they close
	// 2. At year-end, CFO enables AllowAdjustingEntries on the fiscal year
	// 3. Controller enables AllowAdjustingEntries on Period 12 (and/or Period 13)
	// 4. Auditors post their adjustments to the designated period(s)
	// 5. Once audit is complete, periods are closed and eventually permanently closed
	AllowAdjustingEntries bool `json:"allowAdjustingEntries" bun:"allow_adjusting_entries,type:BOOLEAN,default:false"`

	// AdjustmentDeadline is the cutoff timestamp for posting adjusting entries
	// to this period. After this deadline, adjusting entries are rejected even
	// if the period is still Open or Locked and AllowAdjustingEntries is true.
	//
	// This gives the controller fine-grained control: "you have until February 15th
	// to get your December adjustments in." After that, they close the period
	// without having to chase people down.
	AdjustmentDeadline *int64 `json:"adjustmentDeadline" bun:"adjustment_deadline,type:BIGINT,nullzero"`

	// ---------------------------------------------------------------
	// Close Tracking
	// ---------------------------------------------------------------

	// LockedAt records when the period was soft-closed (moved to Locked status).
	// This is the timestamp when subledger postings were cut off but manual
	// JEs were still allowed.
	LockedAt   *int64   `json:"lockedAt"   bun:"locked_at,type:BIGINT,nullzero"`
	LockedByID pulid.ID `json:"lockedById" bun:"locked_by_id,type:VARCHAR(100),nullzero"`

	// ClosedAt records when the period was hard-closed (moved to Closed status).
	// This is the timestamp when all posting was blocked.
	ClosedAt   *int64   `json:"closedAt"   bun:"closed_at,type:BIGINT,nullzero"`
	ClosedByID pulid.ID `json:"closedById" bun:"closed_by_id,type:VARCHAR(100),nullzero"`

	// ---------------------------------------------------------------
	// Reopen Tracking
	// ---------------------------------------------------------------
	// Same rationale as the fiscal year reopen tracking. Auditors will ask
	// "why was October reopened?" and this gives them the answer without
	// having to dig through system logs.
	// ---------------------------------------------------------------

	ReopenedAt   *int64   `json:"reopenedAt"   bun:"reopened_at,type:BIGINT,nullzero"`
	ReopenedByID pulid.ID `json:"reopenedById" bun:"reopened_by_id,type:VARCHAR(100),nullzero"`
	ReopenReason string   `json:"reopenReason" bun:"reopen_reason,type:TEXT,nullzero"`

	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	ClosedBy     *tenant.User         `json:"closedBy,omitempty"     bun:"rel:belongs-to,join:closed_by_id=id"`
	LockedBy     *tenant.User         `json:"lockedBy,omitempty"     bun:"rel:belongs-to,join:locked_by_id=id"`
	ReopenedBy   *tenant.User         `json:"reopenedBy,omitempty"   bun:"rel:belongs-to,join:reopened_by_id=id"`
}

func (fp *FiscalPeriod) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(fp,
		validation.Field(&fp.FiscalYearID,
			validation.Required.Error("Fiscal year is required"),
		),
		validation.Field(&fp.PeriodNumber,
			validation.Required.Error("Period number is required"),
			validation.Min(1).Error("Period number must be at least 1"),
			validation.Max(12).Error("Period number must be at most 12"),
		),
		validation.Field(&fp.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&fp.PeriodType,
			validation.Required.Error("Period type is required"),
			validation.In(
				PeriodTypeMonth,
				PeriodTypeQuarter,
				PeriodTypeWeek,
				PeriodTypeAdjusting,
			).Error("Period type must be Month, Quarter, Week, or Adjusting"),
		),
		validation.Field(&fp.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				StatusInactive,
				StatusOpen,
				StatusClosed,
				StatusLocked,
				StatusPermanentlyClosed,
			).Error("Status must be Inactive, Open, Closed, Locked, or PermanentlyClosed"),
		),
		validation.Field(&fp.StartDate,
			validation.Required.Error("Start date is required"),
		),
		validation.Field(&fp.EndDate,
			validation.Required.Error("End date is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (fp *FiscalPeriod) GetID() pulid.ID {
	return fp.ID
}

func (fp *FiscalPeriod) GetTableName() string {
	return "fiscal_periods"
}

func (fp *FiscalPeriod) GetOrganizationID() pulid.ID {
	return fp.OrganizationID
}

func (fp *FiscalPeriod) GetBusinessUnitID() pulid.ID {
	return fp.BusinessUnitID
}

func (fp *FiscalPeriod) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "fp",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "period_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
		},
	}
}

func (fp *FiscalPeriod) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if fp.ID.IsNil() {
			fp.ID = pulid.MustNew("fp_")
		}

		fp.CreatedAt = now
	case *bun.UpdateQuery:
		fp.UpdatedAt = now
	}

	return nil
}
