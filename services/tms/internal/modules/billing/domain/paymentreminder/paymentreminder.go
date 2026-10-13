// Package paymentreminder is the email a customer receives once its invoices
// go past due, on the days past due its organization schedules.
package paymentreminder

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

// ManualStep marks a reminder a person sent by hand. Scheduled steps count
// from one and are each sent once per invoice; a manual reminder may repeat.
const ManualStep = 0

var _ bun.BeforeAppendModelHook = (*Reminder)(nil)

// Reminder records that one invoice was included in a reminder email: which
// step of the schedule it was, how late and how much was open, and who it
// went to.
type Reminder struct {
	bun.BaseModel `bun:"table:invoice_payment_reminders,alias:ipr" json:"-"`

	ID              pulid.ID `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID  pulid.ID `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	InvoiceID       pulid.ID `json:"invoiceId"       bun:"invoice_id,type:VARCHAR(100),notnull"`
	CustomerID      pulid.ID `json:"customerId"      bun:"customer_id,type:VARCHAR(100),notnull"`
	Step            int16    `json:"step"            bun:"step,type:SMALLINT,notnull"`
	DaysPastDue     int      `json:"daysPastDue"     bun:"days_past_due,type:INTEGER,notnull"`
	OpenAmountMinor int64    `json:"openAmountMinor" bun:"open_amount_minor,type:BIGINT,notnull"`
	CurrencyCode    string   `json:"currencyCode"    bun:"currency_code,type:VARCHAR(3),notnull"`
	SentTo          string   `json:"sentTo"          bun:"sent_to,type:TEXT,notnull"`
	Manual          bool     `json:"manual"          bun:"manual,type:BOOLEAN,notnull"`
	SentByID        pulid.ID `json:"sentById"        bun:"sent_by_id,type:VARCHAR(100),nullzero"`
	SentAt          int64    `json:"sentAt"          bun:"sent_at,type:BIGINT,notnull"`
}

func (r *Reminder) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("ipr_")
		}
		if r.SentAt == 0 {
			r.SentAt = timeutils.NowUnix()
		}
	}
	return nil
}

// StepReached is the latest step of the schedule an invoice this many days
// past due has reached, counting from one; zero before the first.
func StepReached(daysPastDue int, schedule []int32) int {
	reached := 0
	for i, day := range schedule {
		if daysPastDue >= int(day) {
			reached = i + 1
		}
	}
	return reached
}

// IsFinal reports whether the step is the last one the schedule sends.
func IsFinal(step int, schedule []int32) bool {
	return step > 0 && step == len(schedule)
}

// DaysPastDue counts whole days since the due date; zero on or before it.
func DaysPastDue(dueDate, now int64) int {
	if dueDate <= 0 || now <= dueDate {
		return 0
	}
	return int((now - dueDate) / (24 * 60 * 60))
}
