package handlers

import (
	"github.com/emoss08/trenova/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailProfileHandler struct {
	DB *gorm.DB
}

func (s *EmailProfileHandler) GetAll(orgID, buID uuid.UUID, offset, limit int) ([]models.EmailProfile, int64, error) {
	var ep models.EmailProfile

	return ep.FetchEmailProfilesForOrg(s.DB, orgID, buID, offset, limit)
}

func (s *EmailProfileHandler) GetByID(orgID, buID uuid.UUID, id string) (models.EmailProfile, error) {
	var ep models.EmailProfile

	return ep.FetchEmailProfileDetails(s.DB, orgID, buID, id)
}

func (s *EmailProfileHandler) Create(orgID, buID uuid.UUID, emailProfile models.EmailProfile) error {
	emailProfile.BusinessUnitID = buID
	emailProfile.OrganizationID = orgID

	return s.DB.Create(&emailProfile).Error
}

func (s *EmailProfileHandler) Update(orgID, buID uuid.UUID, id string, emailProfile models.EmailProfile) error {
	emailProfile.BusinessUnitID = buID
	emailProfile.OrganizationID = orgID

	return s.DB.Model(&emailProfile).
		Where("id = ? AND organization_id = ? AND business_unit_id = ?", id, orgID, buID).
		Updates(&emailProfile).Error
}
