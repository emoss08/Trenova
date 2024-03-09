package handlers

import (
	"net/http"
	"trenova-go-backend/app/models"
	"trenova-go-backend/utils"

	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

// Returns the organization of the user.
func GetOrganization(db *gorm.DB, store *gormstore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgId, ok := utils.GetUserOrgFromSession(r, store)

		if !ok {
			utils.ResponseWithError(w, http.StatusUnauthorized, utils.ValidationErrorResponse{
				Type: "unauthorized",
				Errors: []utils.ValidationErrorDetail{
					{
						Code:   "invalid",
						Detail: "You are not authorized to access this resource.",
						Attr:   "all",
					},
				},
			})
			return
		}

		var org models.Organization
		if err := db.Model(&models.Organization{}).Where("id = ?", orgId).First(&org).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)
			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, org)
	}
}
