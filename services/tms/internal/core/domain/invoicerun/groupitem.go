package invoicerun

import (
	"context"

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

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RunID          pulid.ID `json:"runId"          bun:"run_id,type:VARCHAR(100),notnull"`
	GroupID        pulid.ID `json:"groupId"        bun:"group_id,type:VARCHAR(100),notnull"`

	BillingQueueItemID pulid.ID `json:"billingQueueItemId" bun:"billing_queue_item_id,type:VARCHAR(100),notnull"`
	ShipmentID         pulid.ID `json:"shipmentId"         bun:"shipment_id,type:VARCHAR(100),notnull"`
	OrderID            pulid.ID `json:"orderId"            bun:"order_id,type:VARCHAR(100),nullzero"`
	ProNumber          string   `json:"proNumber"          bun:"pro_number,type:VARCHAR(100),nullzero"`
	BOL                string   `json:"bol"                bun:"bol,type:VARCHAR(100),nullzero"`
	PONumber           string   `json:"poNumber"           bun:"po_number,type:VARCHAR(100),nullzero"`
	ServiceDate        *int64   `json:"serviceDate"        bun:"service_date,type:BIGINT,nullzero"`
	SortKey            int      `json:"sortKey"            bun:"sort_key,type:INTEGER,notnull"`

	Amount      decimal.Decimal `json:"amount"      bun:"amount,type:NUMERIC(19,4),notnull,default:0"`
	AmountMinor int64           `json:"amountMinor" bun:"amount_minor,type:BIGINT,notnull"`

	Excluded        bool   `json:"excluded"        bun:"excluded,type:BOOLEAN,notnull"`
	ExclusionReason string `json:"exclusionReason" bun:"exclusion_reason,type:TEXT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Group *InvoiceRunGroup `json:"-" bun:"rel:belongs-to,join:group_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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
