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
	Status        StatusType   `json:"status" gorm:"type:status_type;not null;default:'A'" validate:"omitempty,len=1,oneof=A I"`
	Name          string       `json:"name" gorm:"type:varchar(255);not null;" validate:"required,max=255"`
	Username      string       `json:"username" gorm:"type:varchar(30);not null;" validate:"required,max=30"`
	Password      string       `json:"password" gorm:"type:varchar(100);not null;" validate:"required,max=100"`
	Email         string       `json:"email" gorm:"type:varchar(255);not null;" validate:"required,max=255"`
	DateJoined    string       `json:"dateJoined" gorm:"type:date;not null;" validate:"omitempty"`
	Timezone      TimezoneType `json:"timezone" gorm:"type:timezone_type;not null;default:'America/Los_Angeles'" validate:"omitempty,oneof=America/Los_Angeles America/Denver"`
	ProfilePicURL *string      `json:"profilePicUrl" gorm:"type:varchar(255);" validate:"omitempty,url"`
	ThumbnailURL  *string      `json:"thumbnailUrl" gorm:"type:varchar(255);" validate:"omitempty,url"`
	PhoneNumber   *string      `json:"phoneNumber" gorm:"type:varchar(20);" validate:"omitempty,max=20,phoneNum"`
	IsAdmin       bool         `json:"isAdmin" gorm:"type:boolean;not null;default:false" validate:"omitempty"`
	IsSuperAdmin  bool         `json:"isSuperAdmin" gorm:"type:boolean;not null;default:false" validate:"omitempty"`
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

type UserFavorite struct {
	BaseModel
	UserID   *uuid.UUID `json:"userID" gorm:"type:uuid;not null;"`
	User     *User      `json:"user" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	PageLink string     `json:"pageLink" gorm:"type:varchar(255);not null;" validate:"required,max=255"`
}

type JobTitle struct {
	BaseModel
	Status      StatusType      `json:"status" gorm:"type:status_type;not null;default:'A'" validate:"required,len=1,oneof=A I"`
	Name        string          `json:"name" gorm:"type:varchar(100);not null;" validate:"required,max=100"`
	Description *string         `json:"description" gorm:"type:varchar(100);" validate:"required,max=100"`
	JobFunction JobFunctionType `json:"jobFunction" gorm:"type:job_function_type;not null;" validate:"required,len=1,oneof=MGR MT SP D B F S SA A"`
}

type Token struct {
	BaseModel
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
