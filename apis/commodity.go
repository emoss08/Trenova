package handlers

import (
	"github.com/emoss08/trenova/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CommodityHandler struct {
	DB *gorm.DB
}

func (s *CommodityHandler) GetAll(orgID, buID uuid.UUID, offset, limit int) ([]models.Commodity, int64, error) {
	var cm models.Commodity

	return cm.FetchCommoditiesForOrg(s.DB, orgID, buID, offset, limit)
}

func (s *CommodityHandler) GetByID(orgID, buID uuid.UUID, id string) (models.Commodity, error) {
	var cm models.Commodity

	return cm.FetchCommodityDetails(s.DB, orgID, buID, id)
}

func (s *CommodityHandler) Create(orgID, buID uuid.UUID, commodity models.Commodity) error {
	commodity.BusinessUnitID = buID
	commodity.OrganizationID = orgID

	return s.DB.Create(&commodity).Error
}

func (s *CommodityHandler) Update(orgID, buID uuid.UUID, id string, commodity models.Commodity) error {
	commodity.BusinessUnitID = buID
	commodity.OrganizationID = orgID

	return s.DB.Model(&commodity).
		Where("id = ? AND organization_id = ? AND business_unit_id = ?", id, orgID, buID).
		Updates(&commodity).Error
}
