package accounting

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*JournalEntryLine)(nil)
	_ domain.Validatable        = (*JournalEntryLine)(nil)
	_ framework.TenantedEntity  = (*JournalEntryLine)(nil)
)

type JournalEntryLine struct {
	bun.BaseModel `bun:"table:journal_entry_lines,alias:jel" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	JournalEntryID pulid.ID `json:"journalEntryId" bun:"journal_entry_id,type:VARCHAR(100),notnull"`
	GLAccountID    pulid.ID `json:"glAccountId"    bun:"gl_account_id,type:VARCHAR(100),notnull"`

	// Line Details
	LineNumber  int32  `json:"lineNumber"  bun:"line_number,type:INTEGER,notnull"`
	Description string `json:"description" bun:"description,type:TEXT,notnull"`

	// Amounts (in cents)
	DebitAmount  int64 `json:"debitAmount"  bun:"debit_amount,type:BIGINT,notnull,default:0"`
	CreditAmount int64 `json:"creditAmount" bun:"credit_amount,type:BIGINT,notnull,default:0"`

	// Dimensional Analysis (optional)
	DepartmentID *pulid.ID `json:"departmentId" bun:"department_id,type:VARCHAR(100),nullzero"`
	ProjectID    *pulid.ID `json:"projectId"    bun:"project_id,type:VARCHAR(100),nullzero"`
	LocationID   *pulid.ID `json:"locationId"   bun:"location_id,type:VARCHAR(100),nullzero"`
	CustomerID   *pulid.ID `json:"customerId"   bun:"customer_id,type:VARCHAR(100),nullzero"`

	// Reference Information
	ReferenceNumber string    `json:"referenceNumber" bun:"reference_number,type:VARCHAR(100),nullzero"`
	ReferenceType   string    `json:"referenceType"   bun:"reference_type,type:VARCHAR(50),nullzero"`
	ReferenceID     *pulid.ID `json:"referenceId"     bun:"reference_id,type:VARCHAR(100),nullzero"`

	// Tax Information (for future tax tracking)
	TaxCode   string `json:"taxCode"   bun:"tax_code,type:VARCHAR(20),nullzero"`
	TaxAmount int64  `json:"taxAmount" bun:"tax_amount,type:BIGINT,notnull,default:0"` // In cents

	// Reconciliation
	IsReconciled   bool      `json:"isReconciled"   bun:"is_reconciled,type:BOOLEAN,notnull,default:false"`
	ReconciledAt   *int64    `json:"reconciledAt"   bun:"reconciled_at,type:BIGINT,nullzero"`
	ReconciledByID *pulid.ID `json:"reconciledById" bun:"reconciled_by_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	JournalEntry *JournalEntry        `json:"journalEntry,omitempty" bun:"rel:belongs-to,join:journal_entry_id=id"`
	GLAccount    *GLAccount           `json:"glAccount,omitempty"    bun:"rel:belongs-to,join:gl_account_id=id"`
	ReconciledBy *tenant.User         `json:"reconciledBy,omitempty" bun:"rel:belongs-to,join:reconciled_by_id=id"`
}

func (jel *JournalEntryLine) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		jel,
		validation.Field(&jel.JournalEntryID,
			validation.Required.Error("Journal entry is required"),
		),
		validation.Field(&jel.GLAccountID,
			validation.Required.Error("GL account is required"),
		),
		validation.Field(&jel.LineNumber,
			validation.Required.Error("Line number is required"),
			validation.Min(1).Error("Line number must be at least 1"),
		),
		validation.Field(&jel.Description,
			validation.Required.Error("Description is required"),
			validation.Length(1, 500).Error("Description must be between 1 and 500 characters"),
		),
		validation.Field(&jel.DebitAmount,
			validation.Min(int64(0)).Error("Debit amount cannot be negative"),
		),
		validation.Field(&jel.CreditAmount,
			validation.Min(int64(0)).Error("Credit amount cannot be negative"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	// Business rule: A line must have either a debit or credit, but not both
	if jel.DebitAmount > 0 && jel.CreditAmount > 0 {
		multiErr.Add(
			"debitAmount",
			errortypes.ErrInvalid,
			"A line cannot have both debit and credit amounts",
		)
		multiErr.Add(
			"creditAmount",
			errortypes.ErrInvalid,
			"A line cannot have both debit and credit amounts",
		)
	}

	// Business rule: A line must have either a debit or credit amount
	if jel.DebitAmount == 0 && jel.CreditAmount == 0 {
		multiErr.Add(
			"debitAmount",
			errortypes.ErrInvalid,
			"A line must have either a debit or credit amount",
		)
		multiErr.Add(
			"creditAmount",
			errortypes.ErrInvalid,
			"A line must have either a debit or credit amount",
		)
	}
}

func (jel *JournalEntryLine) GetID() string {
	return jel.ID.String()
}

func (jel *JournalEntryLine) GetTableName() string {
	return "journal_entry_lines"
}

func (jel *JournalEntryLine) GetOrganizationID() pulid.ID {
	return jel.OrganizationID
}

func (jel *JournalEntryLine) GetBusinessUnitID() pulid.ID {
	return jel.BusinessUnitID
}

func (jel *JournalEntryLine) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if jel.ID.IsNil() {
			jel.ID = pulid.MustNew("jel_")
		}

		jel.CreatedAt = now
	case *bun.UpdateQuery:
		jel.UpdatedAt = now
	}

	return nil
}

// IsDebit returns true if this line is a debit
func (jel *JournalEntryLine) IsDebit() bool {
	return jel.DebitAmount > 0
}

// IsCredit returns true if this line is a credit
func (jel *JournalEntryLine) IsCredit() bool {
	return jel.CreditAmount > 0
}

// GetAmount returns the non-zero amount (either debit or credit)
func (jel *JournalEntryLine) GetAmount() int64 {
	if jel.IsDebit() {
		return jel.DebitAmount
	}
	return jel.CreditAmount
}
