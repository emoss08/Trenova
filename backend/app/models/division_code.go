package models

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DivisionCode struct {
	TimeStampedModel
	OrganizationID   uuid.UUID    `json:"organizationId" gorm:"type:uuid;not null;uniqueIndex:idx_division_code_organization_id_code" validate:"required"`
	Organization     Organization `json:"-" validate:"omitempty"`
	BusinessUnitID   uuid.UUID    `json:"businessUnitId" gorm:"type:uuid;not null;index" validate:"required"`
	BusinessUnit     BusinessUnit `json:"-" validate:"omitempty"`
	Status           StatusType   `json:"status" gorm:"type:status_type;not null;default:'A'" validate:"required,len=1,oneof=A I"`
	Code             string       `json:"code" gorm:"type:varchar(4);not null;uniqueIndex:idx_division_code_organization_id_code,expression:lower(code)" validate:"required,max=4"`
	Description      string       `json:"description" gorm:"type:varchar(100);not null;" validate:"required,max=100"`
	CashAccountID    *uuid.UUID   `json:"cashAccountID" gorm:"type:uuid;index;"`
	CashAccount      *GeneralLedgerAccount
	ApAccountID      *uuid.UUID `json:"apAccountID" gorm:"type:uuid;index;"`
	ApAccount        *GeneralLedgerAccount
	ExpenseAccountID *uuid.UUID `json:"expenseAccountID" gorm:"type:uuid;index;"`
	ExpenseAccount   *GeneralLedgerAccount
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
