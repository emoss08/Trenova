package invoicerun

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*InvoiceRunGroup)(nil)

// InvoiceRunGroup is one proposed invoice: the shipments that share a split key
// within a run.
type InvoiceRunGroup struct {
	bun.BaseModel `bun:"table:invoice_run_groups,alias:invrg" json:"-"`

	ID             pulid.ID                 `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID                 `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID                 `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RunID          pulid.ID                 `json:"runId"          bun:"run_id,type:VARCHAR(100),notnull"`
	CustomerID     pulid.ID                 `json:"customerId"     bun:"customer_id,type:VARCHAR(100),notnull"`
	GroupKey       string                   `json:"groupKey"       bun:"group_key,type:VARCHAR(255),notnull"`
	GroupLabel     string                   `json:"groupLabel"     bun:"group_label,type:VARCHAR(255),notnull"`
	SplitBy        customer.InvoiceSplitKey `json:"splitBy"        bun:"split_by,type:invoice_split_key_enum,notnull"`
	Status         GroupStatus              `json:"status"         bun:"status,type:invoice_run_group_status_enum,notnull,default:'Pending'"`

	ItemCount           int             `json:"itemCount"           bun:"item_count,type:INTEGER,notnull"`
	SubtotalAmount      decimal.Decimal `json:"subtotalAmount"      bun:"subtotal_amount,type:NUMERIC(19,4),notnull,default:0"`
	SubtotalAmountMinor int64           `json:"subtotalAmountMinor" bun:"subtotal_amount_minor,type:BIGINT,notnull"`
	TotalAmount         decimal.Decimal `json:"totalAmount"         bun:"total_amount,type:NUMERIC(19,4),notnull,default:0"`
	TotalAmountMinor    int64           `json:"totalAmountMinor"    bun:"total_amount_minor,type:BIGINT,notnull"`
	CurrencyCode        string          `json:"currencyCode"        bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`

	// MinimumAmount is the customer's floor for a statement, copied onto the group
	// when the run is built. A group below it defers to the next period rather
	// than billing, so a customer is not sent a four-dollar invoice.
	MinimumAmount decimal.NullDecimal `json:"minimumAmount" bun:"minimum_amount,type:NUMERIC(19,4),nullzero"`
	// AutoBill says the customer asked for this to bill without review, so a
	// scheduled run commits it rather than leaving it for a biller.
	AutoBill bool `json:"autoBill" bun:"auto_bill,type:BOOLEAN,notnull"`

	InvoiceID  pulid.ID `json:"invoiceId"  bun:"invoice_id,type:VARCHAR(100),nullzero"`
	SkipReason string   `json:"skipReason" bun:"skip_reason,type:TEXT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Run      *InvoiceRun            `json:"-"                  bun:"rel:belongs-to,join:run_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Customer *customer.Customer     `json:"customer,omitempty" bun:"rel:belongs-to,join:customer_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Invoice  *invoice.Invoice       `json:"invoice,omitempty"  bun:"rel:belongs-to,join:invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Items    []*InvoiceRunGroupItem `json:"items,omitempty"    bun:"rel:has-many,join:id=group_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (g *InvoiceRunGroup) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if g.ID.IsNil() {
			g.ID = pulid.MustNew("invrg_")
		}
		if g.Status == "" {
			g.Status = GroupStatusPending
		}
		if g.CurrencyCode == "" {
			g.CurrencyCode = "USD"
		}
		g.CreatedAt = now
		g.UpdatedAt = now
	case *bun.UpdateQuery:
		g.UpdatedAt = now
	}

	return nil
}

func (g *InvoiceRunGroup) GetID() pulid.ID             { return g.ID }
func (g *InvoiceRunGroup) GetTableName() string        { return "invoice_run_groups" }
func (g *InvoiceRunGroup) GetOrganizationID() pulid.ID { return g.OrganizationID }
func (g *InvoiceRunGroup) GetBusinessUnitID() pulid.ID { return g.BusinessUnitID }
