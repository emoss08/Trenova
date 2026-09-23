package invoice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// DisputeCaseStatus is where a dispute case stands. The invoice's own
// DisputeStatus flag is derived from it: Disputed while a case is Open.
type DisputeCaseStatus string

const (
	DisputeCaseStatusOpen      = DisputeCaseStatus("Open")
	DisputeCaseStatusResolved  = DisputeCaseStatus("Resolved")
	DisputeCaseStatusWithdrawn = DisputeCaseStatus("Withdrawn")
)

func (s DisputeCaseStatus) IsValid() bool {
	switch s {
	case DisputeCaseStatusOpen, DisputeCaseStatusResolved, DisputeCaseStatusWithdrawn:
		return true
	default:
		return false
	}
}

// DisputeReasonCode is why the customer is withholding payment.
type DisputeReasonCode string

const (
	DisputeReasonRateDiscrepancy      = DisputeReasonCode("RateDiscrepancy")
	DisputeReasonAccessorialDisputed  = DisputeReasonCode("AccessorialDisputed")
	DisputeReasonServiceFailure       = DisputeReasonCode("ServiceFailure")
	DisputeReasonDuplicateBilling     = DisputeReasonCode("DuplicateBilling")
	DisputeReasonWrongBillTo          = DisputeReasonCode("WrongBillTo")
	DisputeReasonMissingDocumentation = DisputeReasonCode("MissingDocumentation")
	DisputeReasonOther                = DisputeReasonCode("Other")
)

func (c DisputeReasonCode) IsValid() bool {
	switch c {
	case DisputeReasonRateDiscrepancy, DisputeReasonAccessorialDisputed,
		DisputeReasonServiceFailure, DisputeReasonDuplicateBilling,
		DisputeReasonWrongBillTo, DisputeReasonMissingDocumentation,
		DisputeReasonOther:
		return true
	default:
		return false
	}
}

// DisputeResolution is how an open dispute ended.
type DisputeResolution string

const (
	DisputeResolutionCreditIssued     = DisputeResolution("CreditIssued")
	DisputeResolutionInvoiceUpheld    = DisputeResolution("InvoiceUpheld")
	DisputeResolutionRebilled         = DisputeResolution("Rebilled")
	DisputeResolutionWrittenOff       = DisputeResolution("WrittenOff")
	DisputeResolutionCustomerWithdrew = DisputeResolution("CustomerWithdrew")
)

func (r DisputeResolution) IsValid() bool {
	switch r {
	case DisputeResolutionCreditIssued, DisputeResolutionInvoiceUpheld,
		DisputeResolutionRebilled, DisputeResolutionWrittenOff,
		DisputeResolutionCustomerWithdrew:
		return true
	default:
		return false
	}
}

// RequiresAdjustment says whether the resolution must point at an executed
// adjustment on the disputed invoice, because money moved to settle it.
func (r DisputeResolution) RequiresAdjustment() bool {
	return r == DisputeResolutionCreditIssued || r == DisputeResolutionWrittenOff
}

var _ bun.BeforeAppendModelHook = (*InvoiceDispute)(nil)

