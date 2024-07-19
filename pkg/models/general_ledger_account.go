// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package models

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/uptrace/bun"
)

type GeneralLedgerAccount struct {
	bun.BaseModel `bun:"table:general_ledger_accounts,alias:gla" json:"-"`

	ID             uuid.UUID                             `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status         property.Status                       `bun:"status,type:status" json:"status"`
	AccountNumber  string                                `bun:"type:VARCHAR(7),notnull" json:"accountNumber" queryField:"true"`
	AccountType    property.GLAccountType                `bun:"type:account_type_enum,notnull" json:"accountType"`
	CashFlowType   *property.GLCashFlowType              `bun:"type:cash_flow_type_enum,nullzero" json:"cashFlowType"`
	AccountSubType *property.GLAccountSubType            `bun:"type:account_sub_type_enum,nullzero" json:"accountSubType"`
	AccountClass   *property.GLAccountClassificationType `bun:"type:account_classification_type_enum,nullzero" json:"accountClass"`
	Balance        float64                               `bun:"type:NUMERIC(14,2),notnull,default:0" json:"balance"`
	InterestRate   float64                               `bun:"type:NUMERIC(5,2),nullzero" json:"interestRate,omitempty"`
	DateClosed     *pgtype.Date                          `bun:",scanonly,nullzero" json:"dateClosed"`
	Notes          string                                `bun:"type:TEXT" json:"notes"`
	IsTaxRelevant  bool                                  `bun:"type:BOOLEAN,default:false" json:"isTaxRelevant"`
	IsReconciled   bool                                  `bun:"type:BOOLEAN,default:false" json:"isReconciled"`
	Version        int64                                 `bun:"type:BIGINT" json:"version"`
	CreatedAt      time.Time                             `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time                             `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	TagIDs         []uuid.UUID `bun:",scanonly" json:"tagIds"`
	BusinessUnitID uuid.UUID   `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID   `bun:"type:uuid,notnull" json:"organizationId"`

	Tags         []*Tag        `bun:"m2m:general_ledger_account_tags,join:GeneralLedgerAccount=Tag" json:"tags"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
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

func (g *GeneralLedgerAccount) BeforeUpdate(_ context.Context) error {
	g.Version++

	return nil
}

func (g *GeneralLedgerAccount) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := g.Version

	if err := g.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(g).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return &validator.BusinessLogicError{
			Message: fmt.Sprintf("Version mismatch. The GeneralLedgerAccount (ID: %s) has been updated by another user. Please refresh and try again.", g.ID),
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
	bun.BaseModel `bun:"table:general_ledger_account_tags" json:"-"`

	GeneralLedgerAccountID uuid.UUID `bun:"general_ledger_account_id,pk,type:uuid" json:"generalLedgerAccountId"`
	TagID                  uuid.UUID `bun:"tag_id,pk,type:uuid" json:"tagId"`

	GeneralLedgerAccount *GeneralLedgerAccount `bun:"rel:belongs-to,join:general_ledger_account_id=id" json:"generalLedgerAccount"`
	Tag                  *Tag                  `bun:"rel:belongs-to,join:tag_id=id" json:"tag"`
}
