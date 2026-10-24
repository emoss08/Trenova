// Package collection is the work of collecting overdue receivables from a
// customer: who is chasing them, when they next follow up, and the history of
// what was said and promised.
package collection

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

// ActivityKind is what one entry in a customer's collections history records.
type ActivityKind string

const (
	ActivityNote              = ActivityKind("Note")
	ActivityCall              = ActivityKind("Call")
	ActivityEmail             = ActivityKind("Email")
	ActivityPromiseToPay      = ActivityKind("PromiseToPay")
	ActivityCollectorAssigned = ActivityKind("CollectorAssigned")
	ActivityFollowUpScheduled = ActivityKind("FollowUpScheduled")
)

func (k ActivityKind) IsValid() bool {
	switch k {
	case ActivityNote, ActivityCall, ActivityEmail, ActivityPromiseToPay,
		ActivityCollectorAssigned, ActivityFollowUpScheduled:
		return true
	default:
		return false
	}
}

// IsContact reports whether a person logs this kind of entry by hand; the
// rest are written when the account's collector or follow-up changes.
func (k ActivityKind) IsContact() bool {
	switch k { //nolint:exhaustive // entries written by the system are not contacts
	case ActivityNote, ActivityCall, ActivityEmail, ActivityPromiseToPay:
		return true
	default:
		return false
	}
}

// MaxSummaryLength bounds a logged note.
const MaxSummaryLength = 4000

var _ bun.BeforeAppendModelHook = (*Activity)(nil)

// Activity is one entry in a customer's collections history. Entries are
// written once and never changed, so the history is what happened.
type Activity struct {
	bun.BaseModel `bun:"table:collection_activities,alias:cact" json:"-"`
	keel.Table    `keel:"collection_activities"`

	ID                  pulid.ID     `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:cact_"`
	OrganizationID      pulid.ID     `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID      pulid.ID     `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	CustomerID          pulid.ID     `json:"customerId"          bun:"customer_id,type:VARCHAR(100),notnull" keel:"customer_id"`
	Kind                ActivityKind `json:"kind"                bun:"kind,type:VARCHAR(30),notnull" keel:"kind"`
	Summary             string       `json:"summary"             bun:"summary,type:TEXT,nullzero" keel:"summary,nullzero"`
	ContactName         string       `json:"contactName"         bun:"contact_name,type:VARCHAR(200),nullzero" keel:"contact_name,nullzero"`
	InvoiceID           pulid.ID     `json:"invoiceId"           bun:"invoice_id,type:VARCHAR(100),nullzero" keel:"invoice_id,nullzero"`
	PromisedAmountMinor *int64       `json:"promisedAmountMinor" bun:"promised_amount_minor,type:BIGINT,nullzero" keel:"promised_amount_minor"`
	PromisedOn          *int64       `json:"promisedOn"          bun:"promised_on,type:BIGINT,nullzero" keel:"promised_on"`
	CurrencyCode        string       `json:"currencyCode"        bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'" keel:"currency_code"`
	FollowUpOn          *int64       `json:"followUpOn"          bun:"follow_up_on,type:BIGINT,nullzero" keel:"follow_up_on"`
	CollectorID         pulid.ID     `json:"collectorId"         bun:"collector_id,type:VARCHAR(100),nullzero" keel:"collector_id,nullzero"`
	CreatedByID         pulid.ID     `json:"createdById"         bun:"created_by_id,type:VARCHAR(100),notnull" keel:"created_by_id"`
	CreatedAt           int64        `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
}

func (a *Activity) GetID() pulid.ID             { return a.ID }
func (a *Activity) GetOrganizationID() pulid.ID { return a.OrganizationID }
func (a *Activity) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }

// Validate checks an entry a person is logging. A promise needs an amount
// and the date the customer said they would pay.
func (a *Activity) Validate(multiErr *errortypes.MultiError) {
	a.Summary = strings.TrimSpace(a.Summary)
	a.ContactName = strings.TrimSpace(a.ContactName)

	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.CustomerID, validation.Required.Error("Customer is required")),
		validation.Field(&a.Kind,
			validation.Required.Error("Kind is required"),
			validation.By(func(any) error {
				if !a.Kind.IsValid() {
					return validation.NewError("invalid_kind", "Kind is invalid")
				}
				return nil
			}),
		),
		validation.Field(&a.Summary,
			validation.When(a.Kind != ActivityPromiseToPay && a.Kind.IsContact(),
				validation.Required.Error("Say what happened"),
			),
			validation.Length(0, MaxSummaryLength).Error("Keep the note under 4,000 characters"),
		),
		validation.Field(&a.ContactName, validation.Length(0, 200)),
		validation.Field(&a.PromisedAmountMinor,
			validation.When(a.Kind == ActivityPromiseToPay,
				validation.Required.Error("Enter the amount the customer promised"),
				validation.By(func(any) error {
					if a.PromisedAmountMinor != nil && *a.PromisedAmountMinor <= 0 {
						return validation.NewError(
							"not_positive",
							"The promised amount must be more than zero",
						)
					}
					return nil
				}),
			),
		),
		validation.Field(&a.PromisedOn,
			validation.When(a.Kind == ActivityPromiseToPay,
				validation.Required.Error("Enter the date the customer promised to pay by"),
			),
		),
	))
}

// IsPromise reports whether the entry is a promise to pay.
func (a *Activity) IsPromise() bool {
	return a.Kind == ActivityPromiseToPay && a.PromisedAmountMinor != nil && a.PromisedOn != nil
}

func (a *Activity) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("cact_")
		}
		if a.CreatedAt == 0 {
			a.CreatedAt = timeutils.NowUnix()
		}
	}
	return nil
}

var _ bun.BeforeAppendModelHook = (*Account)(nil)

// Account is a customer as collections works it: its collector, its next
// follow-up, and the promise to pay standing against its balance.
type Account struct {
	bun.BaseModel `bun:"table:collection_accounts,alias:cacc" json:"-"`
	keel.Table    `keel:"collection_accounts"`

	ID                pulid.ID `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:cacc_"`
	OrganizationID    pulid.ID `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	CustomerID        pulid.ID `json:"customerId"        bun:"customer_id,type:VARCHAR(100),notnull" keel:"customer_id"`
	CollectorID       pulid.ID `json:"collectorId"       bun:"collector_id,type:VARCHAR(100),nullzero" keel:"collector_id,nullzero"`
	NextFollowUpOn    *int64   `json:"nextFollowUpOn"    bun:"next_follow_up_on,type:BIGINT,nullzero" keel:"next_follow_up_on"`
	PromiseActivityID pulid.ID `json:"promiseActivityId" bun:"promise_activity_id,type:VARCHAR(100),nullzero" keel:"promise_activity_id,nullzero"`
	Version           int64    `json:"version"           bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt         int64    `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt         int64    `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
}

func (a *Account) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("cacc_")
		}
		a.CreatedAt = now
		a.UpdatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}
	return nil
}
