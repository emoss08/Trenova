package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccountingControl struct {
	TimeStampedModel
	BusinessUnitID               uuid.UUID                  `gorm:"type:uuid;not null" json:"businessUnitId"`
	BusinessUnit                 BusinessUnit               `json:"-" validate:"omitempty"`
	OrganizationID               uuid.UUID                  `gorm:"type:uuid;not null;" json:"organizationId"`
	AutoCreateJournalEntries     bool                       `json:"autoCreateJournalEntries" gorm:"type:boolean;not null;default:false"`
	JournalEntryCriteria         *AutomaticJournalEntryType `json:"journalEntryCriteria" gorm:"type:varchar(50);default:'ON_SHIPMENT_BILL'" validate:"omitempty,oneof=ON_SHIPMENT_BILL ON_RECEIPT_OF_PAYMENT ON_EXPENSE_RECOGNITION"`
	RestrictManualJournalEntries bool                       `json:"restrictManualJournalEntries" gorm:"type:boolean;not null;default:false"`
	RequireJournalEntryApporval  bool                       `json:"requireJournalEntryApporval" gorm:"type:boolean;not null;default:false" validate:"required"`
	EnableRecNotifications       bool                       `json:"enableRecNotifications" gorm:"type:boolean;not null;default:true" validate:"required"`
	RecThreshold                 int64                      `json:"recThreshold" gorm:"type:int;not null;default:50" validate:"required"`
	RecThresholdAction           ThresholdActiontype        `json:"recThresholdAction" gorm:"type:ac_threshold_action_type;not null;default:'HALT'" validate:"required,oneof=HALT WARN"`
	DefaultRevenueAccount        *GeneralLedgerAccount      `json:"-" gorm:"foreignKey:DefaultRevenueAccountID;references:ID" validate:"omitempty"`
	DefaultRevenueAccountID      *uuid.UUID                 `json:"defaultRevenueAccountId" gorm:"type:uuid" validate:"omitempty"`
	DefaultExpenseAccount        *GeneralLedgerAccount      `json:"-" gorm:"foreignKey:DefaultExpenseAccountID;references:ID" validate:"omitempty"`
	DefaultExpenseAccountID      *uuid.UUID                 `json:"defaultExpenseAccountId" gorm:"type:uuid" validate:"omitempty"`
	HaltOnPendingRec             bool                       `json:"haltOnPendingRec" gorm:"type:boolean;not null;default:false" validate:"required"`
}

func (ac *AccountingControl) validateAccountingControl() error {
	if ac.DefaultExpenseAccountID != nil && ac.DefaultExpenseAccount.AccountType != Exp {
		return errors.New("default expense account must be an expense account")
	}

	if ac.DefaultRevenueAccountID != nil && ac.DefaultRevenueAccount.AccountType != Rev {
		return errors.New("default revenue account must be a revenue account")
	}

	return nil
}

func (ac *AccountingControl) BeforeCreate(tx *gorm.DB) error {
	return ac.validateAccountingControl()
}

func (ac *AccountingControl) BeforeUpdate(tx *gorm.DB) error {
	return ac.validateAccountingControl()
}

type RevenueCode struct {
	BaseModel
	Code             string               `json:"code" gorm:"type:varchar(4);not null;" validate:"required,max=4"`
	Description      string               `json:"description" gorm:"type:varchar(100);not null;" validate:"required,max=100"`
	ExpenseAccount   GeneralLedgerAccount `json:"-" gorm:"foreignKey:ExpenseAccountID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" validate:"omitempty"`
	ExpenseAccountID *uuid.UUID           `json:"expenseAccountId" gorm:"type:uuid"`
	RevenueAccount   GeneralLedgerAccount `json:"-" gorm:"foreignKey:RevenueAccountID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" validate:"omitempty"`
	RevenueAccountID *uuid.UUID           `json:"revenueAccountId" gorm:"type:uuid"`
}

func (rc *RevenueCode) validateRevenueCode() error {
	if rc.ExpenseAccountID != nil && rc.ExpenseAccount.AccountType != Exp {
		return errors.New("expense account must be an expense account")
	}

	if rc.ExpenseAccountID != nil && rc.RevenueAccount.AccountType != Rev {
		return errors.New("revenue account must be a revenue account")
	}

	return nil
}

func (rc *RevenueCode) GetRevenueCodesByOrgID(db *gorm.DB, orgID uuid.UUID, offset, limit int) ([]RevenueCode, int64, error) {
	var revenueCodes []RevenueCode
	var totalRows int64

	if err := db.Model(&RevenueCode{}).Where("organization_id = ?", orgID).Count(&totalRows).Error; err != nil {
		return revenueCodes, 0, err
	}

	if err := db.Model(&RevenueCode{}).Where("organization_id = ?", orgID).Offset(offset).Limit(limit).Find(&revenueCodes).Error; err != nil {
		return revenueCodes, 0, err
	}

	return revenueCodes, totalRows, nil
}

