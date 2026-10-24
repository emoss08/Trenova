package fiscalyear

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*FiscalYear)(nil)
	_ domaintypes.PostgresSearchable     = (*FiscalYear)(nil)
)

// Accounting years and their close state
type FiscalYear struct {
	keel.Table `                                           keel:"fiscal_years,report=Accounting"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:fyr_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	// ---------------------------------------------------------------
	// Core Identity
	// ---------------------------------------------------------------

	Status Status `json:"status" keel:"status"`

	Year int `json:"year" keel:"year,dimension"`

	Name string `json:"name" keel:"name,label=Fiscal Year"`

	Description string `json:"description" keel:"description,nullzero"`

	StartDate int64 `json:"startDate" keel:"start_date,label=Starts"`

	EndDate int64 `json:"endDate" keel:"end_date,label=Ends"`

	// ---------------------------------------------------------------
	// Calendar Flags
	// ---------------------------------------------------------------

	IsCurrent bool `json:"isCurrent" keel:"is_current,label=Current"`

	IsCalendarYear bool `json:"isCalendarYear" keel:"is_calendar_year,label=Calendar Year"`

	// ---------------------------------------------------------------
	// Year-End Controls
	// ---------------------------------------------------------------

	AllowAdjustingEntries bool `json:"allowAdjustingEntries" keel:"allow_adjusting_entries,label=Allows Adjusting Entries"`

	// ---------------------------------------------------------------
	// Close Tracking
	// ---------------------------------------------------------------
	// These fields create an audit trail for who closed/locked the year and when.
	// The distinction between Locked and Closed matters:
	// - LockedAt = soft close, no new subledger postings but JEs still allowed
	// - ClosedAt = hard close, nothing gets in
	// ---------------------------------------------------------------

	ClosedAt   *int64   `json:"closedAt" keel:"closed_at"`
	ClosedByID pulid.ID `json:"closedById" keel:"closed_by_id,nullzero"`
	LockedAt   *int64   `json:"lockedAt" keel:"locked_at"`
	LockedByID pulid.ID `json:"lockedById" keel:"locked_by_id,nullzero"`

	// ---------------------------------------------------------------
	// Reopen Tracking
	// ---------------------------------------------------------------
	// Every major ERP (Oracle, D365, NetSuite, Business Central) allows
	// reopening a closed fiscal year for material corrections. NetSuite
	// requires a justification reason. We track all three: when, who, and why.
	// These fields are cleared when the year is re-closed.
	// ---------------------------------------------------------------

	ReopenedAt   *int64   `json:"reopenedAt" keel:"reopened_at"`
	ReopenedByID pulid.ID `json:"reopenedById" keel:"reopened_by_id,nullzero"`
	ReopenReason string   `json:"reopenReason" keel:"reopen_reason,nullzero"`

	// ---------------------------------------------------------------
	// System Fields
	// ---------------------------------------------------------------

	Version      int64  `json:"version" keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector string `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-" keel:"rank,computed"`

	// ---------------------------------------------------------------
	// Relationships
	// ---------------------------------------------------------------

	BusinessUnit *tenant.BusinessUnit         `json:"businessUnit,omitempty"`
	Organization *tenant.Organization         `json:"organization,omitempty"`
	ClosedBy     *tenant.User                 `json:"closedBy,omitempty" keel:"belongs:closed_by_id,noreport"`
	LockedBy     *tenant.User                 `json:"lockedBy,omitempty" keel:"belongs:locked_by_id,noreport"`
	ReopenedBy   *tenant.User                 `json:"reopenedBy,omitempty" keel:"belongs:reopened_by_id,noreport"`
	Periods      []*fiscalperiod.FiscalPeriod `json:"periods,omitempty" keel:"many:fiscal_year_id,label=Fiscal Periods"`
}

