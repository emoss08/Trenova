package invoicerun

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*InvoiceRunGroupItem)(nil)

// InvoiceRunGroupItem is one billing-queue item proposed for one group.
//
// An operator removing a shipment sets Excluded rather than deleting the row:
// pulling a load off a statement is an auditable decision, and next period's
// biller needs to be able to see that it was made and why.
type InvoiceRunGroupItem struct {
	bun.BaseModel `bun:"table:invoice_run_group_items,alias:invrgi" json:"-"`
	keel.Table    `keel:"invoice_run_group_items"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:invrgi_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	RunID          pulid.ID `json:"runId"          bun:"run_id,type:VARCHAR(100),notnull" keel:"run_id"`
	GroupID        pulid.ID `json:"groupId"        bun:"group_id,type:VARCHAR(100),notnull" keel:"group_id"`

	BillingQueueItemID pulid.ID `json:"billingQueueItemId" bun:"billing_queue_item_id,type:VARCHAR(100),notnull" keel:"billing_queue_item_id"`
	ShipmentID         pulid.ID `json:"shipmentId"         bun:"shipment_id,type:VARCHAR(100),notnull" keel:"shipment_id"`
	OrderID            pulid.ID `json:"orderId"            bun:"order_id,type:VARCHAR(100),nullzero" keel:"order_id,nullzero"`
	ProNumber          string   `json:"proNumber"          bun:"pro_number,type:VARCHAR(100),nullzero" keel:"pro_number,nullzero"`
	BOL                string   `json:"bol"                bun:"bol,type:VARCHAR(100),nullzero" keel:"bol,nullzero"`
	PONumber           string   `json:"poNumber"           bun:"po_number,type:VARCHAR(100),nullzero" keel:"po_number,nullzero"`
	ServiceDate        *int64   `json:"serviceDate"        bun:"service_date,type:BIGINT,nullzero" keel:"service_date"`
	SortKey            int      `json:"sortKey"            bun:"sort_key,type:INTEGER,notnull" keel:"sort_key"`

	Amount      decimal.Decimal `json:"amount"      bun:"amount,type:NUMERIC(19,4),notnull,default:0" keel:"amount"`
	AmountMinor int64           `json:"amountMinor" bun:"amount_minor,type:BIGINT,notnull" keel:"amount_minor"`

	Excluded        bool   `json:"excluded"        bun:"excluded,type:BOOLEAN,notnull" keel:"excluded"`
	ExclusionReason string `json:"exclusionReason" bun:"exclusion_reason,type:TEXT,nullzero" keel:"exclusion_reason,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Group *InvoiceRunGroup `json:"-" bun:"rel:belongs-to,join:group_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:group_id"`
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

func (i *InvoiceRunGroupItem) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("invrgi_")
		}
		i.CreatedAt = now
		i.UpdatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}

func (i *InvoiceRunGroupItem) GetID() pulid.ID             { return i.ID }
func (i *InvoiceRunGroupItem) GetTableName() string        { return "invoice_run_group_items" }
func (i *InvoiceRunGroupItem) GetOrganizationID() pulid.ID { return i.OrganizationID }
func (i *InvoiceRunGroupItem) GetBusinessUnitID() pulid.ID { return i.BusinessUnitID }
