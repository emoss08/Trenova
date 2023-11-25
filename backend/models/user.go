package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

type User struct {
	BaseModel
	IsActive        bool   `gorm:"default:true;" json:"isActive"`
	Username        string `gorm:"size:30;unique;not null" json:"username"`
	Password        string `gorm:"not null" json:"-"`
	Email           string `gorm:"size:255;unique;not null" json:"email"`
	IsAdmin         bool   `gorm:"default:false;" json:"isAdmin"`
	DateJoined      string `gorm:"type:date;" json:"dateJoined"`
	SessionKey      string `gorm:"size:255;" json:"-"`
	Department      Department
	DepartmentID    *uuid.UUID `gorm:"type:uuid;" json:"departmentId"`
	UserProfile     UserProfile
	Roles           []Role  `gorm:"many2many:user_roles;" json:"roles"`
	LastResetSentAt *string `gorm:"type:timestamp;" json:"-" validate:"omitempty"`
}

// HashPassword creates a bcyrpy hash of the password
func (u *User) SetPassword(password string) error {
	// hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	u.Password = string(hashedPassword)

	// Set the last reset sent at time to now
	var now = time.Now().Format(time.RFC3339)
	u.LastResetSentAt = &now
	u.DateJoined = now

	return nil
}

// ValidatePassword validates a plain password against the model's password.
func (u *User) ValidatePassword(password string) bool {
	bytePassword := []byte(password)
	bytePasswordHash := []byte(u.Password)

	err := bcrypt.CompareHashAndPassword(bytePasswordHash, bytePassword)

	return err == nil
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	// Convert username to lowercase
	caser := cases.Lower(language.AmericanEnglish)
	u.Username = caser.String(u.Username)

	// Hash the password
	err = u.SetPassword(u.Password)
	if err != nil {
		return err
	}

	// Set the date joined
	// u.DateJoined = types.Timestamp(time.Now()).String()

	return nil
}

func (u *User) HasRole(roleName string) bool {
	for _, role := range u.Roles {
		if role.Name == roleName {
			return true
		}
	}
	return false
}

func (u *User) HasPermission(permissionName string) bool {
	for _, role := range u.Roles {
		for _, perm := range role.Permissions {
			if perm.Name == permissionName {
				return true
			}
		}
	}
	return false
}

type UserProfile struct {
	BaseModel
	UserID          uuid.UUID  `gorm:"type:uuid;" json:"userId" validate:"required,uuid"`
	JobTitleID      *uuid.UUID `gorm:"type:uuid;" json:"jobTitleId" validate:"required,uuid"`
	JobTitle        JobTitle
	FirstName       string  `gorm:"size:30;" json:"firstName" validate:"required,max=255"`
	LastName        string  `gorm:"size:255;" json:"lastName" validate:"required,max=255"`
	ProfilePic      string  `gorm:"size:255;" json:"profilePic" validate:"omitempty,max=255"`
	Thumbnail       string  `gorm:"size:255;" json:"thumbnail" validate:"omitempty,max=255"`
	AddressLine1    string  `gorm:"size:255;" json:"addressLine1" validate:"required,max=255"`
	AddressLine2    *string `gorm:"size:255;" json:"addressLine2" validate:"omitempty,max=255"`
	City            string  `gorm:"size:255;" json:"city" validate:"omitempty,max=255"`
	State           string  `gorm:"size:2;" json:"state" validate:"omitempty,max=2"`
	ZipCode         string  `gorm:"size:5;" json:"zipCode" validate:"omitempty,max=5;isnumeric"`
	PhoneNumber     *string `gorm:"size:20;" json:"phoneNumber" validate:"omitempty,max=20;isnumeric"`
	IsPhoneVerified bool    `gorm:"default:false;" json:"isPhoneVerified" validate:"required,boolean"`
}

func (up *UserProfile) BeforeCreate(tx *gorm.DB) (err error) {
	caser := cases.Title(language.AmericanEnglish)

	up.FirstName = caser.String(up.FirstName)
	up.LastName = caser.String(up.LastName)

	return nil
}
