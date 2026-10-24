package fiscalperiod

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*FiscalPeriod)(nil)
	_ domaintypes.PostgresSearchable     = (*FiscalPeriod)(nil)
)

// ---------------------------------------------------------------
// Fiscal Period
// ---------------------------------------------------------------

// Accounting periods, their status, and their close and reopen history
type FiscalPeriod struct {
	keel.Table `                                             keel:"fiscal_periods,report=Accounting"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:fp_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	FiscalYearID   pulid.ID `json:"fiscalYearId" keel:"fiscal_year_id"`

	// ---------------------------------------------------------------
	// Core Identity
	// ---------------------------------------------------------------

	PeriodNumber int `json:"periodNumber" keel:"period_number,label=Period,dimension"`

	PeriodType PeriodType `json:"periodType" keel:"period_type"`

	Status Status `json:"status" keel:"status"`

	Name string `json:"name" keel:"name,label=Period Name"`

	StartDate int64 `json:"startDate" keel:"start_date,label=Starts"`

	EndDate int64 `json:"endDate" keel:"end_date,label=Ends"`

	// ---------------------------------------------------------------
	// Period Controls
	// ---------------------------------------------------------------

	IsAdjusting bool `json:"isAdjusting" keel:"is_adjusting,label=Adjusting Period"`

	AllowAdjustingEntries bool `json:"allowAdjustingEntries" keel:"allow_adjusting_entries,label=Allows Adjusting Entries"`

	AdjustmentDeadline *int64 `json:"adjustmentDeadline" keel:"adjustment_deadline,type=epoch"`

	// ---------------------------------------------------------------
	// Close Tracking
	// ---------------------------------------------------------------

	LockedAt   *int64   `json:"lockedAt" keel:"locked_at"`
	LockedByID pulid.ID `json:"lockedById" keel:"locked_by_id,nullzero"`

	ClosedAt   *int64   `json:"closedAt" keel:"closed_at"`
	ClosedByID pulid.ID `json:"closedById" keel:"closed_by_id,nullzero"`

	// ---------------------------------------------------------------
	// Reopen Tracking
	// ---------------------------------------------------------------
	// Same rationale as the fiscal year reopen tracking. Auditors will ask
	// "why was October reopened?" and this gives them the answer without
	// having to dig through system logs.
	// ---------------------------------------------------------------

	ReopenedAt   *int64   `json:"reopenedAt" keel:"reopened_at"`
	ReopenedByID pulid.ID `json:"reopenedById" keel:"reopened_by_id,nullzero"`
	ReopenReason string   `json:"reopenReason" keel:"reopen_reason,nullzero"`

	Version      int64  `json:"version" keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector string `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-" keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
	ClosedBy     *tenant.User         `json:"closedBy,omitempty" keel:"belongs:closed_by_id,noreport"`
	LockedBy     *tenant.User         `json:"lockedBy,omitempty" keel:"belongs:locked_by_id,noreport"`
	ReopenedBy   *tenant.User         `json:"reopenedBy,omitempty" keel:"belongs:reopened_by_id,noreport"`
}

func (fp *FiscalPeriod) BeforeCreate(_ context.Context) error {
	if fp.PeriodType == "" {
		fp.PeriodType = PeriodTypeMonth
	}
	if fp.Status == "" {
		fp.Status = StatusInactive
	}

	return nil
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
