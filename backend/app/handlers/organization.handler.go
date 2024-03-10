package handlers

import (
	"net/http"
	"trenova/app/middleware"
	"trenova/app/models"
	"trenova/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetOrganization Returns the organization of the user.
func GetOrganization(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the orgID and buID from the request's context
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		var org models.Organization
		if err := db.Model(&models.Organization{}).Where("id = ?", orgID).First(&org).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, org)
	}
}

func UpdateOrganization(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		var org models.Organization
		if err := db.Where("id = ? AND business_unit_id = ?", orgID, buID).First(&org).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		// Update the organization with the new details from the form
		if err := utils.ParseBodyAndValidate(w, r, &org); err != nil {
			return
		}

		if err := db.Save(&org).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, org)
	}
}
