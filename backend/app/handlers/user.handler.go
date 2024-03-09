package handlers

import (
	"net/http"
	"trenova-go-backend/app/models"
	"trenova-go-backend/utils"

	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

func GetAuthenticatedUser(db *gorm.DB, store *gormstore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		au, ok := utils.GetUserIDFromSession(r, store)

		if !ok {
			utils.ResponseWithError(w, http.StatusUnauthorized, utils.ValidationErrorDetail{
				Code:   "unauthorized",
				Detail: "Unauthorized",
				Attr:   "userId",
			})
			return
		}

		var u models.User
		user, err := u.GetUserByID(db, au)
		if err != nil {
			utils.ResponseWithError(w, http.StatusInternalServerError, utils.ValidationErrorDetail{
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
