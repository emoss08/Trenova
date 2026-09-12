package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/emoss08/trenova/shared/pulid"
)

// PreviewInvoiceRunRequest builds a proposal for a period. CustomerIDs empty
// means every customer whose schedule is due.
type PreviewInvoiceRunRequest struct {
	TenantInfo  pagination.TenantInfo
	CustomerIDs []pulid.ID
	PeriodStart int64
	PeriodEnd   int64
	InvoiceDate int64
	Source      invoicerun.Source
	Cycle       customer.BillingCycle
}

// ItemExclusion pulls one shipment off the statement. The reason is required
// because next period's biller has to be able to see why.
type ItemExclusion struct {
	ItemID pulid.ID
	Reason string
}

// ItemMove sends one shipment to another group of the same customer.
type ItemMove struct {
	ItemID        pulid.ID
	TargetGroupID pulid.ID
}

// AdjustInvoiceRunMembershipRequest carries a whole operator edit, so the move
// and the exclusion land in one transaction and produce one audit entry rather
// than a trail of single-row changes nobody can read back.
type AdjustInvoiceRunMembershipRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
	Exclude    []ItemExclusion
	Include    []pulid.ID
	Moves      []ItemMove
}

type CommitInvoiceRunRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
}

// CommitGroupResult is what became of one proposed invoice.
type CommitGroupResult struct {
	GroupID       pulid.ID `json:"groupId"`
	GroupLabel    string   `json:"groupLabel"`
	Success       bool     `json:"success"`
	Skipped       bool     `json:"skipped"`
	InvoiceID     pulid.ID `json:"invoiceId,omitempty"`
	InvoiceNumber string   `json:"invoiceNumber,omitempty"`
	Error         string   `json:"error,omitempty"`
}

// CommitInvoiceRunResult mirrors the bulk envelope the rest of the codebase
// already uses, so a partially failed run reads the same way as a partially
// failed bulk transfer.
type CommitInvoiceRunResult struct {
	Run          *invoicerun.InvoiceRun `json:"run"`
	Results      []CommitGroupResult    `json:"results"`
	TotalCount   int                    `json:"totalCount"`
	SuccessCount int                    `json:"successCount"`
	SkippedCount int                    `json:"skippedCount"`
	ErrorCount   int                    `json:"errorCount"`
}

type CancelInvoiceRunRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID
	Reason     string
}

// ListOpenStatementsRequest asks what every statement-billed customer has
// accumulated so far in the period they are currently in.
//
// CustomerID narrows to one customer. IncludeShipments fills in each group's
// members, which the list view does not need and the detail view does.
type ListOpenStatementsRequest struct {
	TenantInfo       pagination.TenantInfo
	CustomerID       pulid.ID
	IncludeShipments bool
}

// StatementShipment is one delivered, approved, uninvoiced shipment sitting on
// an open statement.
type StatementShipment struct {
	BillingQueueItemID pulid.ID        `json:"billingQueueItemId"`
	ShipmentID         pulid.ID        `json:"shipmentId"`
	OrderID            pulid.ID        `json:"orderId,omitempty"`
	ProNumber          string          `json:"proNumber"`
	BOL                string          `json:"bol"`
	PONumber           string          `json:"poNumber"`
	OrderNumber        string          `json:"orderNumber"`
	ServiceDate        *int64          `json:"serviceDate"`
	Amount             decimal.Decimal `json:"amount"`
}

// StatementGroup is one invoice the statement will produce when it bills.
//
// It exists before anything is persisted: the split key is applied to the live
// candidate set every time the statement is read, so a biller watching a
// statement accumulate sees the invoices it would cut right now.
type StatementGroup struct {
	Key           string               `json:"key"`
	Label         string               `json:"label"`
	ShipmentCount int                  `json:"shipmentCount"`
	TotalAmount   decimal.Decimal      `json:"totalAmount"`
	BelowMinimum  bool                 `json:"belowMinimum"`
	Shipments     []*StatementShipment `json:"shipments,omitempty"`
}

