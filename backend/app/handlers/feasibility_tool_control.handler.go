package handlers

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"trenova/app/middleware"
	"trenova/app/models"
	"trenova/utils"
)

func GetFeasibilityToolControl(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		var feasibilityToolControl models.FeasibilityToolControl
		if err := db.Model(&models.FeasibilityToolControl{}).Where("organization_id = ?", orgID).First(&feasibilityToolControl).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, feasibilityToolControl)
	}
}

func UpdateFeasibilityToolControl(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		var feasibilityToolControl models.FeasibilityToolControl
		if err := db.Where("organization_id = ? AND business_unit_id = ?", orgID, buID).First(&feasibilityToolControl).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		if err := utils.ParseBodyAndValidate(w, r, &feasibilityToolControl); err != nil {
			return
		}

		if err := db.Save(&feasibilityToolControl).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, feasibilityToolControl)
	}
}
