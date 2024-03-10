package models

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccountingControl struct {
	TimeStampedModel
	BusinessUnitID               uuid.UUID                  `json:"businessUnitId" gorm:"type:uuid;not null;index"`
	BusinessUnit                 BusinessUnit               `json:"-" validate:"omitempty"`
	OrganizationID               uuid.UUID                  `json:"organizationId" gorm:"type:uuid;not null;index"`
	AutoCreateJournalEntries     bool                       `json:"autoCreateJournalEntries" gorm:"type:boolean;not null;default:false"`
	JournalEntryCriteria         *AutomaticJournalEntryType `json:"journalEntryCriteria" gorm:"type:varchar(50);default:'ON_SHIPMENT_BILL'" validate:"omitempty,oneof=ON_SHIPMENT_BILL ON_RECEIPT_OF_PAYMENT ON_EXPENSE_RECOGNITION"`
	RestrictManualJournalEntries bool                       `json:"restrictManualJournalEntries" gorm:"type:boolean;not null;default:false"`
	RequireJournalEntryApporval  bool                       `json:"requireJournalEntryApporval" gorm:"type:boolean;not null;default:false" validate:"required"`
	EnableRecNotifications       bool                       `json:"enableRecNotifications" gorm:"type:boolean;not null;default:true" validate:"required"`
	RecThreshold                 int64                      `json:"recThreshold" gorm:"type:int;not null;default:50" validate:"required"`
	RecThresholdAction           ThresholdActiontype        `json:"recThresholdAction" gorm:"type:ac_threshold_action_type;not null;default:'HALT'" validate:"required,oneof=HALT WARN"`
	DefaultRevenueAccountID      *uuid.UUID                 `json:"defaultRevenueAccountId" gorm:"type:uuid" validate:"omitempty"`
	DefaultRevenueAccount        *GeneralLedgerAccount      `json:"-" gorm:"foreignKey:DefaultRevenueAccountID;references:ID" validate:"omitempty"`
	DefaultExpenseAccountID      *uuid.UUID                 `json:"defaultExpenseAccountId" gorm:"type:uuid" validate:"omitempty"`
	DefaultExpenseAccount        *GeneralLedgerAccount      `json:"-" gorm:"foreignKey:DefaultExpenseAccountID;references:ID" validate:"omitempty"`
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

