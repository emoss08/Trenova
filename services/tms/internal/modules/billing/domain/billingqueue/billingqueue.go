package billingqueue

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ validationframework.TenantedEntity = (*BillingQueueItem)(nil)
	_ domaintypes.PostgresSearchable     = (*BillingQueueItem)(nil)
)

type BillingQueueItem struct {
	keel.Table `keel:"billing_queue_items"`

	ID                        pulid.ID             `json:"id" keel:"id,pk,autoid:bqi_"`
	OrganizationID            pulid.ID             `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID            pulid.ID             `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	ShipmentID                pulid.ID             `json:"shipmentId" keel:"shipment_id,nullzero"`
	OrderID                   pulid.ID             `json:"orderId" keel:"order_id,nullzero"`
	BillToCustomerID          pulid.ID             `json:"billToCustomerId" keel:"bill_to_customer_id"`
	AllocatedTotalAmount      decimal.Decimal      `json:"allocatedTotalAmount" keel:"allocated_total_amount"`
	AllocatedTotalAmountMinor int64                `json:"allocatedTotalAmountMinor" keel:"allocated_total_amount_minor"`
	AssignedBillerID          *pulid.ID            `json:"assignedBillerId" keel:"assigned_biller_id"`
	Number                    string               `json:"number" keel:"number,nullzero"`
	Status                    Status               `json:"status" keel:"status"`
	BillType                  BillType             `json:"billType" keel:"bill_type"`
	ExceptionReasonCode       *ExceptionReasonCode `json:"exceptionReasonCode" keel:"exception_reason_code"`
	ReviewNotes               string               `json:"reviewNotes" keel:"review_notes,nullzero"`
	ExceptionNotes            string               `json:"exceptionNotes" keel:"exception_notes,nullzero"`
	ReviewStartedAt           *int64               `json:"reviewStartedAt" keel:"review_started_at"`
	ReviewCompletedAt         *int64               `json:"reviewCompletedAt" keel:"review_completed_at"`
	CanceledByID              *pulid.ID            `json:"canceledById" keel:"canceled_by_id"`
	CanceledAt                *int64               `json:"canceledAt" keel:"canceled_at"`
	CancelReason              string               `json:"cancelReason" keel:"cancel_reason,nullzero"`
	HoldReasonCode            *HoldReasonCode      `json:"holdReasonCode" keel:"hold_reason_code"`
	HeldAt                    *int64               `json:"heldAt" keel:"held_at"`
	HeldByID                  *pulid.ID            `json:"heldById" keel:"held_by_id"`
	StatusBeforeHold          *Status              `json:"statusBeforeHold" keel:"status_before_hold"`
	IsAdjustmentOrigin        bool                 `json:"isAdjustmentOrigin" keel:"is_adjustment_origin"`
	InvoiceID                 pulid.ID             `json:"invoiceId" keel:"invoice_id,nullzero"`
	SourceInvoiceID           *pulid.ID            `json:"sourceInvoiceId" keel:"source_invoice_id"`
	SourceInvoiceAdjustmentID *pulid.ID            `json:"sourceInvoiceAdjustmentId" keel:"source_invoice_adjustment_id"`
	SourceCreditMemoInvoiceID *pulid.ID            `json:"sourceCreditMemoInvoiceId" keel:"source_credit_memo_invoice_id"`
	CorrectionGroupID         *pulid.ID            `json:"correctionGroupId" keel:"correction_group_id"`
	RebillStrategy            string               `json:"rebillStrategy" keel:"rebill_strategy,nullzero"`
	RequiresReplacementReview bool                 `json:"requiresReplacementReview" keel:"requires_replacement_review"`
	RerateVariancePercent     decimal.Decimal      `json:"rerateVariancePercent" keel:"rerate_variance_percent"`
	AdjustmentContext         map[string]any       `json:"adjustmentContext" keel:"adjustment_context,jsonb"`
	Version                   int64                `json:"version" keel:"version,version"`
	CreatedAt                 int64                `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt                 int64                `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector              string               `json:"-" keel:"search_vector,tsvector,readonly,noselect"`

	Shipment       *shipment.Shipment `json:"shipment,omitempty" keel:"belongs:shipment_id"`
	BillToCustomer *customer.Customer `json:"billToCustomer,omitempty" keel:"belongs:bill_to_customer_id"`
	AssignedBiller *tenant.User       `json:"assignedBiller,omitempty" keel:"belongs:assigned_biller_id"`
	CanceledBy     *tenant.User       `json:"canceledBy,omitempty" keel:"belongs:canceled_by_id"`

	// PayerShare is this item's payer's part of the shipment's charges, filled
	// when the item is read with its shipment details.
	PayerShare *PayerShare `json:"payerShare,omitempty"`

	// DetentionHolds are the detention charges on the shipment still waiting
	// on an approver, filled when the item is read with its shipment details.
	// While any is listed the item cannot be approved.
	DetentionHolds []*DetentionHold `json:"detentionHolds,omitempty"`

	RatingReviewHold *RatingReviewHold `json:"ratingReviewHold,omitempty"`

	// PONumber is the customer's purchase order, from the item's order. It is
	// read with the item rather than stored on it.
	PONumber string `json:"poNumber,omitempty" keel:"po_number,computed"`

	// Review is everything a biller reads to decide the item, filled when the
	// item is read with its shipment details.
	Review *Review `json:"review,omitempty"`
}

