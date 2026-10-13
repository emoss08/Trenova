// Package customerstatement is a customer's statement of account: what it
// was billed and paid over a period and what it still owes, sent to it as a
// PDF by hand or each month for the month just closed.
package customerstatement

import (
	"context"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

// MaxMonthlyDay is the latest day of the month statements can be scheduled
// for, so the day exists in every month.
const MaxMonthlyDay = 28

var _ bun.BeforeAppendModelHook = (*Statement)(nil)

// Statement records one statement emailed to a customer: the period it
// covered, what it said was owed, and who it went to.
type Statement struct {
	bun.BaseModel `bun:"table:customer_statements,alias:cst" json:"-"`

	ID                 pulid.ID `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	CustomerID         pulid.ID `json:"customerId"         bun:"customer_id,type:VARCHAR(100),notnull"`
	PeriodStart        int64    `json:"periodStart"        bun:"period_start,type:BIGINT,notnull"`
	PeriodEnd          int64    `json:"periodEnd"          bun:"period_end,type:BIGINT,notnull"`
	EndingBalanceMinor int64    `json:"endingBalanceMinor" bun:"ending_balance_minor,type:BIGINT,notnull"`
	PastDueMinor       int64    `json:"pastDueMinor"       bun:"past_due_minor,type:BIGINT,notnull"`
	CurrencyCode       string   `json:"currencyCode"       bun:"currency_code,type:VARCHAR(3),notnull"`
	SentTo             string   `json:"sentTo"             bun:"sent_to,type:TEXT,notnull"`
	Monthly            bool     `json:"monthly"            bun:"monthly,type:BOOLEAN,notnull"`
	SentByID           pulid.ID `json:"sentById"           bun:"sent_by_id,type:VARCHAR(100),nullzero"`
	SentAt             int64    `json:"sentAt"             bun:"sent_at,type:BIGINT,notnull"`
}

func (s *Statement) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("cst_")
		}
		if s.SentAt == 0 {
			s.SentAt = timeutils.NowUnix()
		}
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
