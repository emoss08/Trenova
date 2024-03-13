package handlers

import (
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/password"
	"log"
	"net/http"

	"github.com/emoss08/trenova/models"
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

func Login(db *gorm.DB, store *gormstore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var loginDetails struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := tools.ParseBody(r, &loginDetails); err != nil {
			tools.ResponseWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		var user models.User
		if err := db.Where("username = ?", loginDetails.Username).First(&user).Error; err != nil {
			tools.ResponseWithError(w, http.StatusUnauthorized, models.ValidationErrorResponse{
				Type: "validationError",
				Errors: []models.ValidationErrorDetail{
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
			tools.ResponseWithError(w, http.StatusUnauthorized, models.ValidationErrorResponse{
				Type: "validationError",
				Errors: []models.ValidationErrorDetail{
					{
						Code:   "invalid",
						Detail: "You have entered an incorrect password. Please try again..",
						Attr:   "password",
					},
				},
			})

			return
		}

		// Create a new session
		sessionID := tools.GetSystemSessionID()
		session, err := store.New(r, sessionID)
		if err != nil {
			tools.ResponseWithError(w, http.StatusInternalServerError, "Error creating session")
			return
		}

		// Set some session values
		session.Values["userID"] = user.ID
		session.Values["organizationID"] = user.OrganizationID
		session.Values["businessUnitID"] = user.BusinessUnitID

		// Save it before we write to the response/return from the handler
		if saveErr := store.Save(r, w, session); saveErr != nil {
			log.Printf("Error saving session: %v", saveErr)
			tools.ResponseWithError(w, http.StatusInternalServerError, "Error saving session")

			return
		}

		// Respond with success
		tools.ResponseWithJSON(w, http.StatusOK, "Logged in successfully")
	}
}

func Logout(store *gormstore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := tools.GetSystemSessionID()
		session, err := store.Get(r, sessionID)
		if err != nil {
			tools.ResponseWithError(w, http.StatusUnauthorized, "Not logged in")
			return
		}

		// Invalidate the session by setting MaxAge to -1
		session.Options.MaxAge = -1

		// Save the session to update the session in the database and delete the client's cookie
		if saveErr := store.Save(r, w, session); saveErr != nil {
			log.Printf("Error saving session: %v", saveErr)
			tools.ResponseWithError(w, http.StatusInternalServerError, "Error updating session")

			return
		}

		tools.ResponseWithJSON(w, http.StatusOK, "Logged out successfully")
	}
}