func (b *BillingQueueItem) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		b,
		validation.Field(
			&b.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&b.BusinessUnitID,
			validation.Required.Error("Business unit ID is required"),
		),
		validation.Field(
			&b.BillToCustomerID,
			validation.Required.Error("Bill-to customer is required"),
		),
	))

	// A correction of a consolidated invoice has neither a single shipment nor a
	// single order to point at; the invoice it corrects is what identifies it. A
	// standalone memo has none of those and is identified by its bill-to alone.
	if b.ShipmentID.IsNil() && b.OrderID.IsNil() &&
		(b.SourceInvoiceID == nil || b.SourceInvoiceID.IsNil()) &&
		b.BillToCustomerID.IsNil() {
		multiErr.Add(
			"shipmentId",
			errortypes.ErrRequired,
			"A billing queue item must be attached to a shipment, an order, a source invoice or a bill-to customer",
		)
	}
}

func (b *BillingQueueItem) SyncAllocatedMinor() {
	b.AllocatedTotalAmountMinor = money.MinorUnits(b.AllocatedTotalAmount)
}

func (b *BillingQueueItem) BeforeCreate(ctx context.Context) error {
	if b.Status == "" {
		b.Status = StatusReadyForReview
	}
	if b.BillType == "" {
		b.BillType = BillTypeInvoice
	}

	return nil
}

func (b *BillingQueueItem) GetID() pulid.ID {
	return b.ID
}

func (b *BillingQueueItem) GetOrganizationID() pulid.ID {
	return b.OrganizationID
}

func (b *BillingQueueItem) GetBusinessUnitID() pulid.ID {
	return b.BusinessUnitID
}

func (b *BillingQueueItem) GetTableName() string {
	return "billing_queue_items"
}

func (b *BillingQueueItem) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "bqi",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
			{
				Name:   "review_notes",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "exception_notes",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "cancel_reason",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:        "shipment",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*shipment.Shipment)(nil),
				TargetTable:  "shipments",
				ForeignKey:   "shipment_id",
				ReferenceKey: "id",
				Alias:        "sp",
				Queryable:    true,
			},
			{
				Field:        "bill_to_customer",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*customer.Customer)(nil),
				TargetTable:  "customers",
				ForeignKey:   "bill_to_customer_id",
				ReferenceKey: "id",
				Alias:        "bill_cus",
				Queryable:    true,
			},
			{
				Field:        "assigned_biller",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*tenant.User)(nil),
				TargetTable:  "users",
				ForeignKey:   "assigned_biller_id",
				ReferenceKey: "id",
				Alias:        "usr",
				Queryable:    true,
			},
		},
	}
}
