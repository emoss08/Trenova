package driverpay

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
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
	bun.BaseModel `bun:"table:driver_expenses,alias:dexp" json:"-"`
	keel.Table    `keel:"driver_expenses"`

	ID                pulid.ID      `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:dexp_"`
	BusinessUnitID    pulid.ID      `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID      `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	WorkerID          pulid.ID      `json:"workerId"          bun:"worker_id,type:VARCHAR(100),notnull" keel:"worker_id"`
	ShipmentID        *pulid.ID     `json:"shipmentId"        bun:"shipment_id,type:VARCHAR(100),nullzero" keel:"shipment_id"`
	PayCodeID         *pulid.ID     `json:"payCodeId"         bun:"pay_code_id,type:VARCHAR(100),nullzero" keel:"pay_code_id"`
	Status            ExpenseStatus `json:"status"            bun:"status,type:VARCHAR(20),notnull,default:'Pending'" keel:"status"`
	AmountMinor       int64         `json:"amountMinor"       bun:"amount_minor,type:BIGINT,notnull" keel:"amount_minor"`
	CurrencyCode      string        `json:"currencyCode"      bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'" keel:"currency_code"`
	Description       string        `json:"description"       bun:"description,type:VARCHAR(255),notnull" keel:"description"`
	IncurredDate      int64         `json:"incurredDate"      bun:"incurred_date,type:BIGINT,notnull" keel:"incurred_date"`
	ReceiptDocumentID *pulid.ID     `json:"receiptDocumentId" bun:"receipt_document_id,type:VARCHAR(100),nullzero" keel:"receipt_document_id"`
	SubmittedByUserID pulid.ID      `json:"submittedByUserId" bun:"submitted_by_user_id,type:VARCHAR(100),notnull" keel:"submitted_by_user_id"`
	ReviewNote        string        `json:"reviewNote"        bun:"review_note,type:TEXT,nullzero" keel:"review_note,nullzero"`
	ReviewedByID      *pulid.ID     `json:"reviewedById"      bun:"reviewed_by_id,type:VARCHAR(100),nullzero" keel:"reviewed_by_id"`
	ReviewedAt        *int64        `json:"reviewedAt"        bun:"reviewed_at,type:BIGINT,nullzero" keel:"reviewed_at"`
	SettlementLineID  *pulid.ID     `json:"settlementLineId"  bun:"settlement_line_id,type:VARCHAR(100),nullzero" keel:"settlement_line_id"`
	Version           int64         `json:"version"           bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt         int64         `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt         int64         `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Worker     *worker.Worker `json:"worker,omitempty"     bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:worker_id"`
	PayCode    *PayCode       `json:"payCode,omitempty"    bun:"rel:belongs-to,join:pay_code_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id" keel:"belongs:pay_code_id"`
	ReviewedBy *tenant.User   `json:"reviewedBy,omitempty" bun:"rel:belongs-to,join:reviewed_by_id=id" keel:"belongs:reviewed_by_id"`
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

func (e *Expense) BeforeCreate(ctx context.Context) error {
	return e.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}
