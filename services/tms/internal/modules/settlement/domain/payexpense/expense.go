package payexpense

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ pagination.CursorEntity            = (*Expense)(nil)
	_ validationframework.TenantedEntity = (*Expense)(nil)
	_ domaintypes.PostgresSearchable     = (*Expense)(nil)
)

type ExpenseStatus string

const (
	ExpenseStatusPending    = ExpenseStatus("Pending")
	ExpenseStatusApproved   = ExpenseStatus("Approved")
	ExpenseStatusRejected   = ExpenseStatus("Rejected")
	ExpenseStatusReimbursed = ExpenseStatus("Reimbursed")
	ExpenseStatusCancelled  = ExpenseStatus("Cancelled")
)

func (s ExpenseStatus) String() string { return string(s) }

func (s ExpenseStatus) IsValid() bool {
	switch s {
	case ExpenseStatusPending, ExpenseStatusApproved, ExpenseStatusRejected,
		ExpenseStatusReimbursed, ExpenseStatusCancelled:
		return true
	default:
		return false
	}
}

func (s ExpenseStatus) IsTerminal() bool {
	return s == ExpenseStatusRejected || s == ExpenseStatusReimbursed ||
		s == ExpenseStatusCancelled
}

type Expense struct {
	keel.Table `keel:"driver_expenses"`

	ID                pulid.ID      `json:"id"                keel:"id,pk,autoid:dexp_"`
	BusinessUnitID    pulid.ID      `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID      `json:"organizationId"    keel:"organization_id,pk,tenant"`
	WorkerID          pulid.ID      `json:"workerId"          keel:"worker_id"`
	ShipmentID        *pulid.ID     `json:"shipmentId"        keel:"shipment_id"`
	PayCodeID         *pulid.ID     `json:"payCodeId"         keel:"pay_code_id"`
	Status            ExpenseStatus `json:"status"            keel:"status"`
	AmountMinor       int64         `json:"amountMinor"       keel:"amount_minor"`
	CurrencyCode      string        `json:"currencyCode"      keel:"currency_code"`
	Description       string        `json:"description"       keel:"description"`
	IncurredDate      int64         `json:"incurredDate"      keel:"incurred_date"`
	ReceiptDocumentID *pulid.ID     `json:"receiptDocumentId" keel:"receipt_document_id"`
	SubmittedByUserID pulid.ID      `json:"submittedByUserId" keel:"submitted_by_user_id"`
	ReviewNote        string        `json:"reviewNote"        keel:"review_note,nullzero"`
	ReviewedByID      *pulid.ID     `json:"reviewedById"      keel:"reviewed_by_id"`
	ReviewedAt        *int64        `json:"reviewedAt"        keel:"reviewed_at"`
	SettlementLineID  *pulid.ID     `json:"settlementLineId"  keel:"settlement_line_id"`
	Version           int64         `json:"version"           keel:"version,version"`
	CreatedAt         int64         `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64         `json:"updatedAt"         keel:"updated_at,autoupdate"`

	Worker     *worker.Worker   `json:"worker,omitempty"     keel:"belongs:worker_id"`
	PayCode    *paycode.PayCode `json:"payCode,omitempty"    keel:"belongs:pay_code_id"`
	ReviewedBy *tenant.User     `json:"reviewedBy,omitempty" keel:"belongs:reviewed_by_id"`
}

func (e *Expense) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.WorkerID,
			validation.Required.Error("Worker is required"),
		),
		validation.Field(&e.Description,
			validation.Required.Error("Description is required"),
			validation.Length(1, 255).Error("Description must be between 1 and 255 characters"),
		),
		validation.Field(&e.SubmittedByUserID,
			validation.Required.Error("Submitting user is required"),
		),
		validation.Field(&e.IncurredDate,
			validation.Required.Error("Incurred date is required"),
		),
	))

	if e.AmountMinor <= 0 {
		multiErr.Add("amountMinor", errortypes.ErrInvalid, "Amount must be greater than zero")
	}
	if !e.Status.IsValid() {
		multiErr.Add(
			"status",
			errortypes.ErrInvalid,
			"Status must be Pending, Approved, Rejected, Reimbursed, or Cancelled",
		)
	}
}

func (e *Expense) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dexp",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
		},
	}
}

func (e *Expense) GetID() pulid.ID { return e.ID }

func (e *Expense) GetCreatedAt() int64 { return e.CreatedAt }

func (e *Expense) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *Expense) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *Expense) GetTableName() string { return "driver_expenses" }

func (e *Expense) BeforeCreate(ctx context.Context) error {
	if e.Status == "" {
		e.Status = ExpenseStatusPending
	}
	if e.CurrencyCode == "" {
		e.CurrencyCode = money.DefaultCurrencyCode
	}

	return nil
}
