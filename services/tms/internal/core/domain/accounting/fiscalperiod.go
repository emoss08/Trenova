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
	_ bun.BeforeAppendModelHook      = (*FiscalPeriod)(nil)
	_ domaintypes.PostgresSearchable = (*FiscalPeriod)(nil)
	_ domain.Validatable             = (*FiscalPeriod)(nil)
	_ framework.TenantedEntity       = (*FiscalPeriod)(nil)
)

type FiscalPeriod struct {
	bun.BaseModel `bun:"table:fiscal_periods,alias:fp" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	FiscalYearID   pulid.ID `json:"fiscalYearId"   bun:"fiscal_year_id,type:VARCHAR(100),notnull"`

	PeriodNumber int          `json:"periodNumber" bun:"period_number,type:INTEGER,notnull"`
	PeriodType   PeriodType   `json:"periodType"   bun:"period_type,type:period_type_enum,notnull,default:'Month'"`
	Name         string       `json:"name"         bun:"name,type:VARCHAR(100),notnull"`
	StartDate    int64        `json:"startDate"    bun:"start_date,type:BIGINT,notnull"`
	EndDate      int64        `json:"endDate"      bun:"end_date,type:BIGINT,notnull"`
	Status       PeriodStatus `json:"status"       bun:"status,type:period_status_enum,notnull,default:'Open'"`

	ClosedAt   *int64    `json:"closedAt"   bun:"closed_at,type:BIGINT,nullzero"`
	ClosedByID *pulid.ID `json:"closedById" bun:"closed_by_id,type:VARCHAR(100),nullzero"`

	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	FiscalYear   *FiscalYear          `json:"fiscalYear,omitempty"   bun:"rel:belongs-to,join:fiscal_year_id=id"`
	ClosedBy     *tenant.User         `json:"closedBy,omitempty"     bun:"rel:belongs-to,join:closed_by_id=id"`
}

func (fp *FiscalPeriod) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		fp,
		validation.Field(&fp.FiscalYearID,
			validation.Required.Error("Fiscal year is required"),
		),
		validation.Field(&fp.PeriodNumber,
			validation.Required.Error("Period number is required"),
			validation.Min(1).Error("Period number must be at least 1"),
			validation.Max(12).Error("Period number cannot exceed 12"),
		),
		validation.Field(&fp.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&fp.StartDate,
			validation.Required.Error("Start date is required"),
		),
		validation.Field(&fp.EndDate,
			validation.Required.Error("End date is required"),
		),
		validation.Field(&fp.PeriodType,
			validation.Required.Error("Period type is required"),
			validation.In(
				PeriodTypeMonth,
				PeriodTypeQuarter,
				PeriodTypeYear,
			).Error("Period type must be a valid period type"),
		),
		validation.Field(&fp.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				PeriodStatusOpen,
				PeriodStatusClosed,
				PeriodStatusLocked,
			).Error("Status must be a valid period status"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (fp *FiscalPeriod) GetID() string {
	return fp.ID.String()
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
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "period_number",
				Type:   domaintypes.FieldTypeNumber,
				Weight: domaintypes.SearchWeightA,
			},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (fp *FiscalPeriod) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

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
