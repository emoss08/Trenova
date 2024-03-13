package handlers

import (
	"github.com/emoss08/trenova/tools"
	"net/http"

	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/models"
	"github.com/google/uuid"
	"github.com/gorilla/csrf"
	"gorm.io/gorm"
)

func UpdateUser(db *gorm.DB, validator *tools.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		buID, ok := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Business Unit ID not found in the request context",
				Attr:   "businessUnitId",
			})
		}

		userID := tools.GetMuxVar(w, r, "userID")
		if userID == "" {
			// Error is already handled in GetMuxVar
			return
		}

		var u models.User

		// Let's make sure we're updating the right user, for the right organization and business unit
		if err := db.
			Where("id = ? AND organization_id = ? AND business_unit_id = ?", userID, orgID, buID).
			First(&u).Error; err != nil {
			tools.ResponseWithError(w, http.StatusNotFound, models.ValidationErrorDetail{
				Code:   "notFound",
				Detail: "User not found",
				Attr:   "id",
			})

			return
		}

		if err := tools.ParseBodyAndValidate(validator, w, r, &u); err != nil {
			return
		}

		if err := db.Save(&u).Error; err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		tools.ResponseWithJSON(w, http.StatusOK, u)
	}
}

func GetAuthenticatedUser(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(middleware.ContextKeyUserID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "userId",
			})
		}

		var u models.User
		user, err := u.GetUserByID(db, userID)
		if err != nil {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "databaseError",
				Detail: err.Error(),
				Attr:   "all",
			})

			return
		}

		tools.ResponseWithJSON(w, http.StatusOK, UserResponse{
			BusinessUnitID: user.BusinessUnitID,
			OrganizationID: user.OrganizationID,
			ID:             user.ID,
			Status:         user.Status,
			Name:           user.Name,
			Username:       user.Username,
			Email:          user.Email,
			DateJoined:     user.DateJoined,
			Timezone:       user.Timezone,
			ProfilePicURL:  user.ProfilePicURL,
			ThumbnailURL:   user.ThumbnailURL,
			PhoneNumber:    user.PhoneNumber,
			IsAdmin:        user.IsAdmin,
			IsSuperAdmin:   user.IsSuperAdmin,
		})
	}
}

// GetUserFavorites returns a list of user favorites.
func GetUserFavorites(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		buID, ok := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Business Unit ID not found in the request context",
				Attr:   "businessUnitId",
			})
		}

		userID, ok := r.Context().Value(middleware.ContextKeyUserID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "User ID not found in the request context",
				Attr:   "userId",
			})
		}

		var uf models.UserFavorite

		userFavorites, err := uf.FetchUserFavorites(db, userID, orgID, buID)
		if err != nil {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "databaseError",
				Detail: err.Error(),
				Attr:   "all",
			})

			return
		}

		w.Header().Set("X-CSRF-Token", csrf.Token(r))
		tools.ResponseWithJSON(w, http.StatusOK, userFavorites)
	}
}

func AddUserFavorite(db *gorm.DB, validator *tools.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var uf models.UserFavorite

		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		buID, ok := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Business Unit ID not found in the request context",
				Attr:   "businessUnitId",
			})
		}

		userID, ok := r.Context().Value(middleware.ContextKeyUserID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "User ID not found in the request context",
				Attr:   "userId",
			})
		}

		uf.OrganizationID = orgID
		uf.BusinessUnitID = buID
		uf.UserID = userID

		if err := tools.ParseBodyAndValidate(validator, w, r, &uf); err != nil {
			return
		}

		if err := db.Create(&uf).Error; err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		tools.ResponseWithJSON(w, http.StatusCreated, uf)
	}
}

func RemoveUserFavorite(db *gorm.DB, validator *tools.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		buID, ok := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Business Unit ID not found in the request context",
				Attr:   "businessUnitId",
			})
		}

		userID, ok := r.Context().Value(middleware.ContextKeyUserID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "User ID not found in the request context",
				Attr:   "userId",
			})
		}

		var uf models.UserFavorite
		uf.OrganizationID = orgID
		uf.BusinessUnitID = buID
		uf.UserID = userID

		// Get the pageLink from the body
		if err := tools.ParseBodyAndValidate(validator, w, r, &uf); err != nil {
			return
		}

		// Delete the user favorite by the pageLink
		if err := db.
			Where("organization_id = ? AND business_unit_id = ? AND user_id = ? AND page_link = ?",
				orgID, buID, userID, uf.PageLink).
			Delete(&uf).Error; err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		tools.ResponseWithJSON(w, http.StatusNoContent, nil)
	}
}
