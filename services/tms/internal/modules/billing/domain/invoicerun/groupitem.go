package invoicerun

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

// InvoiceRunGroupItem is one billing-queue item proposed for one group.
//
// An operator removing a shipment sets Excluded rather than deleting the row:
// pulling a load off a statement is an auditable decision, and next period's
// biller needs to be able to see that it was made and why.
type InvoiceRunGroupItem struct {
	keel.Table `keel:"invoice_run_group_items"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:invrgi_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	RunID          pulid.ID `json:"runId"          keel:"run_id"`
	GroupID        pulid.ID `json:"groupId"        keel:"group_id"`

	BillingQueueItemID pulid.ID `json:"billingQueueItemId" keel:"billing_queue_item_id"`
	ShipmentID         pulid.ID `json:"shipmentId"         keel:"shipment_id"`
	OrderID            pulid.ID `json:"orderId"            keel:"order_id,nullzero"`
	ProNumber          string   `json:"proNumber"          keel:"pro_number,nullzero"`
	BOL                string   `json:"bol"                keel:"bol,nullzero"`
	PONumber           string   `json:"poNumber"           keel:"po_number,nullzero"`
	ServiceDate        *int64   `json:"serviceDate"        keel:"service_date"`
	SortKey            int      `json:"sortKey"            keel:"sort_key"`

	Amount      decimal.Decimal `json:"amount"      keel:"amount"`
	AmountMinor int64           `json:"amountMinor" keel:"amount_minor"`

	Excluded        bool   `json:"excluded"        keel:"excluded"`
	ExclusionReason string `json:"exclusionReason" keel:"exclusion_reason,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Group *InvoiceRunGroup `json:"-" keel:"belongs:group_id"`
}

func (i *InvoiceRunGroupItem) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.BillingQueueItemID,
			validation.Required.Error("Billing queue item is required"),
		),
		validation.Field(&i.ShipmentID, validation.Required.Error("Shipment is required")),
	))

	if i.Excluded && i.ExclusionReason == "" {
		multiErr.Add(
			"exclusionReason",
			errortypes.ErrRequired,
			"An excluded shipment must record why it was pulled off the invoice",
		)
	}
}

func (i *InvoiceRunGroupItem) SyncMinorAmount() {
	i.AmountMinor = money.MinorUnits(i.Amount)
}

func (i *InvoiceRunGroupItem) GetID() pulid.ID             { return i.ID }
func (i *InvoiceRunGroupItem) GetTableName() string        { return "invoice_run_group_items" }
func (i *InvoiceRunGroupItem) GetOrganizationID() pulid.ID { return i.OrganizationID }
func (i *InvoiceRunGroupItem) GetBusinessUnitID() pulid.ID { return i.BusinessUnitID }
