package handlers

import (
	"net/http"
	"trenova/app/middleware"
	"trenova/app/models"
	"trenova/utils"

	"github.com/google/uuid"
	"github.com/gorilla/csrf"
	"gorm.io/gorm"
)

func GetAuthenticatedUser(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(middleware.ContextKeyUserID).(uuid.UUID)

		var u models.User
		user, err := u.GetUserByID(db, userID)
		if err != nil {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "databaseError",
				Detail: err.Error(),
				Attr:   "all",
			})
			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, UserResponse{
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

func GetUserFavorites(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)
		userID := r.Context().Value(middleware.ContextKeyUserID).(uuid.UUID)

		var uf models.UserFavorite

		userFavorites, err := uf.FetchUserFavorites(db, userID, orgID, buID)
		if err != nil {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "databaseError",
				Detail: err.Error(),
				Attr:   "all",
			})
			return
		}

		w.Header().Set("X-CSRF-Token", csrf.Token(r))
		utils.ResponseWithJSON(w, http.StatusOK, userFavorites)
	}
}

func AddUserFavorite(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var uf models.UserFavorite
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)
		userID := r.Context().Value(middleware.ContextKeyUserID).(uuid.UUID)

		uf.OrganizationID = orgID
		uf.BusinessUnitID = buID
		uf.UserID = &userID

		if err := utils.ParseBodyAndValidate(w, r, &uf); err != nil {
			return
		}

		if err := db.Create(&uf).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)
			return
		}

		utils.ResponseWithJSON(w, http.StatusCreated, uf)
	}
}

func RemoveUserFavorite(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)
		userID := r.Context().Value(middleware.ContextKeyUserID).(uuid.UUID)

		var uf models.UserFavorite
		uf.OrganizationID = orgID
		uf.BusinessUnitID = buID
		uf.UserID = &userID

		// Get the pageLink from the body
		if err := utils.ParseBodyAndValidate(w, r, &uf); err != nil {
			return
		}

		// Delete the user favorite by the pageLink
		if err := db.Where("organization_id = ? AND business_unit_id = ? AND user_id = ? AND page_link = ?", orgID, buID, userID, uf.PageLink).Delete(&uf).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)
			return
		}

		utils.ResponseWithJSON(w, http.StatusNoContent, nil)
	}
}
