package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserFavorite struct {
	TimeStampedModel
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;index" json:"organizationId" validate:"required"`
	Organization   Organization `json:"-" validate:"omitempty"`
	BusinessUnitID uuid.UUID    `json:"businessUnitId" gorm:"type:uuid;not null" validate:"required"`
	BusinessUnit   BusinessUnit `json:"-" validate:"omitempty"`
	UserID         *uuid.UUID   `json:"userID" gorm:"type:uuid;not null;uniqueIndex:idx_user_page_link"`
	User           *User        `json:"user" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	PageLink       string       `json:"pageLink" gorm:"type:varchar(255);not null;uniqueIndex:idx_user_page_link" validate:"required,max=255"`
}

func (uf *UserFavorite) FetchUserFavorites(db *gorm.DB, userID, orgID, buID uuid.UUID) ([]UserFavorite, error) {
	var userFavorites []UserFavorite
	if err := db.Model(&UserFavorite{}).Where("user_id = ? AND organization_id = ? AND business_unit_id = ?", userID, orgID, buID).Find(&userFavorites).Error; err != nil {
		return userFavorites, err
	}

	return userFavorites, nil
}
