package models

import (
	"context"
	"regexp"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/uptrace/bun"
)

type GeneralLedgerAccountPermission string

const (
	// PermissionGeneralLedgerAccountView is the permission to view general ledger account details
	PermissionGeneralLedgerAccountView = GeneralLedgerAccountPermission("generalledgeraccount.view")

	// PermissionGeneralLedgerAccountEdit is the permission to edit general ledger account details
	PermissionGeneralLedgerAccountEdit = GeneralLedgerAccountPermission("generalledgeraccount.edit")

	// PermissionGeneralLedgerAccountAdd is the permission to add a new general ledger account
	PermissionGeneralLedgerAccountAdd = GeneralLedgerAccountPermission("generalledgeraccount.add")

	// PermissionGeneralLedgerAccountDelete is the permission to delete an general ledger account
	PermissionGeneralLedgerAccountDelete = GeneralLedgerAccountPermission("generalledgeraccount.delete")
)

// String returns the string representation of the GeneralLedgerAccountPermission
func (p GeneralLedgerAccountPermission) String() string {
	return string(p)
}

type GeneralLedgerAccount struct {
	bun.BaseModel  `bun:"table:general_ledger_accounts,alias:gla" json:"-"`
	CreatedAt      time.Time                             `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time                             `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID                             `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status         property.Status                       `bun:"status,type:status" json:"status"`
	AccountNumber  string                                `bun:"type:VARCHAR(7),notnull" json:"accountNumber" queryField:"true"`
	AccountType    property.GLAccountType                `bun:"type:account_type_enum,notnull" json:"accountType"`
	CashFlowType   *property.GLCashFlowType              `bun:"type:cash_flow_type_enum,nullzero" json:"cashFlowType"`
	AccountSubType *property.GLAccountSubType            `bun:"type:account_sub_type_enum,nullzero" json:"accountSubType"`
	AccountClass   *property.GLAccountClassificationType `bun:"type:account_classification_type_enum,nullzero" json:"accountClass"`
	Balance        string                                `bun:"type:NUMERIC(14,2),notnull,default:0" json:"balance"`
	InterestRate   string                                `bun:"type:NUMERIC(5,2),nullzero" json:"interestRate,omitempty"`
	DateClosed     *pgtype.Date                          `bun:",scanonly,nullzero" json:"dateClosed"`
	Notes          string                                `bun:"type:TEXT" json:"notes"`
	IsTaxRelevant  bool                                  `bun:"type:BOOLEAN,default:false" json:"isTaxRelevant"`
	IsReconciled   bool                                  `bun:"type:BOOLEAN,default:false" json:"isReconciled"`
	TagIDs         []uuid.UUID                           `bun:",scanonly" json:"tagIds"`
	BusinessUnitID uuid.UUID                             `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID                             `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Tags         []*Tag        `bun:"m2m:general_ledger_account_tags,join:GeneralLedgerAccount=Tag" json:"tags"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (g GeneralLedgerAccount) Validate() error {
	return validation.ValidateStruct(
		&g,
		validation.Field(&g.AccountNumber,
			validation.Required,
			validation.Length(7, 7).Error("account number must be 7 characters"),
			validation.Match(regexp.MustCompile("^[0-9]{4}-[0-9]{2}$")),
		),
		validation.Field(&g.AccountType, validation.Required),
		validation.Field(&g.BusinessUnitID, validation.Required),
		validation.Field(&g.OrganizationID, validation.Required),
	)
}

// ClearTags clears all tags associated with the given general ledger account ID.
func (g GeneralLedgerAccount) ClearTags(ctx context.Context, tx bun.Tx, accountID uuid.UUID) error {
	_, err := tx.NewDelete().
		Model((*GeneralLedgerAccountTag)(nil)).
		Where("general_ledger_account_id = ?", accountID).
		Exec(ctx)
	return err
}

// AssociateTagsByID associates a slice of tags with the given general ledger account ID.
func (g GeneralLedgerAccount) AssociateTagsByID(ctx context.Context, tx bun.Tx, accountID uuid.UUID, tagIDs []uuid.UUID) error {
	for _, tagID := range tagIDs {
		if _, err := tx.NewInsert().
			Model(&GeneralLedgerAccountTag{
				GeneralLedgerAccountID: accountID,
				TagID:                  tagID,
			}).
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

var _ bun.BeforeAppendModelHook = (*GeneralLedgerAccount)(nil)

func (g *GeneralLedgerAccount) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		g.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		g.UpdatedAt = time.Now()
	}
	return nil
}

type GeneralLedgerAccountTag struct {
	bun.BaseModel          `bun:"table:general_ledger_account_tags" json:"-"`
	GeneralLedgerAccountID uuid.UUID `bun:"general_ledger_account_id,pk,type:uuid" json:"generalLedgerAccountId"`
	TagID                  uuid.UUID `bun:"tag_id,pk,type:uuid" json:"tagId"`

	GeneralLedgerAccount *GeneralLedgerAccount `bun:"rel:belongs-to,join:general_ledger_account_id=id" json:"generalLedgerAccount"`
	Tag                  *Tag                  `bun:"rel:belongs-to,join:tag_id=id" json:"tag"`
}
