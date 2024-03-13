package models

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UnitOfMeasure string

// Commodity represents a commodity.
type Commodity struct {
	BaseModel
	HazardousMaterial   HazardousMaterial `gorm:"foreignKey:HazardousMaterialID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-" validate:"omitempty"`
	Organization        Organization      `json:"-" validate:"omitempty"`
	BusinessUnit        BusinessUnit      `json:"-" validate:"omitempty"`
	OrganizationID      uuid.UUID         `gorm:"type:uuid;not null;uniqueIndex:idx_commodity_name_organization_id" json:"organizationId" validate:"required"`
	BusinessUnitID      uuid.UUID         `gorm:"type:uuid;not null;index" json:"businessUnitId" validate:"required"`
	Status              StatusType        `gorm:"type:status_type;not null;default:'A'" json:"status" validate:"required,len=1,oneof=A I"`
	Name                string            `gorm:"type:varchar(100);not null;uniqueIndex:idx_commodity_name_organization_id,expression:lower(name);check:(length(name)>=3)" json:"name" validate:"required"`
	IsHazmat            string            `gorm:"type:varchar(1);not null;default:N" json:"isHazmat" validate:"omitempty,oneof=Y N"`
	UnitOfMeasure       UnitOfMeasure     `gorm:"type:varchar(1);" json:"unitOfMeasure" validate:"omitempty,oneof=P T D C A B O L I S"`
	HazardousMaterialID *uuid.UUID        `gorm:"type:uuid;uniqueIndex:idx_commodity_hazardous_material_id" json:"hazardousMaterialId" validate:"omitempty"`
	MinTemp             *float64          `gorm:"type:numeric(10,2);" json:"minTemp" validate:"omitempty,ltfield=MaxTemp"`
	MaxTemp             *float64          `gorm:"type:numeric(10,2);" json:"maxTemp" validate:"omitempty,gtfield=MinTemp"`
	SetPointTemp        *float64          `gorm:"type:numeric(10,2);" json:"setPointTemperature" validate:"omitempty"`
	Description         *string           `gorm:"type:text;" json:"description" validate:"omitempty"`
}

func (c *Commodity) SetOrgID(orgID uuid.UUID) {
	c.OrganizationID = orgID
}

func (c *Commodity) SetBuID(buID uuid.UUID) {
	c.BusinessUnitID = buID
}

var errIsHazmatWithoutHazardousMaterial = errors.New("commodity is hazardous material but has no hazardous material")

func (c *Commodity) validateCommodity() error {
	if c.IsHazmat == "Y" && c.HazardousMaterialID == nil {
		return errIsHazmatWithoutHazardousMaterial
	}

	return nil
}

func (c *Commodity) BeforeCreate(_ *gorm.DB) error {
	if c.HazardousMaterialID != nil {
		c.IsHazmat = "Y"
	}

	return c.validateCommodity()
}

func (c *Commodity) BeforeUpdate(_ *gorm.DB) error {
	if c.HazardousMaterialID != nil {
		c.IsHazmat = "Y"
	}

	return c.validateCommodity()
}

func (c *Commodity) FetchCommoditiesForOrg(db *gorm.DB, orgID, buID uuid.UUID, offset, limit int) ([]Commodity, int64, error) {
	var commodities []Commodity

	var totalRows int64

	if err := db.
		Model(&Commodity{}).Where("organization_id = ? AND business_unit_id = ?", orgID, buID).Count(&totalRows).Error; err != nil {
		return commodities, 0, err
	}

	if err := db.
		Model(&Commodity{}).
		Where("organization_id = ? AND business_unit_id = ?", orgID, buID).Offset(offset).Limit(limit).Order("created_at desc").
		Find(&commodities).Error; err != nil {
		return commodities, 0, err
	}

	return commodities, totalRows, nil
}

func (c *Commodity) FetchCommodityDetails(db *gorm.DB, orgID, buID uuid.UUID, id string) (Commodity, error) {
	var commodity Commodity

	if err := db.
		Model(&Commodity{}).Where("organization_id = ? AND id = ? AND business_unit_id = ?", orgID, id, buID).
		First(&commodity).Error; err != nil {
		return commodity, err
	}

	return commodity, nil
}
