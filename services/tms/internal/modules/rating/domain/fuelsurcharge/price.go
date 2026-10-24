package fuelsurcharge

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/shared/timeutils"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*FuelIndexPrice)(nil)
)

const PriceDateLayout = "2006-01-02"

type FuelIndexPrice struct {
	keel.Table `                                                 keel:"fuel_index_prices"`

	ID             pulid.ID        `json:"id" keel:"id,pk,autoid:fip_"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID        `json:"organizationId" keel:"organization_id,pk,tenant"`
	FuelIndexID    pulid.ID        `json:"fuelIndexId" keel:"fuel_index_id"`
	PriceDate      string          `json:"priceDate" keel:"price_date"`
	Price          decimal.Decimal `json:"price" keel:"price"`
	Currency       string          `json:"currency" keel:"currency"`
	IsManual       bool            `json:"isManual" keel:"is_manual"`
	EnteredByID    *pulid.ID       `json:"enteredById" keel:"entered_by_id"`
	SourceRaw      string          `json:"sourceRaw" keel:"source_raw,nullzero"`
	FetchedAt      time.Time       `json:"fetchedAt" keel:"fetched_at"`

	FuelIndex    *FuelIndex           `json:"fuelIndex,omitempty" keel:"belongs:fuel_index_id"`
	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
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

func (fp *FuelIndexPrice) BeforeCreate(ctx context.Context) error {
	if fp.Currency == "" {
		fp.Currency = "USD"
	}
	if fp.FetchedAt.IsZero() {
		fp.FetchedAt = timeutils.Now()
	}
	now := time.Now().UTC()
	if fp.FetchedAt.IsZero() {
		fp.FetchedAt = now
	}

	return nil
}

func (fp *FuelIndexPrice) BeforeUpdate(_ context.Context) error {
	now := time.Now().UTC()
	fp.FetchedAt = now
	fp.FetchedAt = time.Now().UTC()

	return nil
}
