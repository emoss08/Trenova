package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

type User struct {
	BaseModel
	BusinessUnit   BusinessUnit `json:"-" validate:"omitempty"`
	Organization   Organization `json:"-" validate:"omitempty"`
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;index"                                  json:"organizationId" validate:"required"`
	BusinessUnitID uuid.UUID    `gorm:"type:uuid;not null"                                        json:"businessUnitId" validate:"required"`
	Status         StatusType   `gorm:"type:status_type;not null;default:'A'"                     json:"status"        validate:"omitempty,len=1,oneof=A I"`
	Name           string       `gorm:"type:varchar(255);not null;"                               json:"name"          validate:"required,max=255"`
	Username       string       `gorm:"type:varchar(30);not null;"                                json:"username"      validate:"required,max=30"`
	Password       string       `gorm:"type:varchar(100);not null;"                               json:"password"      validate:"required,max=100"`
	Email          string       `gorm:"type:varchar(255);not null;"                               json:"email"         validate:"required,max=255"`
	DateJoined     string       `gorm:"type:date;not null;"                                       json:"dateJoined"    validate:"omitempty"`
	Timezone       TimezoneType `gorm:"type:timezone_type;not null;default:'America/Los_Angeles'" json:"timezone"      validate:"omitempty,oneof=America/Los_Angeles America/Denver"`
	ProfilePicURL  *string      `gorm:"type:varchar(255);"                                        json:"profilePicUrl" validate:"omitempty,url"`
	ThumbnailURL   *string      `gorm:"type:varchar(255);"                                        json:"thumbnailUrl"  validate:"omitempty,url"`
	PhoneNumber    *string      `gorm:"type:varchar(20);"                                         json:"phoneNumber"   validate:"omitempty,max=20,phoneNum"`
	IsAdmin        bool         `gorm:"type:boolean;not null;default:false"                       json:"isAdmin"       validate:"omitempty"`
	IsSuperAdmin   bool         `gorm:"type:boolean;not null;default:false"                       json:"isSuperAdmin"  validate:"omitempty"`
}

func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.DateJoined == "" {
		u.DateJoined = time.Now().Format("2006-01-02")
	}

	err := u.uppercaseName()
	if err != nil {
		return err
	}

	return nil
}

func (u *User) BeforeUpdate(_ *gorm.DB) error {
	err := u.uppercaseName()
	if err != nil {
		return err
	}

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
