package accessorialcharge

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*AccessorialCharge)(nil)
	_ domain.Validatable        = (*AccessorialCharge)(nil)
	_ infra.PostgresSearchable  = (*AccessorialCharge)(nil)
)

type AccessorialCharge struct {
	bun.BaseModel `bun:"table:accessorial_charges,alias:acc" json:"-"`

	ID             pulid.ID        `json:"id" bun:",pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	Status         domain.Status   `json:"status" bun:"status,type:status_enum,notnull,default:'Active'"`
	Code           string          `json:"code" bun:"code,type:VARCHAR(10),notnull"`
	Description    string          `json:"description" bun:"description,type:TEXT,notnull"`
	SearchVector   string          `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string          `json:"-" bun:"rank,type:VARCHAR(100),scanonly"`
	Method         Method          `json:"method" bun:"method,type:accessorial_method_enum,notnull"`
	Amount         decimal.Decimal `json:"amount" bun:"amount,type:NUMERIC(19,4),notnull"`
	Unit           int16           `json:"unit" bun:"unit,type:INTEGER,notnull"`
	Version        int64           `json:"version" bun:"version,type:BIGINT"`
	CreatedAt      int64           `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64           `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (a *AccessorialCharge) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, a,
		// * Ensure code is populated and is the proper length
		validation.Field(&a.Code,
			validation.Required.Error("Code is required"),
			validation.Length(3, 10).Error("Code must be between 3 and 10 characters"),
		),

		// * Ensure description is populated
		validation.Field(&a.Description,
			validation.Required.Error("Description is required"),
		),

		// * Ensure unit is greater than or equal to 1
		validation.Field(&a.Unit,
			validation.Min(1).Error("Unit must be greater than or equal to 1"),
		),

		// * Ensure method is populated and is valid
		validation.Field(&a.Method,
			validation.Required.Error("Method is required"),
			validation.In(MethodFlat, MethodDistance, MethodPercentage).Error("Invalid method"),
		),

		// * Ensure amount is populated and is greater than 0
		validation.Field(a.Amount.IntPart,
			validation.Required.Error("Amount is required"),
			validation.Min(1).Error("Amount must be greater than 1"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// Pagination Configuration
func (a *AccessorialCharge) GetID() string {
	return a.ID.String()
}

func (a *AccessorialCharge) GetTableName() string {
	return "accessorial_charges"
}

func (a *AccessorialCharge) BeforeAppendModel(_ context.Context, query bun.Query) error {
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

func (a *AccessorialCharge) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "acc",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "code",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "description",
				Weight: "B",
				Type:   infra.PostgresSearchTypeText,
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
	}
}
