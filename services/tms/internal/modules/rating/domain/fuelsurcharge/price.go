package fuelsurcharge

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*FuelIndexPrice)(nil)
	_ validationframework.TenantedEntity = (*FuelIndexPrice)(nil)
)

const PriceDateLayout = "2006-01-02"

type FuelIndexPrice struct {
	bun.BaseModel `bun:"table:fuel_index_prices,alias:fip" json:"-"`
	keel.Table    `keel:"fuel_index_prices"`

	ID             pulid.ID        `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:fip_"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	FuelIndexID    pulid.ID        `json:"fuelIndexId"    bun:"fuel_index_id,type:VARCHAR(100),notnull" keel:"fuel_index_id"`
	PriceDate      string          `json:"priceDate"      bun:"price_date,type:DATE,notnull" keel:"price_date"`
	Price          decimal.Decimal `json:"price"          bun:"price,type:NUMERIC(19,4),notnull" keel:"price"`
	Currency       string          `json:"currency"       bun:"currency,type:VARCHAR(3),notnull,default:'USD'" keel:"currency"`
	IsManual       bool            `json:"isManual"       bun:"is_manual,type:BOOLEAN,notnull" keel:"is_manual"`
	EnteredByID    *pulid.ID       `json:"enteredById"    bun:"entered_by_id,type:VARCHAR(100),nullzero" keel:"entered_by_id"`
	SourceRaw      string          `json:"sourceRaw"      bun:"source_raw,type:VARCHAR(64),nullzero" keel:"source_raw,nullzero"`
	FetchedAt      time.Time       `json:"fetchedAt"      bun:"fetched_at,type:TIMESTAMPTZ,notnull,default:current_timestamp" keel:"fetched_at"`

	FuelIndex    *FuelIndex           `json:"fuelIndex,omitempty" bun:"rel:belongs-to,join:fuel_index_id=id" keel:"belongs:fuel_index_id"`
	BusinessUnit *tenant.BusinessUnit `json:"-"                   bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-"                   bun:"rel:belongs-to,join:organization_id=id"`
}

func (fp *FuelIndexPrice) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(fp,
		validation.Field(&fp.FuelIndexID,
			validation.Required.Error("Fuel index is required"),
		),
		validation.Field(&fp.PriceDate,
			validation.Required.Error("Price date is required"),
			validation.Date(PriceDateLayout).Error("Price date must be a valid date (YYYY-MM-DD)"),
		),
		validation.Field(&fp.Price,
			validation.Required.Error("Price is required"),
			validation.By(validatePositivePrice),
		),
		validation.Field(&fp.Currency,
			validation.Required.Error("Currency is required"),
			validation.Length(3, 3).Error("Currency must be 3 characters"),
		),
	))
}

func validatePositivePrice(value any) error {
	price, ok := value.(decimal.Decimal)
	if !ok {
		return nil
	}
	if price.LessThanOrEqual(decimal.Zero) {
		return errors.New("must be greater than zero")
	}
	return nil
}

func (fp *FuelIndexPrice) GetID() pulid.ID {
	return fp.ID
}

func (fp *FuelIndexPrice) GetOrganizationID() pulid.ID {
	return fp.OrganizationID
}

func (fp *FuelIndexPrice) GetBusinessUnitID() pulid.ID {
	return fp.BusinessUnitID
}

func (fp *FuelIndexPrice) GetTableName() string {
	return "fuel_index_prices"
}

func (fp *FuelIndexPrice) ParsedPriceDate() (time.Time, error) {
	return time.Parse(PriceDateLayout, fp.PriceDate)
}

func (fp *FuelIndexPrice) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := time.Now().UTC()

	switch query.(type) {
	case *bun.InsertQuery:
		if fp.ID.IsNil() {
			fp.ID = pulid.MustNew("fip_")
		}
		if fp.FetchedAt.IsZero() {
			fp.FetchedAt = now
		}
	case *bun.UpdateQuery:
		fp.FetchedAt = now
	}

	return nil
}

func (fp *FuelIndexPrice) BeforeCreate(ctx context.Context) error {
	return fp.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}
