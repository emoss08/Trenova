package invoice

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
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

// InvoiceDispute is one dispute case raised against a posted invoice. An
// invoice has at most one open case; resolving or withdrawing it clears the
// invoice's Disputed flag.
type InvoiceDispute struct {
	keel.Table `                                                 keel:"invoice_disputes"`

	ID                     pulid.ID          `json:"id" keel:"id,pk,autoid:idsp_"`
	OrganizationID         pulid.ID          `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID         pulid.ID          `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	InvoiceID              pulid.ID          `json:"invoiceId" keel:"invoice_id"`
	CustomerID             pulid.ID          `json:"customerId" keel:"customer_id"`
	Status                 DisputeCaseStatus `json:"status" keel:"status"`
	ReasonCode             DisputeReasonCode `json:"reasonCode" keel:"reason_code"`
	DisputedAmount         decimal.Decimal   `json:"disputedAmount" keel:"disputed_amount"`
	DisputedAmountMinor    int64             `json:"disputedAmountMinor" keel:"disputed_amount_minor"`
	Notes                  string            `json:"notes" keel:"notes,nullzero"`
	OpenedByID             pulid.ID          `json:"openedById" keel:"opened_by_id"`
	OpenedAt               int64             `json:"openedAt" keel:"opened_at"`
	ResolvedByID           pulid.ID          `json:"resolvedById" keel:"resolved_by_id,nullzero"`
	ResolvedAt             *int64            `json:"resolvedAt" keel:"resolved_at"`
	Resolution             DisputeResolution `json:"resolution" keel:"resolution,nullzero"`
	ResolutionAdjustmentID pulid.ID          `json:"resolutionAdjustmentId" keel:"resolution_adjustment_id,nullzero"`
	ResolutionNotes        string            `json:"resolutionNotes" keel:"resolution_notes,nullzero"`
	Version                int64             `json:"version" keel:"version,version"`
	CreatedAt              int64             `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt              int64             `json:"updatedAt" keel:"updated_at,autoupdate"`

	Invoice  *Invoice           `json:"invoice,omitempty" keel:"belongs:invoice_id"`
	Customer *customer.Customer `json:"customer,omitempty" keel:"belongs:customer_id"`
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

func (d *InvoiceDispute) BeforeCreate(ctx context.Context) error {
	if d.Status == "" {
		d.Status = DisputeCaseStatusOpen
	}
	now := timeutils.NowUnix()
	if d.OpenedAt == 0 {
		d.OpenedAt = now
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
