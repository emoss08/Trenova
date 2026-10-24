package fiscalperiod

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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

// Accounting periods, their status, and their close and reopen history
type FiscalPeriod struct {
	bun.BaseModel             `bun:"table:fiscal_periods,alias:fp" json:"-"`
	keel.Table                `                                             keel:"fiscal_periods,report=Accounting"`
	pagination.CursorValueSet `bun:",embed"                        json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"               keel:"id,pk,autoid:fp_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"  keel:"organization_id,pk,tenant"`
	FiscalYearID   pulid.ID `json:"fiscalYearId"   bun:"fiscal_year_id,type:VARCHAR(100),notnull"      keel:"fiscal_year_id"`

	// ---------------------------------------------------------------
	// Core Identity
	// ---------------------------------------------------------------

	PeriodNumber int `json:"periodNumber" bun:"period_number,type:INT,notnull" keel:"period_number,label=Period,dimension"`

	PeriodType PeriodType `json:"periodType" bun:"period_type,type:period_type_enum,notnull,default:'Month'" keel:"period_type"`

	Status Status `json:"status" bun:"status,type:period_status_enum,notnull,default:'Inactive'" keel:"status"`

	Name string `json:"name" bun:"name,type:VARCHAR(100),notnull" keel:"name,label=Period Name"`

	StartDate int64 `json:"startDate" bun:"start_date,type:BIGINT,notnull" keel:"start_date,label=Starts"`

	EndDate int64 `json:"endDate" bun:"end_date,type:BIGINT,notnull" keel:"end_date,label=Ends"`

	// ---------------------------------------------------------------
	// Period Controls
	// ---------------------------------------------------------------

	IsAdjusting bool `json:"isAdjusting" bun:"is_adjusting,type:BOOLEAN" keel:"is_adjusting,label=Adjusting Period"`

	AllowAdjustingEntries bool `json:"allowAdjustingEntries" bun:"allow_adjusting_entries,type:BOOLEAN" keel:"allow_adjusting_entries,label=Allows Adjusting Entries"`

	AdjustmentDeadline *int64 `json:"adjustmentDeadline" bun:"adjustment_deadline,type:BIGINT,nullzero" keel:"adjustment_deadline,type=epoch"`

	// ---------------------------------------------------------------
	// Close Tracking
	// ---------------------------------------------------------------

	LockedAt   *int64   `json:"lockedAt"   bun:"locked_at,type:BIGINT,nullzero"          keel:"locked_at"`
	LockedByID pulid.ID `json:"lockedById" bun:"locked_by_id,type:VARCHAR(100),nullzero" keel:"locked_by_id,nullzero"`

	ClosedAt   *int64   `json:"closedAt"   bun:"closed_at,type:BIGINT,nullzero"          keel:"closed_at"`
	ClosedByID pulid.ID `json:"closedById" bun:"closed_by_id,type:VARCHAR(100),nullzero" keel:"closed_by_id,nullzero"`

	// ---------------------------------------------------------------
	// Reopen Tracking
	// ---------------------------------------------------------------
	// Same rationale as the fiscal year reopen tracking. Auditors will ask
	// "why was October reopened?" and this gives them the answer without
	// having to dig through system logs.
	// ---------------------------------------------------------------

	ReopenedAt   *int64   `json:"reopenedAt"   bun:"reopened_at,type:BIGINT,nullzero"          keel:"reopened_at"`
	ReopenedByID pulid.ID `json:"reopenedById" bun:"reopened_by_id,type:VARCHAR(100),nullzero" keel:"reopened_by_id,nullzero"`
	ReopenReason string   `json:"reopenReason" bun:"reopen_reason,type:TEXT,nullzero"          keel:"reopen_reason,nullzero"`

	Version      int64  `json:"version"   bun:"version,type:BIGINT"                                                      keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"                                     keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"                                          keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	ClosedBy     *tenant.User         `json:"closedBy,omitempty"     bun:"rel:belongs-to,join:closed_by_id=id"     keel:"belongs:closed_by_id,noreport"`
	LockedBy     *tenant.User         `json:"lockedBy,omitempty"     bun:"rel:belongs-to,join:locked_by_id=id"     keel:"belongs:locked_by_id,noreport"`
	ReopenedBy   *tenant.User         `json:"reopenedBy,omitempty"   bun:"rel:belongs-to,join:reopened_by_id=id"   keel:"belongs:reopened_by_id,noreport"`
}

func (fp *FiscalPeriod) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(fp,
		validation.Field(&fp.FiscalYearID,
			validation.Required.Error("Fiscal year is required"),
		),
		validation.Field(&fp.PeriodNumber,
			validation.Required.Error("Period number is required"),
			validation.Min(1).Error("Period number must be at least 1"),
			validation.Max(fp.maxPeriodNumber()).
				Error(fmt.Sprintf("Period number must be at most %d", fp.maxPeriodNumber())),
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
	))
}

func (fp *FiscalPeriod) IsAdjustingPeriod() bool {
	return fp.IsAdjusting || fp.PeriodType == PeriodTypeAdjusting
}

// maxPeriodNumber caps the ordinal a period may take. Operating periods stop at
// 12; adjusting periods sit above them (Period 13/14) and share the final
// operating period's date range, which is why they are excluded from the
// fiscal_periods no-overlap constraint.
func (fp *FiscalPeriod) maxPeriodNumber() int {
	if fp.IsAdjustingPeriod() {
		return MaxAdjustingPeriodNumber
	}

	return MaxOperatingPeriodNumber
}

func (fp *FiscalPeriod) GetID() pulid.ID {
	return fp.ID
}

func (fp *FiscalPeriod) GetCreatedAt() int64 {
	return fp.CreatedAt
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
