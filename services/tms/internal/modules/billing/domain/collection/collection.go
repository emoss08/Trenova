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
	validation "github.com/go-ozzo/ozzo-validation/v4"
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

// Activity is one entry in a customer's collections history. Entries are
// written once and never changed, so the history is what happened.
type Activity struct {
	keel.Table `keel:"collection_activities"`

	ID                  pulid.ID     `json:"id"                  keel:"id,pk,autoid:cact_"`
	OrganizationID      pulid.ID     `json:"organizationId"      keel:"organization_id,pk,tenant"`
	BusinessUnitID      pulid.ID     `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	CustomerID          pulid.ID     `json:"customerId"          keel:"customer_id"`
	Kind                ActivityKind `json:"kind"                keel:"kind"`
	Summary             string       `json:"summary"             keel:"summary,nullzero"`
	ContactName         string       `json:"contactName"         keel:"contact_name,nullzero"`
	InvoiceID           pulid.ID     `json:"invoiceId"           keel:"invoice_id,nullzero"`
	PromisedAmountMinor *int64       `json:"promisedAmountMinor" keel:"promised_amount_minor"`
	PromisedOn          *int64       `json:"promisedOn"          keel:"promised_on"`
	CurrencyCode        string       `json:"currencyCode"        keel:"currency_code"`
	FollowUpOn          *int64       `json:"followUpOn"          keel:"follow_up_on"`
	CollectorID         pulid.ID     `json:"collectorId"         keel:"collector_id,nullzero"`
	CreatedByID         pulid.ID     `json:"createdById"         keel:"created_by_id"`
	CreatedAt           int64        `json:"createdAt"           keel:"created_at,autocreate"`
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

func (a *Activity) BeforeCreate(ctx context.Context) error {
	if a.CurrencyCode == "" {
		a.CurrencyCode = "USD"
	}

	return nil
}

// Account is a customer as collections works it: its collector, its next
// follow-up, and the promise to pay standing against its balance.
type Account struct {
	keel.Table `keel:"collection_accounts"`

	ID                pulid.ID `json:"id"                keel:"id,pk,autoid:cacc_"`
	OrganizationID    pulid.ID `json:"organizationId"    keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	CustomerID        pulid.ID `json:"customerId"        keel:"customer_id"`
	CollectorID       pulid.ID `json:"collectorId"       keel:"collector_id,nullzero"`
	NextFollowUpOn    *int64   `json:"nextFollowUpOn"    keel:"next_follow_up_on"`
	PromiseActivityID pulid.ID `json:"promiseActivityId" keel:"promise_activity_id,nullzero"`
	Version           int64    `json:"version"           keel:"version,version"`
	CreatedAt         int64    `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64    `json:"updatedAt"         keel:"updated_at,autoupdate"`
}
