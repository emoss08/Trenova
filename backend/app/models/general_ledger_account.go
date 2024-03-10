package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GeneralLedgerAccount struct {
	TimeStampedModel
	OrganizationID uuid.UUID             `json:"organizationId" gorm:"type:uuid;not null;uniqueIndex:idx_gl_account_number_organization_id" validate:"required"`
	BusinessUnitID uuid.UUID             `json:"businessUnitId" gorm:"type:uuid;not null"                                                   validate:"required"`
	Organization   Organization          `json:"-"                                                                                          validate:"omitempty"`
	BusinessUnit   BusinessUnit          `json:"-"                                                                                          validate:"omitempty"`
	Status         StatusType            `json:"status"         gorm:"type:status_type;not null;default:'A'"                                validate:"required,len=1,oneof=A I"`
	AccountNumber  string                `json:"accountNumber"  gorm:"type:varchar(7);not null;uniqueIndex:idx_gl_account_number_organization_id,expression:lower(account_number)" validate:"required,max=7"`
	AccountType    AcAccountType         `json:"accountType"    gorm:"type:ac_account_type;not null"       validate:"required,oneof=ASSET LIABILITY EQUITY REVENUE EXPENSE"`
	CashFlowType   CashFlowType          `json:"cashFlowType"   gorm:"type:ac_cash_flow_type;"             validate:"omitempty,oneof=OPERATING INVESTING FINANCING"`
	AccountSubType AccountSubType        `json:"accountSubType" gorm:"type:ac_account_sub_type;"           validate:"omitempty,oneof=CURRENT_ASSET FIXED_ASSET OTHER_ASSET CURRENT_LIABILITY LONG_TERM_LIABILITY EQUITY REVENUE COST_OF_GOODS_SOLD EXPENSE OTHER_INCOME OTHER_EXPENSE"`
	AccountClass   AccountClassification `json:"accountClass"   gorm:"type:ac_account_classification;"     validate:"omitempty,oneof=BANK CASH ACCOUNTS_RECEIVABLE ACCOUNTS_PAYABLE INVENTORY OTHER_CURRENT_ASSET FIXED_ASSET"`
	Balance        *float64              `json:"balance"        gorm:"type:numeric(20,2);"                 validate:"omitempty"`
	InterestRate   *float64              `json:"interestRate"   gorm:"type:numeric(5,2)"                   validate:"omitempty"`
	DateOpened     time.Time             `json:"dateOpened"     gorm:"type:date"                           validate:"required"`
	DateClosed     *time.Time            `json:"dateClosed"     gorm:"type:date"                           validate:"omitempty"`
	Notes          *string               `json:"notes"          gorm:"type:text"                           validate:"omitempty"`
	IsTaxRelevant  bool                  `json:"isTaxRelevant"  gorm:"type:boolean;not null;default:false" validate:"required"`
	IsReconciled   bool                  `json:"isReconciled"   gorm:"type:boolean;not null;default:false" validate:"required"`
	Tag            []*Tag                `json:"tag"            gorm:"many2many:general_ledger_account_tags;" validate:"omitempty"`
}

func (gla *GeneralLedgerAccount) BeforeCreate(_ *gorm.DB) error {
	if gla.DateOpened.IsZero() {
		gla.DateOpened = time.Now()
	}

	return nil
}
