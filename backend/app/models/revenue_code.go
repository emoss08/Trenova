package models

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RevenueCode struct {
	TimeStampedModel
	OrganizationID   uuid.UUID            `gorm:"type:uuid;not null;uniqueIndex:idx_rev_code_organization_id" json:"organizationId" validate:"required"`
	BusinessUnitID   uuid.UUID            `gorm:"type:uuid;not null;index"                                    json:"businessUnitId" validate:"required"`
	Organization     Organization         `json:"-" validate:"omitempty"`
	BusinessUnit     BusinessUnit         `json:"-" validate:"omitempty"`
	Code             string               `gorm:"type:varchar(4);not null;uniqueIndex:idx_division_code_organization_id_code,expression:lower(code)" json:"code"        validate:"required,max=4"`
	Description      string               `gorm:"type:varchar(100);not null;"                                                                        json:"description" validate:"required,max=100"`
	ExpenseAccountID *uuid.UUID           `gorm:"type:uuid;index"                                                                                    json:"expenseAccountId"`
	RevenueAccountID *uuid.UUID           `gorm:"type:uuid;index"                                                                                    json:"revenueAccountId"`
	ExpenseAccount   GeneralLedgerAccount `gorm:"foreignKey:ExpenseAccountID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"                          json:"-" validate:"omitempty"`
	RevenueAccount   GeneralLedgerAccount `gorm:"foreignKey:RevenueAccountID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"                          json:"-" validate:"omitempty"`
}

func (rc *RevenueCode) validateRevenueCode() error {
	if rc.ExpenseAccountID != nil && rc.ExpenseAccount.AccountType != AccountTypeExpense {
		return errExpenseAccountMustBeExpense
	}

	if rc.RevenueAccountID != nil && rc.RevenueAccount.AccountType != AccountTypeRevenue {
		return errRevenueAccountMustBeRevenue
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

func (rc *RevenueCode) BeforeCreate(_ *gorm.DB) error {
	if rc.Code != "" {
		rc.Code = strings.ToUpper(rc.Code)
	}

	return rc.validateRevenueCode()
}