// InvoiceDispute is one dispute case raised against a posted invoice. An
// invoice has at most one open case; resolving or withdrawing it clears the
// invoice's Disputed flag.
type InvoiceDispute struct {
	bun.BaseModel `bun:"table:invoice_disputes,alias:idsp" json:"-"`

	ID                     pulid.ID          `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID         pulid.ID          `json:"organizationId"         bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID          `json:"businessUnitId"         bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	InvoiceID              pulid.ID          `json:"invoiceId"              bun:"invoice_id,type:VARCHAR(100),notnull"`
	CustomerID             pulid.ID          `json:"customerId"             bun:"customer_id,type:VARCHAR(100),notnull"`
	Status                 DisputeCaseStatus `json:"status"                 bun:"status,type:VARCHAR(20),notnull,default:'Open'"`
	ReasonCode             DisputeReasonCode `json:"reasonCode"             bun:"reason_code,type:VARCHAR(40),notnull"`
	DisputedAmount         decimal.Decimal   `json:"disputedAmount"         bun:"disputed_amount,type:NUMERIC(19,4),notnull"`
	DisputedAmountMinor    int64             `json:"disputedAmountMinor"    bun:"disputed_amount_minor,type:BIGINT,notnull"`
	Notes                  string            `json:"notes"                  bun:"notes,type:TEXT,nullzero"`
	OpenedByID             pulid.ID          `json:"openedById"             bun:"opened_by_id,type:VARCHAR(100),notnull"`
	OpenedAt               int64             `json:"openedAt"               bun:"opened_at,type:BIGINT,notnull"`
	ResolvedByID           pulid.ID          `json:"resolvedById"           bun:"resolved_by_id,type:VARCHAR(100),nullzero"`
	ResolvedAt             *int64            `json:"resolvedAt"             bun:"resolved_at,type:BIGINT,nullzero"`
	Resolution             DisputeResolution `json:"resolution"             bun:"resolution,type:VARCHAR(30),nullzero"`
	ResolutionAdjustmentID pulid.ID          `json:"resolutionAdjustmentId" bun:"resolution_adjustment_id,type:VARCHAR(100),nullzero"`
	ResolutionNotes        string            `json:"resolutionNotes"        bun:"resolution_notes,type:TEXT,nullzero"`
	Version                int64             `json:"version"                bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt              int64             `json:"createdAt"              bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64             `json:"updatedAt"              bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Invoice  *Invoice           `json:"invoice,omitempty"  bun:"rel:belongs-to,join:invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Customer *customer.Customer `json:"customer,omitempty" bun:"rel:belongs-to,join:customer_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (d *InvoiceDispute) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		d,
		validation.Field(&d.OrganizationID, validation.Required),
		validation.Field(&d.BusinessUnitID, validation.Required),
		validation.Field(&d.InvoiceID, validation.Required.Error("Invoice is required")),
		validation.Field(&d.CustomerID, validation.Required.Error("Customer is required")),
		validation.Field(
			&d.Status,
			validation.Required,
			domainvalidation.ValidEnum[DisputeCaseStatus]("Invalid dispute status"),
		),
		validation.Field(
			&d.ReasonCode,
			validation.Required.Error("Reason is required"),
			domainvalidation.ValidEnum[DisputeReasonCode]("Invalid dispute reason"),
		),
		validation.Field(&d.OpenedByID, validation.Required.Error("Opened by is required")),
		validation.Field(&d.OpenedAt, validation.Required.Error("Opened at is required")),
	))
	if d.DisputedAmount.LessThanOrEqual(decimal.Zero) {
		multiErr.Add(
			"disputedAmount",
			errortypes.ErrInvalid,
			"Disputed amount must be greater than zero",
		)
	}
	if d.Status == DisputeCaseStatusResolved {
		if !d.Resolution.IsValid() {
			multiErr.Add(
				"resolution",
				errortypes.ErrRequired,
				"A resolved dispute needs a resolution",
			)
		}
		if d.Resolution.RequiresAdjustment() && d.ResolutionAdjustmentID.IsNil() {
			multiErr.Add(
				"resolutionAdjustmentId",
				errortypes.ErrRequired,
				"This resolution needs the adjustment that settled it",
			)
		}
	}
}

func (d *InvoiceDispute) IsOpen() bool {
	return d != nil && d.Status == DisputeCaseStatusOpen
}

// SyncMinor keeps the minor-unit column in step with the decimal amount.
func (d *InvoiceDispute) SyncMinor() {
	d.DisputedAmountMinor = money.MinorUnits(d.DisputedAmount)
}

func (d *InvoiceDispute) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("idsp_")
		}
		if d.Status == "" {
			d.Status = DisputeCaseStatusOpen
		}
		if d.OpenedAt == 0 {
			d.OpenedAt = now
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}

func (d *InvoiceDispute) GetID() pulid.ID {
	return d.ID
}

func (d *InvoiceDispute) GetOrganizationID() pulid.ID {
	return d.OrganizationID
}

func (d *InvoiceDispute) GetBusinessUnitID() pulid.ID {
	return d.BusinessUnitID
}
