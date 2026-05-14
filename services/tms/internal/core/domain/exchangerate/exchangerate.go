package exchangerate

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*ExchangeRate)(nil)
	_ validationframework.TenantedEntity = (*ExchangeRate)(nil)
)

type ExchangeRate struct {
	bun.BaseModel `bun:"table:exchange_rates,alias:er" json:"-"`

	ID             pulid.ID        `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	FromCurrency   string          `json:"fromCurrency"   bun:"from_currency,type:VARCHAR(3),notnull"`
	ToCurrency     string          `json:"toCurrency"     bun:"to_currency,type:VARCHAR(3),notnull"`
	Rate           decimal.Decimal `json:"rate"           bun:"rate,type:NUMERIC(19,6),notnull"`
	Date           string          `json:"date"           bun:"date,type:DATE,notnull"`
	FetchedAt      int64           `json:"fetchedAt"      bun:"fetched_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (e *ExchangeRate) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(e,
		validation.Field(&e.FromCurrency,
			validation.Required.Error("From currency is required"),
			validation.Length(3, 3).Error("From currency must be 3 characters"),
		),
		validation.Field(&e.ToCurrency,
			validation.Required.Error("To currency is required"),
			validation.Length(3, 3).Error("To currency must be 3 characters"),
		),
		validation.Field(&e.Rate,
			validation.Required.Error("Rate is required"),
		),
		validation.Field(&e.Date,
			validation.Required.Error("Date is required"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (e *ExchangeRate) GetID() pulid.ID {
	return e.ID
}

func (e *ExchangeRate) GetTableName() string {
	return "exchange_rates"
}

func (e *ExchangeRate) GetOrganizationID() pulid.ID {
	return e.OrganizationID
}

func (e *ExchangeRate) GetBusinessUnitID() pulid.ID {
	return e.BusinessUnitID
}

func (e *ExchangeRate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("fxr_")
		}
		e.FetchedAt = now
	case *bun.UpdateQuery:
		e.FetchedAt = now
	}

	return nil
}
