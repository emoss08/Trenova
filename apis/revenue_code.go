package handlers

import (
	"github.com/emoss08/trenova/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RevenueCodeHandler struct {
	DB *gorm.DB
}

func (s *RevenueCodeHandler) GetAll(orgID, buID uuid.UUID, offset, limit int) ([]models.RevenueCode, int64, error) {
	var rc models.RevenueCode

	return rc.FetchRevenueCodesForOrg(s.DB, orgID, buID, offset, limit)
}

func (s *RevenueCodeHandler) GetByID(orgID, buID uuid.UUID, id string) (models.RevenueCode, error) {
	var rc models.RevenueCode

	return rc.FetchRevenueCodeDetails(s.DB, orgID, buID, id)
}

func (s *RevenueCodeHandler) Create(orgID, buID uuid.UUID, revenueCode models.RevenueCode) error {
	revenueCode.BusinessUnitID = buID
	revenueCode.OrganizationID = orgID

	return s.DB.Create(&revenueCode).Error
}

func (s *RevenueCodeHandler) Update(orgID, buID uuid.UUID, id string, revenueCode models.RevenueCode) error {
	revenueCode.BusinessUnitID = buID
	revenueCode.OrganizationID = orgID

	return s.DB.Model(&revenueCode).
		Where("id = ? AND organization_id = ? AND business_unit_id = ?", id, orgID, buID).
		Updates(&revenueCode).Error
}
