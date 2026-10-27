// Package customerstatement is a customer's statement of account: what it
// was billed and paid over a period and what it still owes, sent to it as a
// PDF by hand or each month for the month just closed.
package customerstatement

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// MaxMonthlyDay is the latest day of the month statements can be scheduled
// for, so the day exists in every month.
const MaxMonthlyDay = 28

// Statement records one statement emailed to a customer: the period it
// covered, what it said was owed, and who it went to.
type Statement struct {
	keel.Table `keel:"customer_statements"`

	ID                 pulid.ID `json:"id"                 keel:"id,pk,autoid:cst_"`
	OrganizationID     pulid.ID `json:"organizationId"     keel:"organization_id,pk,tenant"`
	BusinessUnitID     pulid.ID `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	CustomerID         pulid.ID `json:"customerId"         keel:"customer_id"`
	PeriodStart        int64    `json:"periodStart"        keel:"period_start"`
	PeriodEnd          int64    `json:"periodEnd"          keel:"period_end"`
	EndingBalanceMinor int64    `json:"endingBalanceMinor" keel:"ending_balance_minor"`
	PastDueMinor       int64    `json:"pastDueMinor"       keel:"past_due_minor"`
	CurrencyCode       string   `json:"currencyCode"       keel:"currency_code"`
	SentTo             string   `json:"sentTo"             keel:"sent_to"`
	Monthly            bool     `json:"monthly"            keel:"monthly"`
	SentByID           pulid.ID `json:"sentById"           keel:"sent_by_id,nullzero"`
	SentAt             int64    `json:"sentAt"             keel:"sent_at"`
}

func (s *Statement) BeforeCreate(ctx context.Context) error {
	if s.SentAt == 0 {
		s.SentAt = timeutils.NowUnix()
	}

	return nil
}

// Period is the span a statement covers, both ends inclusive, in Unix
// seconds. A zero Start runs from the account's first transaction.
type Period struct {
	Start int64
	End   int64
}

// MonthlyPeriod is the calendar month before now in loc, and whether its
// statement is due: it is once the local day of the month reaches day. A
// day outside 1 to MaxMonthlyDay is treated as the first.
//
// The month is due from day until the month ends, so a sweep that misses a
// day still sends it the next.
func MonthlyPeriod(now int64, loc *time.Location, day int) (Period, bool) {
	if loc == nil {
		loc = time.UTC
	}
	if day < 1 || day > MaxMonthlyDay {
		day = 1
	}
	local := time.Unix(now, 0).In(loc)
	thisMonth := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
	lastMonth := thisMonth.AddDate(0, -1, 0)
	return Period{
		Start: lastMonth.Unix(),
		End:   thisMonth.Unix() - 1,
	}, local.Day() >= day
}
