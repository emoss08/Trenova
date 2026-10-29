package paymentreminderports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/paymentreminder"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// OverdueInvoice is one posted invoice past its due date with money still
// open, and the latest scheduled reminder step it has already been sent.
type OverdueInvoice struct {
	InvoiceID     pulid.ID
	InvoiceNumber string
	CustomerID    pulid.ID
	CurrencyCode  string
	InvoiceDate   int64
	DueDate       int64
	OpenMinor     int64
	LastStep      int
	Disputed      bool
}

// ListOverdueInvoicesRequest reads a tenant's overdue invoices as of a time,
// for every customer or the ones named.
type ListOverdueInvoicesRequest struct {
	TenantInfo  pagination.TenantInfo
	AsOf        int64
	CustomerIDs []pulid.ID
}

type Repository interface {
	ListOverdueInvoices(
		ctx context.Context,
		req ListOverdueInvoicesRequest,
	) ([]*OverdueInvoice, error)
	InsertReminders(ctx context.Context, reminders []*paymentreminder.Reminder) error
}

// SweepResult is what one organization's scheduled reminders did.
type SweepResult struct {
	Sent            int
	Customers       int
	NoRecipients    []string
	Failed          int
	WaitedOnPromise int
}

// SendNowRequest sends a customer a reminder of everything overdue, now,
// whatever the schedule says.
type SendNowRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	CustomerID pulid.ID              `json:"-"`
	Note       string                `json:"note"`
}

// SendNowResult says who the reminder went to and what it listed.
type SendNowResult struct {
	SentTo         []string `json:"sentTo"`
	InvoiceNumbers []string `json:"invoiceNumbers"`
	TotalOverdue   string   `json:"totalOverdue"`
}

type SendNowPreview struct {
	CustomerID        pulid.ID
	CustomerName      string
	CurrencyCode      string
	InvoiceNumbers    []string
	TotalOverdue      string
	OldestDaysPastDue int
	Email             *emailports.PlannedEmail
}

type Service interface {
	SendDue(
		ctx context.Context,
		control *billingcontrol.BillingControl,
		now int64,
	) (*SweepResult, error)
	SendNow(
		ctx context.Context,
		req *SendNowRequest,
		actor *permissionports.RequestActor,
	) ([]*SendNowResult, error)
	PreviewSendNow(
		ctx context.Context,
		req *SendNowRequest,
		actor *permissionports.RequestActor,
	) ([]*SendNowPreview, error)
}
