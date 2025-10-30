package accounting

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*FiscalYear)(nil)
	_ domaintypes.PostgresSearchable = (*FiscalYear)(nil)
	_ domain.Validatable             = (*FiscalYear)(nil)
	_ framework.TenantedEntity       = (*FiscalYear)(nil)
)

type FiscalYear struct {
	bun.BaseModel `bun:"table:fiscal_years,alias:fy" json:"-"`

	ID             pulid.ID         `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID         `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID         `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Status         FiscalYearStatus `json:"status"         bun:"status,type:fiscal_year_status_enum,notnull,default:'Draft'"`
	Year           int              `json:"year"           bun:"year,type:INTEGER,notnull"`
	Name           string           `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Description    string           `json:"description"    bun:"description,type:TEXT,nullzero"`
	StartDate      int64            `json:"startDate"      bun:"start_date,type:BIGINT,notnull"`
	EndDate        int64            `json:"endDate"        bun:"end_date,type:BIGINT,notnull"`
	TaxYear        int              `json:"taxYear"        bun:"tax_year,type:INTEGER,nullzero"` // Optional: Only if differs from Year (requires IRS approval)

	// Financial Planning
	BudgetAmount       int64 `json:"budgetAmount"       bun:"budget_amount,type:BIGINT,nullzero,default:0"` // Stored in cents
	AdjustmentDeadline int64 `json:"adjustmentDeadline" bun:"adjustment_deadline,type:BIGINT,nullzero"`     // Date when adjusting entires are no longer accepted

	// Control Flags
	IsCurrent             bool `json:"isCurrent"             bun:"is_current,type:BOOLEAN,notnull,default:false"`
	IsCalendarYear        bool `json:"isCalendarYear"        bun:"is_calendar_year,type:BOOLEAN,notnull,default:false"`
	AllowAdjustingEntries bool `json:"allowAdjustingEntries" bun:"allow_adjusting_entries,type:BOOLEAN,notnull,default:false"`

	ClosedAt   int64     `json:"closedAt"   bun:"closed_at,type:BIGINT,nullzero"`
	LockedAt   int64     `json:"lockedAt"   bun:"locked_at,type:BIGINT,nullzero"`
	ClosedByID *pulid.ID `json:"closedById" bun:"closed_by_id,type:VARCHAR(100),nullzero"`
	LockedByID *pulid.ID `json:"lockedById" bun:"locked_by_id,type:VARCHAR(100),nullzero"`

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
	PriorYear    *FiscalYear          `json:"priorYear,omitempty"    bun:"rel:belongs-to,join:prior_year_id=id"`
}

func (fy *FiscalYear) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		fy,
		validation.Field(&fy.Year, validation.Required.Error("Year is required"),
			validation.Min(1900).Error("Year must be between 1900 and 2100"),
			validation.Max(2100).Error("Year must be between 1900 and 2100"),
		),
		validation.Field(&fy.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(
			&fy.BudgetAmount,
			validation.Min(0).Error("Budget amount cannot be negative"),
		),
		validation.Field(&fy.StartDate, validation.Required.Error("Start date is required")),
		validation.Field(&fy.EndDate, validation.Required.Error("End date is required")),
		validation.Field(&fy.Status, validation.Required.Error("Status is required"),
			validation.In(
				FiscalYearStatusDraft,
				FiscalYearStatusOpen,
				FiscalYearStatusClosed,
				FiscalYearStatusLocked,
			).Error("Status must be a valid fiscal year status"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (fy *FiscalYear) GetID() string {
	return fy.ID.String()
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
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "year", Type: domaintypes.FieldTypeNumber, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (fy *FiscalYear) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

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