// OpenStatement is one customer's current billing period as it stands right now.
//
// Nothing here is persisted. The period comes from the customer's own schedule,
// the members from the live billing queue, and the split from their profile — so
// the statement is always current and a shipment approved a minute ago is on it
// a minute later, with no job to wait for.
type OpenStatement struct {
	CustomerID     pulid.ID `json:"customerId"`
	CustomerName   string   `json:"customerName"`
	CustomerCode   string   `json:"customerCode"`
	CustomerStatus string   `json:"customerStatus"`

	Cycle                 customer.BillingCycle `json:"cycle"`
	BillingCycleAnchorDay int16                 `json:"billingCycleAnchorDay"`
	BillingCycleTimezone  string                `json:"billingCycleTimezone"`
	PeriodStart           int64                 `json:"periodStart"`
	PeriodEnd             int64                 `json:"periodEnd"`
	LastBilledPeriodEnd   *int64                `json:"lastBilledPeriodEnd"`

	ShipmentCount int             `json:"shipmentCount"`
	InvoiceCount  int             `json:"invoiceCount"`
	TotalAmount   decimal.Decimal `json:"totalAmount"`
	CurrencyCode  string          `json:"currencyCode"`

	SplitBy       customer.InvoiceSplitKey   `json:"splitBy"`
	SectionBy     customer.InvoiceSectionKey `json:"sectionBy"`
	Detail        customer.InvoiceDetail     `json:"detail"`
	MinimumAmount decimal.NullDecimal        `json:"minimumAmount"`
	AutoBill      bool                       `json:"autoBill"`

	// BelowMinimum means every group is under the customer's floor, so billing
	// today would produce nothing and the freight would roll into next period.
	BelowMinimum bool `json:"belowMinimum"`

	// HeldCount and HeldAmount are the freight that belongs to this period but is
	// still waiting on a biller, and so will not be on the invoice.
	//
	// Without them the statement understates the period: a biller sees 197
	// shipments and has no way to know 12 more are sitting in review. Chasing
	// those down is the work that has to happen before the cycle closes.
	HeldCount  int             `json:"heldCount"`
	HeldAmount decimal.Decimal `json:"heldAmount"`

	Groups []*StatementGroup `json:"groups,omitempty"`
}

// BillStatementNowRequest bills an open period before it closes.
//
// Reason is required and recorded. Billing off-cycle is allowed — a customer
// closing their books early, a credit hold about to bite — but it is a deviation
// from what the customer agreed to, so the next person to look has to be able to
// see who decided it and why.
type BillStatementNowRequest struct {
	TenantInfo pagination.TenantInfo
	CustomerID pulid.ID
	Reason     string
	// Exclude holds back shipments the biller pulled off the statement before
	// billing, keyed by billing-queue item because that is the only id an open
	// statement has — nothing is persisted until it bills. They stay approved and
	// uninvoiced, so the next period picks them up.
	Exclude []StatementExclusion
}

// StatementExclusion pulls one shipment off a statement at billing time. The
// reason rides along for the same reason ItemExclusion's does: next period's
// biller has to be able to see why.
type StatementExclusion struct {
	BillingQueueItemID pulid.ID
	Reason             string
}

// InvoiceRunSweepResult is what one pass of the scheduled sweep did.
type InvoiceRunSweepResult struct {
	SchedulesDue    int `json:"schedulesDue"`
	RunsBuilt       int `json:"runsBuilt"`
	InvoicesCreated int `json:"invoicesCreated"`
	GroupsSkipped   int `json:"groupsSkipped"`
	Failed          int `json:"failed"`
}

// InvoiceRunSweeper is the scheduled half of statement billing.
//
// It is an interface here rather than a concrete dependency because the billing
// job package is imported by the invoice service, and the run service depends on
// the invoice service — taking the concrete type would close that loop.
type InvoiceRunSweeper interface {
	SweepDueSchedules(
		ctx context.Context,
		actor *RequestActor,
	) (*InvoiceRunSweepResult, error)
}
