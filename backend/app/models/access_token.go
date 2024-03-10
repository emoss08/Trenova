package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Token struct {
	TimeStampedModel
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;index" json:"organizationId" validate:"required"`
	Organization   Organization `json:"-" validate:"omitempty"`
	BusinessUnitID uuid.UUID    `json:"businessUnitId" gorm:"type:uuid;not null" validate:"required"`
	BusinessUnit   BusinessUnit `json:"-" validate:"omitempty"`

	User     User       `json:"user" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	UserID   *uuid.UUID `json:"userID" gorm:"type:uuid;not null;" validate:"required"`
	LastUsed time.Time  `json:"lastUsed" gorm:"type:timestamp;not null;" validate:"required"`
	Expires  time.Time  `json:"expires" gorm:"type:timestamp;not null;" validate:"required"`
	Token    string     `json:"token" gorm:"type:varchar(255);not null;unique" validate:"required,max=255"`
	Key      string     `json:"key" gorm:"type:varchar(255);not null;unique" validate:"required,max=255"`
}

func (t *Token) BeforeCreate(tx *gorm.DB) error {
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
