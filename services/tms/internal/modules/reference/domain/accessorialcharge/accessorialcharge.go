package accessorialcharge

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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

// The organization's accessorial catalog and default pricing
type AccessorialCharge struct {
	bun.BaseModel `bun:"table:accessorial_charges,alias:acc" json:"-"`
	keel.Table    `                                                   keel:"accessorial_charges,report=Billing"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"                                                      keel:"id,pk,autoid:acc_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"                                        keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"                                         keel:"organization_id,pk,tenant"`
	Code           string             `json:"code"           bun:"code,type:VARCHAR(10),notnull"                                                        keel:"code"`
	Description    string             `json:"description"    bun:"description,type:TEXT,notnull"                                                        keel:"description"`
	Status         domaintypes.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"                                     keel:"status"`
	Method         Method             `json:"method"         bun:"method,type:accessorial_method_enum,notnull"                                          keel:"method"`
	RateUnit       RateUnit           `json:"rateUnit"       bun:"rate_unit,type:rate_unit_enum,nullzero"                                               keel:"rate_unit,nullzero"`
	Amount         decimal.Decimal    `json:"amount"         bun:"amount,type:NUMERIC(19,4),notnull,default:0"                                          keel:"amount,label=Default Amount,format=money"`
	SearchVector   string             `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"                                                 keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string             `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"                                                      keel:"rank,computed"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT"                                                                  keel:"version,version"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

var CodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{2,9}$`)

func NormalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func (a *AccessorialCharge) Validate(multiErr *errortypes.MultiError) {
	a.Code = NormalizeCode(a.Code)
	multiErr.AddOzzoError(validation.ValidateStruct(
		a,
		validation.Field(
			&a.Code,
			validation.Required.Error("Code is required"),
			validation.Match(CodePattern).Error(
				"Code must be 3 to 10 characters: a letter, then letters, digits or underscores",
			),
		),
		validation.Field(&a.Description,
			validation.Required.Error("Description is required"),
		),
		validation.Field(&a.Amount,
			validation.By(func(_ any) error {
				if a.Amount.LessThanOrEqual(decimal.Zero) {
					return errors.New("amount must be greater than zero")
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
	))
}

func (a *AccessorialCharge) GetID() pulid.ID {
	return a.ID
}

func (a *AccessorialCharge) GetCreatedAt() int64 {
	return a.CreatedAt
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
	a.Code = NormalizeCode(a.Code)
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

func (a *AccessorialCharge) BeforeCreate(ctx context.Context) error {
	return a.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}
