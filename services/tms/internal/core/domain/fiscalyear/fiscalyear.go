package fiscalyear

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
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
	_ bun.BeforeAppendModelHook          = (*FiscalYear)(nil)
	_ validationframework.TenantedEntity = (*FiscalYear)(nil)
	_ domaintypes.PostgresSearchable     = (*FiscalYear)(nil)
)

type FiscalYear struct {
	bun.BaseModel `bun:"table:fiscal_years,alias:fy" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`

	// ---------------------------------------------------------------
	// Core Identity
	// ---------------------------------------------------------------

	// Status tracks the fiscal year's position in its lifecycle.
	// Draft → Open → Closed → PermanentlyClosed.
	// Only Closed can revert to Open (with proper authorization).
	Status Status `json:"status" bun:"status,type:fiscal_year_status_enum,notnull,default:'Draft'"`

	// Year is the fiscal year number (e.g. 2026). For fiscal years that span
	// calendar years (Oct 2025 – Sep 2026), this is typically the year the
	// fiscal year ends in, but the carrier can set it to match their convention.
	Year int `json:"year" bun:"year,type:INT,notnull"`

	// Name is the display name (e.g. "FY2026", "Fiscal Year 2025-2026").
	Name string `json:"name" bun:"name,type:VARCHAR(100),notnull"`

	// Description is optional free-text for notes about this fiscal year.
	Description string `json:"description" bun:"description,type:TEXT,nullzero"`

	// StartDate is the first day of the fiscal year as a Unix epoch timestamp.
	StartDate int64 `json:"startDate" bun:"start_date,type:BIGINT,notnull"`

	// EndDate is the last day of the fiscal year as a Unix epoch timestamp.
	EndDate int64 `json:"endDate" bun:"end_date,type:BIGINT,notnull"`

	// ---------------------------------------------------------------
	// Calendar Flags
	// ---------------------------------------------------------------

	// IsCurrent indicates this is the active fiscal year for day-to-day operations.
	// Only one fiscal year per organization+business_unit can be current at a time.
	// Enforced at the DB level with a partial unique index where is_current = TRUE.
	IsCurrent bool `json:"isCurrent" bun:"is_current,type:BOOLEAN,default:false"`

	// IsCalendarYear is a denormalized flag indicating the fiscal year aligns
	// with Jan 1 – Dec 31. Saves a date comparison on every query that needs
	// to know whether this is a standard or offset fiscal year.
	IsCalendarYear bool `json:"isCalendarYear" bun:"is_calendar_year,type:BOOLEAN,default:false"`

	// ---------------------------------------------------------------
	// Year-End Controls
	// ---------------------------------------------------------------

	// AllowAdjustingEntries is the master switch for whether any period in this
	// fiscal year can accept adjusting journal entries. If false, no period can
	// accept adjusting entries regardless of its own per-period setting. If true,
	// it defers to each period's AllowAdjustingEntries flag. This gives the CFO
	// a single kill-switch to lock down the entire year.
	AllowAdjustingEntries bool `json:"allowAdjustingEntries" bun:"allow_adjusting_entries,type:BOOLEAN,default:false"`

	// ---------------------------------------------------------------
	// Close Tracking
	// ---------------------------------------------------------------
	// These fields create an audit trail for who closed/locked the year and when.
	// The distinction between Locked and Closed matters:
	// - LockedAt = soft close, no new subledger postings but JEs still allowed
	// - ClosedAt = hard close, nothing gets in
	// ---------------------------------------------------------------

	ClosedAt   *int64   `json:"closedAt"   bun:"closed_at,type:BIGINT,nullzero"`
	ClosedByID pulid.ID `json:"closedById" bun:"closed_by_id,type:VARCHAR(100),nullzero"`
	LockedAt   *int64   `json:"lockedAt"   bun:"locked_at,type:BIGINT,nullzero"`
	LockedByID pulid.ID `json:"lockedById" bun:"locked_by_id,type:VARCHAR(100),nullzero"`

	// ---------------------------------------------------------------
	// Reopen Tracking
	// ---------------------------------------------------------------
	// Every major ERP (Oracle, D365, NetSuite, Business Central) allows
	// reopening a closed fiscal year for material corrections. NetSuite
	// requires a justification reason. We track all three: when, who, and why.
	// These fields are cleared when the year is re-closed.
	// ---------------------------------------------------------------

	ReopenedAt   *int64   `json:"reopenedAt"   bun:"reopened_at,type:BIGINT,nullzero"`
	ReopenedByID pulid.ID `json:"reopenedById" bun:"reopened_by_id,type:VARCHAR(100),nullzero"`
	ReopenReason string   `json:"reopenReason" bun:"reopen_reason,type:TEXT,nullzero"`

	// ---------------------------------------------------------------
	// System Fields
	// ---------------------------------------------------------------

	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`

	// ---------------------------------------------------------------
	// Relationships
	// ---------------------------------------------------------------

	BusinessUnit *tenant.BusinessUnit         `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization         `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	ClosedBy     *tenant.User                 `json:"closedBy,omitempty"     bun:"rel:belongs-to,join:closed_by_id=id"`
	LockedBy     *tenant.User                 `json:"lockedBy,omitempty"     bun:"rel:belongs-to,join:locked_by_id=id"`
	ReopenedBy   *tenant.User                 `json:"reopenedBy,omitempty"   bun:"rel:belongs-to,join:reopened_by_id=id"`
	Periods      []*fiscalperiod.FiscalPeriod `json:"periods,omitempty"      bun:"rel:has-many,join:id=fiscal_year_id"`
}

func (fy *FiscalYear) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(fy,
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
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (fy *FiscalYear) GetID() pulid.ID {
	return fy.ID
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

func (fy *FiscalYear) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if fy.ID.IsNil() {
			fy.ID = pulid.MustNew("fyr_")
		}

		fy.CreatedAt = now
	case *bun.UpdateQuery:
		fy.UpdatedAt = now
	}

	return nil
}
