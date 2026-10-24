package invoicerun

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

// InvoiceRunGroup is one proposed invoice: the shipments that share a split key
// within a run.
type InvoiceRunGroup struct {
	keel.Table `                                                    keel:"invoice_run_groups"`

	ID             pulid.ID                 `json:"id" keel:"id,pk,autoid:invrg_"`
	BusinessUnitID pulid.ID                 `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID                 `json:"organizationId" keel:"organization_id,pk,tenant"`
	RunID          pulid.ID                 `json:"runId" keel:"run_id"`
	CustomerID     pulid.ID                 `json:"customerId" keel:"customer_id"`
	GroupKey       string                   `json:"groupKey" keel:"group_key"`
	GroupLabel     string                   `json:"groupLabel" keel:"group_label"`
	SplitBy        customer.InvoiceSplitKey `json:"splitBy" keel:"split_by"`
	Status         GroupStatus              `json:"status" keel:"status"`

	ItemCount           int             `json:"itemCount" keel:"item_count"`
	SubtotalAmount      decimal.Decimal `json:"subtotalAmount" keel:"subtotal_amount"`
	SubtotalAmountMinor int64           `json:"subtotalAmountMinor" keel:"subtotal_amount_minor"`
	TotalAmount         decimal.Decimal `json:"totalAmount" keel:"total_amount"`
	TotalAmountMinor    int64           `json:"totalAmountMinor" keel:"total_amount_minor"`
	CurrencyCode        string          `json:"currencyCode" keel:"currency_code"`

	// MinimumAmount is the customer's floor for a statement, copied onto the group
	// when the run is built. A group below it defers to the next period rather
	// than billing, so a customer is not sent a four-dollar invoice.
	MinimumAmount decimal.NullDecimal `json:"minimumAmount" keel:"minimum_amount"`
	// AutoBill says the customer asked for this to bill without review, so a
	// scheduled run commits it rather than leaving it for a biller.
	AutoBill bool `json:"autoBill" keel:"auto_bill"`

	InvoiceID  pulid.ID `json:"invoiceId" keel:"invoice_id,nullzero"`
	SkipReason string   `json:"skipReason" keel:"skip_reason,nullzero"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Run      *InvoiceRun            `json:"-" keel:"belongs:run_id"`
	Customer *customer.Customer     `json:"customer,omitempty" keel:"belongs:customer_id"`
	Invoice  *invoice.Invoice       `json:"invoice,omitempty" keel:"belongs:invoice_id"`
	Items    []*InvoiceRunGroupItem `json:"items,omitempty" keel:"many:group_id"`
}

func (g *InvoiceRunGroup) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(g,
		validation.Field(&g.CustomerID, validation.Required.Error("Customer is required")),
		validation.Field(&g.GroupKey,
			validation.Required.Error("Group key is required"),
			validation.Length(1, 255).Error("Group key cannot be longer than 255 characters"),
		),
		validation.Field(&g.GroupLabel,
			validation.Required.Error("Group label is required"),
			validation.Length(1, 255).Error("Group label cannot be longer than 255 characters"),
		),
	))

	if !g.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Group status is invalid")
	}
	// A skipped group is a decision, and next month's biller needs to know why.
	if g.Status == GroupStatusSkipped && g.SkipReason == "" {
		multiErr.Add(
			"skipReason",
			errortypes.ErrRequired,
			"A skipped group must record why it was not billed",
		)
	}
	if g.Status == GroupStatusCommitted && g.InvoiceID.IsNil() {
		multiErr.Add(
			"invoiceId",
			errortypes.ErrRequired,
			"A committed group must name the invoice it produced",
		)
	}
}

// IncludedItems are the items that would actually be billed.
func (g *InvoiceRunGroup) IncludedItems() []*InvoiceRunGroupItem {
	included := make([]*InvoiceRunGroupItem, 0, len(g.Items))
	for _, item := range g.Items {
		if item != nil && !item.Excluded {
			included = append(included, item)
		}
	}

	return included
}

func (g *InvoiceRunGroup) ExcludedCount() int {
	count := 0
	for _, item := range g.Items {
		if item != nil && item.Excluded {
			count++
		}
	}

	return count
}

// SyncTotals recomputes the group's rollups from the items it would bill.
//
// These are advisory: the binding totals come from the invoice lines at commit.
// They exist so the operator sees a number that moves the instant a shipment is
// excluded, rather than one that only agrees after the run is committed.
func (g *InvoiceRunGroup) SyncTotals() {
	total := decimal.Zero
	count := 0

	for _, item := range g.IncludedItems() {
		total = total.Add(item.Amount)
		count++
	}

	g.ItemCount = count
	g.SubtotalAmount = total
	g.SubtotalAmountMinor = money.MinorUnits(total)
	g.TotalAmount = total
	g.TotalAmountMinor = money.MinorUnits(total)
}

func (g *InvoiceRunGroup) BeforeCreate(ctx context.Context) error {
	if g.Status == "" {
		g.Status = GroupStatusPending
	}
	if g.CurrencyCode == "" {
		g.CurrencyCode = "USD"
	}

	return nil
}

func (g *InvoiceRunGroup) GetID() pulid.ID             { return g.ID }
func (g *InvoiceRunGroup) GetTableName() string        { return "invoice_run_groups" }
func (g *InvoiceRunGroup) GetOrganizationID() pulid.ID { return g.OrganizationID }
func (g *InvoiceRunGroup) GetBusinessUnitID() pulid.ID { return g.BusinessUnitID }
