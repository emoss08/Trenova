package fuelpurchase

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type TractorDay struct {
	keel.Table `keel:"fuel_tractor_daily"`

	OrganizationID pulid.ID        `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	DayAt          int64           `json:"dayAt"          keel:"day_at,pk"`
	TractorID      pulid.ID        `json:"tractorId"      keel:"tractor_id,pk"`
	Purchases      int             `json:"purchases"      keel:"purchases"`
	Gallons        decimal.Decimal `json:"gallons"        keel:"gallons"`
	AmountMinor    int64           `json:"amountMinor"    keel:"amount_minor"`
	CurrencyCode   string          `json:"currencyCode"   keel:"currency_code,nullzero"`
	Currencies     int             `json:"currencies"     keel:"currencies"`
	Miles          decimal.Decimal `json:"miles"          keel:"miles"`
	RefreshedAt    int64           `json:"refreshedAt"    keel:"refreshed_at"`
}