func (fy *FiscalYear) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(fy,
		validation.Field(&fy.Year,
			validation.Required.Error("Year is required"),
			validation.Min(1900).Error("Year must be at least 1900"),
			validation.Max(2100).Error("Year must be at most 2100"),
		),
		validation.Field(&fy.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&fy.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				StatusDraft,
				StatusOpen,
				StatusClosed,
				StatusPermanentlyClosed,
			).Error("Status must be Draft, Open, Closed, or PermanentlyClosed"),
		),
		validation.Field(&fy.StartDate,
			validation.Required.Error("Start date is required"),
		),
		validation.Field(&fy.EndDate,
			validation.Required.Error("End date is required"),
		),
	))
}

func (fy *FiscalYear) GetID() pulid.ID {
	return fy.ID
}

func (fy *FiscalYear) GetCreatedAt() int64 {
	return fy.CreatedAt
}

func (fy *FiscalYear) GetTableName() string {
	return "fiscal_years"
}

func (fy *FiscalYear) GetOrganizationID() pulid.ID {
	return fy.OrganizationID
}

func (fy *FiscalYear) GetBusinessUnitID() pulid.ID {
	return fy.BusinessUnitID
}

func (fy *FiscalYear) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "fy",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "year",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (fy *FiscalYear) GenerateMonthlyPeriods() []*fiscalperiod.FiscalPeriod {
	startTime := time.Unix(fy.StartDate, 0).UTC()
	endTime := time.Unix(fy.EndDate, 0).UTC()

	periods := make([]*fiscalperiod.FiscalPeriod, 0, 12)
	currentStart := startTime

	for i := 1; currentStart.Before(endTime); i++ {
		nextMonthStart := currentStart.AddDate(0, 1, 0)
		periodEnd := nextMonthStart.Add(-time.Second)

		if periodEnd.After(endTime) || periodEnd.Equal(endTime) {
			periodEnd = endTime
		}

		periods = append(periods, &fiscalperiod.FiscalPeriod{
			FiscalYearID:   fy.ID,
			OrganizationID: fy.OrganizationID,
			BusinessUnitID: fy.BusinessUnitID,
			PeriodNumber:   i,
			PeriodType:     fiscalperiod.PeriodTypeMonth,
			Name:           fmt.Sprintf("Period %d - %s", i, currentStart.Format("January 2006")),
			StartDate:      currentStart.Unix(),
			EndDate:        periodEnd.Unix(),
			Status:         fiscalperiod.StatusOpen,
		})

		currentStart = periodEnd.Add(time.Second)
	}

	return appendAdjustingPeriod(periods, fy)
}

// appendAdjustingPeriod adds the year-end adjusting period (Period 13) that
// holds closing entries and auditor adjustments. It shares the final operating
// period's date range on purpose — that is what keeps December showing December
// once the close posts — and starts Inactive so nothing lands in it until a
// controller opens it or the year-end close seals it.
func appendAdjustingPeriod(
	periods []*fiscalperiod.FiscalPeriod,
	fy *FiscalYear,
) []*fiscalperiod.FiscalPeriod {
	if len(periods) == 0 {
		return periods
	}

	last := periods[len(periods)-1]
	number := last.PeriodNumber + 1
	if number > fiscalperiod.MaxAdjustingPeriodNumber {
		return periods
	}

	return append(periods, &fiscalperiod.FiscalPeriod{
		FiscalYearID:          fy.ID,
		OrganizationID:        fy.OrganizationID,
		BusinessUnitID:        fy.BusinessUnitID,
		PeriodNumber:          number,
		PeriodType:            fiscalperiod.PeriodTypeAdjusting,
		Name:                  fmt.Sprintf("Adjusting Period - %s", fy.Name),
		StartDate:             last.StartDate,
		EndDate:               last.EndDate,
		Status:                fiscalperiod.StatusInactive,
		IsAdjusting:           true,
		AllowAdjustingEntries: true,
	})
}

func (fy *FiscalYear) BeforeCreate(ctx context.Context) error {
	if fy.Status == "" {
		fy.Status = StatusDraft
	}

	return nil
}
