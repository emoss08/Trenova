package handlers

import (
	"log"
	"net/http"
	"trenova-go-backend/app/models"
	"trenova-go-backend/utils"
	"trenova-go-backend/utils/password"

	"github.com/google/uuid"
	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

type UserResponse struct {
	BusinessUnitID uuid.UUID           `json:"businessUnitId"`
	OrganizationID uuid.UUID           `json:"organizationId"`
	ID             uuid.UUID           `json:"id"`
	Status         models.StatusType   `json:"status"`
	Name           string              `json:"name"`
	Username       string              `json:"username"`
	Password       string              `json:"-"`
	Email          string              `json:"email"`
	DateJoined     string              `json:"dateJoined"`
	Timezone       models.TimezoneType `json:"timezone"`
	ProfilePicURL  *string             `json:"profilePicUrl"`
	ThumbnailURL   *string             `json:"thumbnailUrl"`
	PhoneNumber    *string             `json:"phoneNumber"`
	IsAdmin        bool                `json:"isAdmin"`
	IsSuperAdmin   bool                `json:"isSuperAdmin"`
}

// func SignUp(db *gorm.DB, store *gormstore.Store) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		u := new(models.User)

// 		if err := utils.ParseBodyAndValidate(w, r, u); err != nil {
// 			return
// 		}

// 		user := &models.User{
// 			BaseModel: models.BaseModel{
// 				OrganizationID: u.OrganizationID,
// 				BusinessUnitID: u.BusinessUnitID,
// 			},
// 			Name:     u.Name,
// 			Password: password.Generate(u.Password),
// 			Email:    u.Email,
// 			Username: u.Username,
// 		}

// 		if err := db.Create(&user).Error; err != nil {
// 			errorResponse := utils.FormatDatabaseError(err)
// 			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)
// 			return
// 		}

// 		// // Create a new session
// 		// ns, err := store.New(r, "session")
// 		// if err != nil {
// 		// 	utils.ResponseWithError(w, http.StatusInternalServerError, "Error creating session")
// 		// 	return
// 		// }

// 		// ns.Values["userID"] = user.ID
// 		// ns.Values["organizationID"] = user.OrganizationID

// 		// // Save it before we write to the response/return from the handler
// 		// if err := store.Save(r, w, ns); err != nil {
// 		// 	log.Println(err)
// 		// }

// 		// // Print out the username from the user data

// 		utils.ResponseWithJSON(w, http.StatusCreated, UserResponse{
// 			BusinessUnitID: user.BaseModel.BusinessUnitID,
// 			OrganizationID: user.BaseModel.OrganizationID,
// 			ID:             user.ID,
// 			Status:         user.Status,
// 			Name:           user.Name,
// 			Username:       user.Username,
// 			Email:          user.Email,
// 			DateJoined:     user.DateJoined,
// 			Timezone:       user.Timezone,
// 			ProfilePicURL:  user.ProfilePicURL,
// 			ThumbnailURL:   user.ThumbnailURL,
// 			PhoneNumber:    user.PhoneNumber,
// 		})
// 	}
// }

func Login(db *gorm.DB, store *gormstore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var loginDetails struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := utils.ParseBody(r, &loginDetails); err != nil {
			utils.ResponseWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		var user models.User
		if err := db.Where("username = ?", loginDetails.Username).First(&user).Error; err != nil {
			utils.ResponseWithError(w, http.StatusUnauthorized, utils.ValidationErrorResponse{
				Type: "validationError",
				Errors: []utils.ValidationErrorDetail{
					{
						Code:   "invalid",
						Detail: "Invalid username or password",
						Attr:   "username",
					},
				},
			})
			return
		}

		// Check if the password is correct
		if err := password.Verify(user.Password, loginDetails.Password); err != nil {
			utils.ResponseWithError(w, http.StatusUnauthorized, utils.ValidationErrorResponse{
				Type: "validationError",
				Errors: []utils.ValidationErrorDetail{
					{
						Code:   "invalid",
						Detail: "You have enetered an incorrect password. Please try again..",
						Attr:   "password",
					},
				},
			})
			return
		}

		// Create a new session
		sessionID := utils.GetSystemSessionID()
		session, err := store.New(r, sessionID)
		if err != nil {
			utils.ResponseWithError(w, http.StatusInternalServerError, "Error creating session")
			return
		}

		// Set some session values
		session.Values["userID"] = user.ID
		session.Values["organizationID"] = user.OrganizationID

		// Save it before we write to the response/return from the handler
		if err := store.Save(r, w, session); err != nil {
			log.Println(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, "Error saving session")
			return
		}

		// Respond with success
		utils.ResponseWithJSON(w, http.StatusOK, "Logged in successfully")
	}
}

func Logout(store *gormstore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := utils.GetSystemSessionID()
		session, err := store.Get(r, sessionID)
		if err != nil {
			utils.ResponseWithError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		// Invalidate the session by setting MaxAge to -1
		session.Options.MaxAge = -1

		// Save the session to update the session in the database and delete the client's cookie
		if err := store.Save(r, w, session); err != nil {
			utils.ResponseWithError(w, http.StatusInternalServerError, "Error updating session")
			return
		}

		// No need to call store.Delete as Save takes care of removing the session when MaxAge is -1

		utils.ResponseWithJSON(w, http.StatusOK, "Logged out successfully")
	}
}
