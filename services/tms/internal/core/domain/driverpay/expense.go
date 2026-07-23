package driverpay

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Expense)(nil)
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
	bun.BaseModel             `bun:"table:driver_expenses,alias:dexp" json:"-"`
	pagination.CursorValueSet `bun:",embed"                           json:"-"`

	ID                pulid.ID      `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID      `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID      `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID          pulid.ID      `json:"workerId"          bun:"worker_id,type:VARCHAR(100),notnull"`
	ShipmentID        *pulid.ID     `json:"shipmentId"        bun:"shipment_id,type:VARCHAR(100),nullzero"`
	PayCodeID         *pulid.ID     `json:"payCodeId"         bun:"pay_code_id,type:VARCHAR(100),nullzero"`
	Status            ExpenseStatus `json:"status"            bun:"status,type:VARCHAR(20),notnull,default:'Pending'"`
	AmountMinor       int64         `json:"amountMinor"       bun:"amount_minor,type:BIGINT,notnull"`
	CurrencyCode      string        `json:"currencyCode"      bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	Description       string        `json:"description"       bun:"description,type:VARCHAR(255),notnull"`
	IncurredDate      int64         `json:"incurredDate"      bun:"incurred_date,type:BIGINT,notnull"`
	ReceiptDocumentID *pulid.ID     `json:"receiptDocumentId" bun:"receipt_document_id,type:VARCHAR(100),nullzero"`
	SubmittedByUserID pulid.ID      `json:"submittedByUserId" bun:"submitted_by_user_id,type:VARCHAR(100),notnull"`
	ReviewNote        string        `json:"reviewNote"        bun:"review_note,type:TEXT,nullzero"`
	ReviewedByID      *pulid.ID     `json:"reviewedById"      bun:"reviewed_by_id,type:VARCHAR(100),nullzero"`
	ReviewedAt        *int64        `json:"reviewedAt"        bun:"reviewed_at,type:BIGINT,nullzero"`
	SettlementLineID  *pulid.ID     `json:"settlementLineId"  bun:"settlement_line_id,type:VARCHAR(100),nullzero"`
	Version           int64         `json:"version"           bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt         int64         `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64         `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker     *worker.Worker `json:"worker,omitempty"     bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	PayCode    *PayCode       `json:"payCode,omitempty"    bun:"rel:belongs-to,join:pay_code_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	ReviewedBy *tenant.User   `json:"reviewedBy,omitempty" bun:"rel:belongs-to,join:reviewed_by_id=id"`
}

func (e *Expense) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(e,
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
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

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
			{Name: "description", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
	}
}

func (e *Expense) GetID() pulid.ID { return e.ID }

func (e *Expense) GetCreatedAt() int64 { return e.CreatedAt }

func (e *Expense) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *Expense) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *Expense) GetTableName() string { return "driver_expenses" }

func (e *Expense) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("dexp_")
		}
		e.CreatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}
	return nil
}
