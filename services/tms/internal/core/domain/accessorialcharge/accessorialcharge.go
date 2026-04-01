package accessorialcharge

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
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*AccessorialCharge)(nil)
	_ validationframework.TenantedEntity = (*AccessorialCharge)(nil)
	_ domaintypes.PostgresSearchable     = (*AccessorialCharge)(nil)
)

type AccessorialCharge struct {
	bun.BaseModel `bun:"table:accessorial_charges,alias:acc" json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	Code           string             `json:"code"           bun:"code,type:VARCHAR(10),notnull"`
	Description    string             `json:"description"    bun:"description,type:TEXT,notnull"`
	Status         domaintypes.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	Method         Method             `json:"method"         bun:"method,type:accessorial_method_enum,notnull"`
	RateUnit       RateUnit           `json:"rateUnit"       bun:"rate_unit,type:rate_unit_enum,nullzero"`
	Amount         decimal.Decimal    `json:"amount"         bun:"amount,type:NUMERIC(19,4),notnull,default:0"`
	SearchVector   string             `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string             `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relations
	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (a *AccessorialCharge) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		a,
		validation.Field(
			&a.Code,
			validation.Required.Error("Code is required"),
			validation.Length(3, 10).Error("Code must be between 3 and 10 characters"),
		),
		validation.Field(&a.Description,
			validation.Required.Error("Description is required"),
		),
		validation.Field(&a.Amount,
			validation.By(func(_ any) error {
				if a.Amount.LessThanOrEqual(decimal.Zero) {
					return errors.New("Amount must be greater than zero")
				}
				return nil
			}),
		),
		validation.Field(
			&a.RateUnit,
			validation.When(
				a.Method == MethodPerUnit,
				validation.Required.Error("Rate unit is required when method is PerUnit"),
			),
		),
		validation.Field(&a.Method,
			validation.Required.Error("Method is required"),
			validation.In(MethodFlat, MethodPerUnit, MethodPercentage).Error("Invalid method"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (a *AccessorialCharge) GetID() pulid.ID {
	return a.ID
}

func (a *AccessorialCharge) GetOrganizationID() pulid.ID {
	return a.OrganizationID
}

func (a *AccessorialCharge) GetBusinessUnitID() pulid.ID {
	return a.BusinessUnitID
}

func (a *AccessorialCharge) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "acc",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (a *AccessorialCharge) GetTableName() string {
	return "accessorial_charges"
}

func (a *AccessorialCharge) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if a.Method != MethodPerUnit {
		a.RateUnit = ""
	}

	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("acc_")
		}

		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}
