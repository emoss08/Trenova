package customerledger

import (
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

type MonthlyBalance struct {
	keel.Table `keel:"customer_ledger_monthly_balances"`

	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	CustomerID     pulid.ID `json:"customerId"     keel:"customer_id,pk"`
	MonthStart     int64    `json:"monthStart"     keel:"month_start,pk"`
	AmountMinor    int64    `json:"amountMinor"    keel:"amount_minor"`
}

func MonthStart(at int64) int64 {
	t := time.Unix(at, 0).UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).Unix()
}
