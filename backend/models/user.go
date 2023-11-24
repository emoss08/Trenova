package models

import (
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

type User struct {
	BaseModel
	IsActive     bool   `gorm:"default:true;" json:"isActive" validate:"required,boolean"`
	Username     string `gorm:"size:30;unique;not null" json:"username" validate:"required,max=30"`
	Password     string `gorm:"not null" json:"password" validate:"required"`
	Email        string `gorm:"size:255;unique;not null" json:"email" validate:"required,email"`
	IsStaff      bool   `gorm:"default:false;" json:"isStaff" validate:"required,boolean"`
	DateJoined   string `gorm:"type:date;" json:"dateJoined" validate:"required"`
	SessionKey   string `gorm:"size:255;" json:"sessionKey" validate:"omitempty"`
	Department   Department
	DepartmentID uuid.UUID `gorm:"type:uuid;" json:"departmentId" validate:"required,uuid"`
	UserProfile  UserProfile
}

// HashPassword creates a bcyrpy hash of the password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil
	}
	return string(bytes), nil
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	// Convert username to lowercase
	caser := cases.Lower(language.AmericanEnglish)
	u.Username = caser.String(u.Username)

	// Hashing the password
	hp, err := HashPassword(u.Password)
	if err != nil {
		return err
	}

	u.Password = hp

	return nil
}

type UserProfile struct {
	BaseModel
	UserID          uuid.UUID `gorm:"type:uuid;" json:"userId" validate:"required,uuid"`
	JobTitleID      uuid.UUID `gorm:"type:uuid;" json:"jobTitleId" validate:"required,uuid"`
	JobTitle        JobTitle
	FirstName       string  `gorm:"size:255;" json:"firstName" validate:"required,max=255"`
	LastName        string  `gorm:"size:255;" json:"lastName" validate:"required,max=255"`
	ProfilePic      string  `gorm:"size:255;" json:"profilePic" validate:"omitempty,max=255"`
	Thumbnail       string  `gorm:"size:255;" json:"thumbnail" validate:"omitempty,max=255"`
	AddressLine1    string  `gorm:"size:255;" json:"addressLine1" validate:"required,max=255"`
	AddressLine2    *string `gorm:"size:255;" json:"addressLine2" validate:"omitempty,max=255"`
	City            *string `gorm:"size:255;" json:"city" validate:"omitempty,max=255"`
	State           *string `gorm:"size:2;" json:"state" validate:"omitempty,max=2"`
	ZipCode         *string `gorm:"size:5;" json:"zipCode" validate:"omitempty,max=5;isnumeric"`
	PhoneNumber     *string `gorm:"size:20;" json:"phoneNumber" validate:"omitempty,max=20;isnumeric"`
	IsPhoneVerified bool    `gorm:"default:false;" json:"isPhoneVerified" validate:"required,boolean"`
}

func (up *UserProfile) BeforeCreate(tx *gorm.DB) (err error) {
	caser := cases.Title(language.AmericanEnglish)

	up.FirstName = caser.String(up.FirstName)
	up.LastName = caser.String(up.LastName)

	return nil
}
