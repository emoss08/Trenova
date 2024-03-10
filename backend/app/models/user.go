package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

type User struct {
	TimeStampedModel
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;index" json:"organizationId" validate:"required"`
	Organization   Organization `json:"-" validate:"omitempty"`
	BusinessUnitID uuid.UUID    `json:"businessUnitId" gorm:"type:uuid;not null" validate:"required"`
	BusinessUnit   BusinessUnit `json:"-" validate:"omitempty"`
	Status         StatusType   `json:"status" gorm:"type:status_type;not null;default:'A'" validate:"omitempty,len=1,oneof=A I"`
	Name           string       `json:"name" gorm:"type:varchar(255);not null;" validate:"required,max=255"`
	Username       string       `json:"username" gorm:"type:varchar(30);not null;" validate:"required,max=30"`
	Password       string       `json:"password" gorm:"type:varchar(100);not null;" validate:"required,max=100"`
	Email          string       `json:"email" gorm:"type:varchar(255);not null;" validate:"required,max=255"`
	DateJoined     string       `json:"dateJoined" gorm:"type:date;not null;" validate:"omitempty"`
	Timezone       TimezoneType `json:"timezone" gorm:"type:timezone_type;not null;default:'America/Los_Angeles'" validate:"omitempty,oneof=America/Los_Angeles America/Denver"`
	ProfilePicURL  *string      `json:"profilePicUrl" gorm:"type:varchar(255);" validate:"omitempty,url"`
	ThumbnailURL   *string      `json:"thumbnailUrl" gorm:"type:varchar(255);" validate:"omitempty,url"`
	PhoneNumber    *string      `json:"phoneNumber" gorm:"type:varchar(20);" validate:"omitempty,max=20,phoneNum"`
	IsAdmin        bool         `json:"isAdmin" gorm:"type:boolean;not null;default:false" validate:"omitempty"`
	IsSuperAdmin   bool         `json:"isSuperAdmin" gorm:"type:boolean;not null;default:false" validate:"omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.DateJoined == "" {
		u.DateJoined = time.Now().Format("2006-01-02")
	}

	u.uppercaseName()

	return nil
}

func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.uppercaseName()

	return nil
}

func (u *User) uppercaseName() error {
	// Capitalize the first letter of the user's name
	caser := cases.Title(language.AmericanEnglish)
	u.Name = caser.String(u.Name)

	return nil
}

func (u *User) GetUserByID(db *gorm.DB, userID uuid.UUID) (User, error) {
	var user User
	if err := db.Model(&User{}).Where("id = ?", userID).First(&user).Error; err != nil {
		return user, err
	}

	return user, nil
}
