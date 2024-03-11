package models

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DivisionCode struct {
	TimeStampedModel
	OrganizationID   uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_division_code_organization_id_code" json:"organizationId" validate:"required"`
	BusinessUnitID   uuid.UUID    `gorm:"type:uuid;not null;index"                                              json:"businessUnitId" validate:"required"`
	Organization     Organization `json:"-" validate:"omitempty"`
	BusinessUnit     BusinessUnit `json:"-" validate:"omitempty"`
	Status           StatusType   `gorm:"type:status_type;not null;default:'A'"                                                              json:"status"      validate:"required,len=1,oneof=A I"`
	Code             string       `gorm:"type:varchar(4);not null;uniqueIndex:idx_division_code_organization_id_code,expression:lower(code)" json:"code"        validate:"required,max=4"`
	Description      string       `gorm:"type:varchar(100);not null;"                                                                        json:"description" validate:"required,max=100"`
	CashAccountID    *uuid.UUID   `gorm:"type:uuid;index;"                                                                                   json:"cashAccountId" `
	ApAccountID      *uuid.UUID   `gorm:"type:uuid;index;"                                                                                   json:"apAccountId" `
	ExpenseAccountID *uuid.UUID   `gorm:"type:uuid;index;"                                                                                   json:"expenseAccountId" `
	CashAccount      *GeneralLedgerAccount
	ApAccount        *GeneralLedgerAccount
	ExpenseAccount   *GeneralLedgerAccount
}

var (
	errCashAccountMustBeCash       = errors.New("cash account must be a cash account")
	errExpenseAccountMustBeExpense = errors.New("expense account must be an expense account")
	errRevenueAccountMustBeRevenue = errors.New("revenue account must be a revenue account")
	errApAccountMustBeAp           = errors.New("ap account must be an ap account")
)

func (dc *DivisionCode) validateDivisionCode() error {
	if dc.CashAccount.AccountClass != AccountClassificationCash {
		return errCashAccountMustBeCash
	}

	if dc.ExpenseAccount.AccountType != AccountTypeExpense {
		return errExpenseAccountMustBeExpense
	}

	if dc.ApAccount.AccountClass != AccountClassificationAP {
		return errApAccountMustBeAp
	}

	return nil
}

func (dc *DivisionCode) BeforeCreate(_ *gorm.DB) error {
	if dc.CashAccount.AccountClass != AccountClassificationCash {
		return errCashAccountMustBeCash
	}

	return dc.validateDivisionCode()
}
