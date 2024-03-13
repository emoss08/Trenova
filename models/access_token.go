package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Token struct {
	BaseModel
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;index" json:"organizationId" validate:"required"`
	BusinessUnitID uuid.UUID    `gorm:"type:uuid;not null" json:"businessUnitId" validate:"required"`
	UserID         uuid.UUID    `gorm:"type:uuid;not null;" json:"userID" validate:"required"`
	Organization   Organization `json:"-" validate:"omitempty"`
	BusinessUnit   BusinessUnit `json:"-" validate:"omitempty"`
	User           User         `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user"`
	LastUsed       time.Time    `gorm:"type:timestamp;not null;" json:"lastUsed" validate:"required"`
	Expires        time.Time    `gorm:"type:timestamp;not null;" json:"expires" validate:"required"`
	Token          string       `gorm:"type:varchar(255);not null;unique" json:"token" validate:"required,max=255"`
	Key            string       `gorm:"type:varchar(255);not null;unique" json:"key" validate:"required,max=255"`
}

func (t *Token) BeforeCreate(_ *gorm.DB) error {
	if t.Key == "" {
		err := t.generateKey()
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Token) IsExpired() bool {
	return t.Expires.Before(time.Now())
}

func (t *Token) generateKey() error {
	// Generate a random key for the token
	key, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	t.Key = key.String()

	return nil
}
