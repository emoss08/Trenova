package models

import (
	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

type User struct {
	BaseModel
	IsActive     bool   `gorm:"default:true;"`
	Username     string `gorm:"size:255;unique;not null"`
	Password     string `gorm:"not null"`
	Email        string `gorm:"size:255;unique;not null"`
	IsStaff      bool   `gorm:"default:false;"`
	DateJoined   string `gorm:"type:date;"`
	SessionKey   string `gorm:"size:255;"`
	Department   Department
	DepartmentID uuid.UUID `gorm:"type:uuid;"`
	UserProfile  UserProfile
}

type UserProfile struct {
	BaseModel
	UserID          uuid.UUID `gorm:"type:uuid;"`
	JobTitleID      uuid.UUID `gorm:"type:uuid;"`
	JobTitle        JobTitle
	FirstName       string  `gorm:"size:255;"`
	LastName        string  `gorm:"size:255;"`
	ProfilePic      string  `gorm:"size:255;"`
	Thumbnail       string  `gorm:"size:255;"`
	AddressLine1    string  `gorm:"size:255;"`
	AddressLine2    *string `gorm:"size:255;"`
	City            *string `gorm:"size:255;"`
	State           *string `gorm:"size:2;"`
	ZipCode         *string `gorm:"size:5;"`
	PhoneNumber     *string `gorm:"size:20;"`
	IsPhoneVerified bool    `gorm:"default:false;"`
}

func (up *UserProfile) BeforeCreate(tx *gorm.DB) (err error) {
	caser := cases.Title(language.AmericanEnglish)

	up.FirstName = caser.String(up.FirstName)
	up.LastName = caser.String(up.LastName)

	return nil
}
