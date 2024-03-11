package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserFavorite struct {
	TimeStampedModel
	Organization   Organization `json:"-" validate:"omitempty"`
	BusinessUnit   BusinessUnit `json:"-" validate:"omitempty"`
	User           User         `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-" `
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;index"                                        json:"organizationId" validate:"required"`
	BusinessUnitID uuid.UUID    `gorm:"type:uuid;not null"                                              json:"businessUnitId" validate:"required"`
	UserID         uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_user_page_link"               json:"userId"`
	PageLink       string       `gorm:"type:varchar(255);not null;uniqueIndex:idx_user_page_link"       json:"pageLink" validate:"required,max=255"`
}

func (uf *UserFavorite) FetchUserFavorites(db *gorm.DB, userID, orgID, buID uuid.UUID) ([]UserFavorite, error) {
	var userFavorites []UserFavorite
	if err := db.Model(&UserFavorite{}).Where("user_id = ? AND organization_id = ? AND business_unit_id = ?", userID, orgID, buID).Find(&userFavorites).Error; err != nil {
		return userFavorites, err
	}

	return userFavorites, nil
}
