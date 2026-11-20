// Package customerstatement is a customer's statement of account: what it
// was billed and paid over a period and what it still owes, sent to it as a
// PDF by hand or each month for the month just closed.
package customerstatement

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
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

const (
	// MaxPeriodDays is the longest span one statement covers; the balance
	// before it is carried in as the opening balance.
	MaxPeriodDays = 366
	// MaxTransactions is the most entries one statement lists.
	MaxTransactions = 5000

	secondsPerDay = int64(24 * 60 * 60)
)

// Period is the span a statement covers, both ends inclusive, in Unix
// seconds.
type Period struct {
	Start int64
	End   int64
}

// Resolve checks the span and fills a zero Start with the calendar month
// before End in loc (UTC when nil), clamped to the month's end, so every
// statement reads a bounded window of the ledger on the monthly cycle.
func (p Period) Resolve(loc *time.Location) (Period, error) {
	if p.Start < 0 || p.End < 0 {
		return p, errortypes.NewValidationError(
			"startDate", errortypes.ErrInvalid, "Statement dates cannot be negative",
		)
	}
	if p.Start == 0 {
		if loc == nil {
			loc = time.UTC
		}
		p.Start = max(timeutils.AddMonths(time.Unix(p.End, 0).In(loc), -1).Unix(), 0)
	}
	if p.Start > p.End {
		return p, errortypes.NewValidationError(
			"startDate",
			errortypes.ErrInvalid,
			"Start the statement on or before its statement date",
		)
	}
	if p.End-p.Start > MaxPeriodDays*secondsPerDay {
		return p, errortypes.NewValidationError(
			"startDate",
			errortypes.ErrInvalid,
			"A statement covers at most {0} days; start it later",
			MaxPeriodDays,
		)
	}
	return p, nil
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
