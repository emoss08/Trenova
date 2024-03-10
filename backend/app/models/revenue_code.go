package models

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RevenueCode struct {
	TimeStampedModel
	OrganizationID   uuid.UUID            `json:"organizationId" gorm:"type:uuid;not null;uniqueIndex:idx_rev_code_organization_id" validate:"required"`
	Organization     Organization         `json:"-" validate:"omitempty"`
	BusinessUnitID   uuid.UUID            `json:"businessUnitId" gorm:"type:uuid;not null;index" validate:"required"`
	BusinessUnit     BusinessUnit         `json:"-" validate:"omitempty"`
	Code             string               `json:"code" gorm:"type:varchar(4);not null;uniqueIndex:idx_division_code_organization_id_code,expression:lower(code)" validate:"required,max=4"`
	Description      string               `json:"description" gorm:"type:varchar(100);not null;" validate:"required,max=100"`
	ExpenseAccount   GeneralLedgerAccount `json:"-" gorm:"foreignKey:ExpenseAccountID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" validate:"omitempty"`
	ExpenseAccountID *uuid.UUID           `json:"expenseAccountId" gorm:"type:uuid;index"`
	RevenueAccount   GeneralLedgerAccount `json:"-" gorm:"foreignKey:RevenueAccountID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" validate:"omitempty"`
	RevenueAccountID *uuid.UUID           `json:"revenueAccountId" gorm:"type:uuid;index"`
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

func (rc *RevenueCode) FetchRevenueCodesForOrg(db *gorm.DB, orgID, buID uuid.UUID, offset, limit int) ([]RevenueCode, int64, error) {
	var revenueCodes []RevenueCode
	var totalRows int64

	if err := db.Model(&RevenueCode{}).Where("organization_id = ? AND business_unit_id = ?", orgID, buID).Count(&totalRows).Error; err != nil {
		return revenueCodes, 0, err
	}

	if err := db.Model(&RevenueCode{}).Where("organization_id = ? AND business_unit_id = ?", orgID, buID).Offset(offset).Limit(limit).Order("created_at desc").Find(&revenueCodes).Error; err != nil {
		return revenueCodes, 0, err
	}

	return revenueCodes, totalRows, nil
}

func (rc *RevenueCode) FetchRevenueCodeDetails(db *gorm.DB, orgID, buID uuid.UUID, id string) (RevenueCode, error) {
	var revenueCode RevenueCode

	if err := db.Model(&RevenueCode{}).Where("organization_id = ? AND id = ? AND business_unit_id = ?", orgID, id, buID).First(&revenueCode).Error; err != nil {
		return revenueCode, err
	}

	return revenueCode, nil
}

func (rc *RevenueCode) BeforeCreate(tx *gorm.DB) error {
	if rc.Code != "" {
		rc.Code = strings.ToUpper(rc.Code)
	}

	return rc.validateRevenueCode()
}
