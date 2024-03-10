package handlers

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"trenova/app/middleware"
	"trenova/app/models"
	"trenova/utils"
)

func GetBillingControl(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		var bc models.BillingControl
		if err := db.Model(&models.BillingControl{}).Where("organization_id = ?", orgID).First(&bc).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, bc)
	}
}

func UpdateBillingControl(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		var bc models.BillingControl
		if err := db.Where("organization_id = ? AND business_unit_id = ?", orgID, buID).First(&bc).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		if err := utils.ParseBodyAndValidate(w, r, &bc); err != nil {
			return
		}

		if err := db.Save(&bc).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, bc)
	}
}