func (rc *RevenueCode) BeforeCreate(tx *gorm.DB) error {
	if rc.Code != "" {
		rc.Code = strings.ToUpper(rc.Code)
	}

	return rc.validateRevenueCode()
}

type DivisionCode struct {
	BaseModel
	Status           StatusType           `json:"status" gorm:"type:status_type;not null;default:'A'" validate:"required,len=1,oneof=A I"`
	Code             string               `json:"code" gorm:"type:varchar(4);not null;" validate:"required,max=4"`
	Description      string               `json:"description" gorm:"type:varchar(100);not null;" validate:"required,max=100"`
	CashAccount      GeneralLedgerAccount `json:"-" gorm:"foreignKey:CashAccountID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CashAccountID    *uuid.UUID           `json:"cashAccountID" gorm:"type:uuid;"`
	ApAccount        GeneralLedgerAccount `json:"-" gorm:"foreignKey:ApAccountID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	ApAccountID      *uuid.UUID           `json:"apAccountID" gorm:"type:uuid;"`
	ExpenseAccount   GeneralLedgerAccount `json:"-" gorm:"foreignKey:ExpenseAccountID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	ExpenseAccountID *uuid.UUID           `json:"expenseAccountID" gorm:"type:uuid;"`
}

func (dc *DivisionCode) validateDivisionCode() error {
	if dc.CashAccount.AccountClass != Cash {
		return errors.New("cash account must be a cash account")
	}

	if dc.ExpenseAccount.AccountType != Exp {
		return errors.New("expense account must be an expense account")
	}

	if dc.ApAccount.AccountClass != Ap {
		return errors.New("ap account must be an ap account")
	}

	return nil
}

func (dc *DivisionCode) BeforeCreate(tx *gorm.DB) error {
	if dc.CashAccount.AccountClass != Cash {
		return errors.New("cash account must be a cash account")
	}

	return dc.validateDivisionCode()
}

type GeneralLedgerAccount struct {
	BaseModel
	Status         StatusType            `json:"status" gorm:"type:status_type;not null;default:'A'" validate:"required,len=1,oneof=A I"`
	AccountNumber  string                `json:"accountNumber" gorm:"type:varchar(7);not null;" validate:"required,max=7"`
	AccountType    AcAccountType         `json:"accountType" gorm:"type:ac_account_type;not null" validate:"required,oneof=ASSET LIABILITY EQUITY REVENUE EXPENSE"`
	CashFlowType   CashFlowType          `json:"cashFlowType" gorm:"type:ac_cash_flow_type;" validate:"omitempty,oneof=OPERATING INVESTING FINANCING"`
	AccountSubType AccountSubType        `json:"accountSubType" gorm:"type:ac_account_sub_type;" validate:"omitempty,oneof=CURRENT_ASSET FIXED_ASSET OTHER_ASSET CURRENT_LIABILITY LONG_TERM_LIABILITY EQUITY REVENUE COST_OF_GOODS_SOLD EXPENSE OTHER_INCOME OTHER_EXPENSE"`
	AccountClass   AccountClassification `json:"accountClass" gorm:"type:ac_account_classification;" validate:"omitempty,oneof=BANK CASH ACCOUNTS_RECEIVABLE ACCOUNTS_PAYABLE INVENTORY OTHER_CURRENT_ASSET FIXED_ASSET"`
	Balance        *float64              `json:"balance" gorm:"type:numeric(20,2);" validate:"omitempty"`
	IsReconciled   bool                  `json:"isReconciled" gorm:"type:boolean;not null;default:false" validate:"required"`
	DateOpened     time.Time             `json:"dateOpened" gorm:"type:date" validate:"required"`
	DateClosed     *time.Time            `json:"dateClosed" gorm:"type:date" validate:"omitempty"`
	Notes          *string               `json:"notes" gorm:"type:text" validate:"omitempty"`
	IsTaxRelevant  bool                  `json:"isTaxRelevant" gorm:"type:boolean;not null;default:false" validate:"required"`
	InterestRate   *float64              `json:"interestRate" gorm:"type:numeric(5,2)" validate:"omitempty"`
	Tag            []*Tag                `json:"tag" gorm:"many2many:general_ledger_account_tags;"`
	Test           *string               `json:"test" gorm:"type:text" validate:"omitempty"`
}

func (gla *GeneralLedgerAccount) BeforeCreate(tx *gorm.DB) error {
	if gla.DateOpened.IsZero() {
		gla.DateOpened = time.Now()
	}

	return nil
}

type Tag struct {
	BaseModel
	Name        string  `json:"name" gorm:"type:varchar(50);not null;" validate:"required,max=50"`
	Description *string `json:"description" gorm:"type:text;not null;" validate:"omitempty,max=255"`
}
