package service

import (
	"backend/models"
	"backend/validation"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

type CreateUserRequest struct {
	Username       string      `json:"username" validate:"required,max=30"`
	Password       string      `json:"password" validate:"required"`
	Email          string      `json:"email" validate:"required,email"`
	IsAdmin        bool        `json:"isAdmin" validate:"omitempty,boolean"`
	DepartmentID   *uuid.UUID  `json:"departmentId" validate:"omitempty,uuid"`
	Roles          []uuid.UUID `json:"roles" validate:"omitempty,uuid"`
	BusinessUnitID uuid.UUID   `json:"businessUnitId" validate:"required,uuid"`
	OrganizationID uuid.UUID   `json:"organizationId" validate:"required,uuid"`
	UserProfile    struct {
		BusinessUnitID uuid.UUID  `json:"businessUnitId" validate:"required,uuid"`
		OrganizationID uuid.UUID  `json:"organizationId" validate:"required,uuid"`
		JobTitleID     *uuid.UUID `json:"jobTitleId" validate:"omitempty,uuid"`
		FirstName      string     `json:"firstName" validate:"required,max=30"`
		LastName       string     `json:"lastName" validate:"required,max=30"`
		ProfilePic     string     `json:"profilePic" validate:"omitempty,url"`
		AddressLine1   string     `json:"addressLine1" validate:"required,max=255"`
		AddressLine2   *string    `json:"addressLine2" validate:"omitempty,max=255"`
		City           string     `json:"city" validate:"required,max=255"`
		State          string     `json:"state" validate:"required,max=2"`
		ZipCode        string     `json:"zipCode" validate:"required,max=5,usazipcode"` // Example: 12345
		PhoneNumber    *string    `json:"phoneNumber" validate:"omitempty,e164,max=10"` // Example: +15555555555
	} `json:"userProfile" validate:"required"`
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) CreateUser(createReq *CreateUserRequest) error {
	fieldErrors, err := validation.ValidateStruct(createReq)
	if err != nil {
		// Return a custom error type with more details
		return models.NewAPIError(http.StatusInternalServerError, "internal_error", "An internal error occurred", nil)
	}
	if fieldErrors != nil {
		// Return a custom error for validation issues
		return models.NewAPIError(http.StatusBadRequest, "validation_error", "Validation failed", fieldErrors)
	}

	// Check for existing username
	var existingUser models.User
	result := s.db.Where("username = ?", createReq.Username).First(&existingUser)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// If the error is not "record not found", return a database error
			return models.NewAPIError(http.StatusInternalServerError, "database_error", "Database error", nil)
		}
	} else {
		// User with the same username found
		return models.NewAPIError(http.StatusBadRequest, "username_exists", "Username already exists", nil)
	}

	// Map CreateUserRequest to models.User
	user := models.User{
		Username:     createReq.Username,
		Password:     createReq.Password,
		Email:        createReq.Email,
		IsAdmin:      createReq.IsAdmin,
		DepartmentID: createReq.DepartmentID,
		Roles:        []models.Role{},
		BaseModel: models.BaseModel{
			BusinessUnitID: createReq.BusinessUnitID,
			OrganizationID: createReq.OrganizationID,
		},
	}

	// Hash the password if provided.
	if user.Password != "" {
		if err := user.SetPassword(user.Password); err != nil {
			return models.NewAPIError(http.StatusInternalServerError, "hashing_error", "Failed to hash password", nil)
		}
	}

	// Create the user in the database
	if err := s.db.Create(&user).Error; err != nil {
		dbErr := models.FieldError{
			Code:   "database_error",
			Detail: err.Error(),
			Attr:   "__all__",
		}
		return models.NewAPIError(http.StatusInternalServerError, "database_error", "Failed to create user: ", []models.FieldError{dbErr})
	}

	userProfile := models.UserProfile{
		UserID:       user.ID,
		JobTitleID:   createReq.UserProfile.JobTitleID,
		FirstName:    createReq.UserProfile.FirstName,
		LastName:     createReq.UserProfile.LastName,
		ProfilePic:   createReq.UserProfile.ProfilePic,
		AddressLine1: createReq.UserProfile.AddressLine1,
		AddressLine2: createReq.UserProfile.AddressLine2,
		City:         createReq.UserProfile.City,
		State:        createReq.UserProfile.State,
		ZipCode:      createReq.UserProfile.ZipCode,
		PhoneNumber:  createReq.UserProfile.PhoneNumber,
		BaseModel: models.BaseModel{
			BusinessUnitID: createReq.UserProfile.BusinessUnitID,
			OrganizationID: createReq.UserProfile.OrganizationID,
		},
	}

	if err := s.db.Create(&userProfile).Error; err != nil {
		dbErr := models.FieldError{
			Code:   "database_error",
			Detail: err.Error(),
			Attr:   "__all__",
		}
		return models.NewAPIError(http.StatusInternalServerError, "profile_creation_error", "Failed to create user profile", []models.FieldError{dbErr})
	}

	return nil
}

func (s *UserService) GetAllUsers() ([]models.User, error) {
	var users []models.User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
