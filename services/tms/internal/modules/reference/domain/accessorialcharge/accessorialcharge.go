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
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*AccessorialCharge)(nil)
	_ domaintypes.PostgresSearchable     = (*AccessorialCharge)(nil)
)

// The organization's accessorial catalog and default pricing
type AccessorialCharge struct {
	keel.Table `keel:"accessorial_charges,report=Billing"`

	ID             pulid.ID           `json:"id"             keel:"id,pk,autoid:acc_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	Code           string             `json:"code"           keel:"code"`
	Description    string             `json:"description"    keel:"description"`
	Status         domaintypes.Status `json:"status"         keel:"status"`
	Method         Method             `json:"method"         keel:"method"`
	RateUnit       RateUnit           `json:"rateUnit"       keel:"rate_unit,nullzero"`
	Amount         decimal.Decimal    `json:"amount"         keel:"amount,label=Default Amount,format=money"`
	SearchVector   string             `json:"-"              keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string             `json:"-"              keel:"rank,computed"`
	Version        int64              `json:"version"        keel:"version,version"`
	CreatedAt      int64              `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt"      keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (a *AccessorialCharge) BeforeUpdate(_ context.Context) error {
	a.Code = NormalizeCode(a.Code)
	if a.Method != MethodPerUnit {
		a.RateUnit = ""
	}

	return nil
}

func (a *AccessorialCharge) BeforeCreate(_ context.Context) error {
	a.Code = NormalizeCode(a.Code)
	if a.Method != MethodPerUnit {
		a.RateUnit = ""
	}
	if a.Status == "" {
		a.Status = domaintypes.StatusActive
	}

	return nil
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

func (a *AccessorialCharge) BeforeSave(_ context.Context) error {
	a.Code = NormalizeCode(a.Code)
	if a.Method != MethodPerUnit {
		a.RateUnit = ""
	}

	return nil
}

func (AccessorialCharge) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_accessorial_charges_code",
			Message: "Accessorial charge with this code already exists in your organization",
		},
	}
}
