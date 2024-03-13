package handlers

import (
	"github.com/emoss08/trenova/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QualifierCodeHandler struct {
	DB *gorm.DB
}

func (s *QualifierCodeHandler) GetAll(orgID, buID uuid.UUID, offset, limit int) ([]models.QualifierCode, int64, error) {
	var qc models.QualifierCode

	return qc.FetchQualifierCodesForOrg(s.DB, orgID, buID, offset, limit)
}

func (s *QualifierCodeHandler) GetByID(orgID, buID uuid.UUID, id string) (models.QualifierCode, error) {
	var qc models.QualifierCode

	return qc.FetchQualifierCodeDetails(s.DB, orgID, buID, id)
}

func (s *QualifierCodeHandler) Create(orgID, buID uuid.UUID, qualifierCode models.QualifierCode) error {
	qualifierCode.BusinessUnitID = buID
	qualifierCode.OrganizationID = orgID

	return s.DB.Create(&qualifierCode).Error
}

func (s *QualifierCodeHandler) Update(orgID, buID uuid.UUID, id string, qualifierCode models.QualifierCode) error {
	qualifierCode.BusinessUnitID = buID
	qualifierCode.OrganizationID = orgID

	return s.DB.Model(&qualifierCode).
		Where("id = ? AND organization_id = ? AND business_unit_id = ?", id, orgID, buID).
		Updates(&qualifierCode).Error
}
