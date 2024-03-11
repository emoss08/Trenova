package handlers

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"trenova/app/middleware"
	"trenova/app/models"
	"trenova/utils"
)

func GetDispatchControl(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		var dc models.DispatchControl
		if err := db.Model(&models.DispatchControl{}).Where("organization_id = ?", orgID).First(&dc).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, dc)
	}
}

func UpdateDispatchControl(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		var dc models.DispatchControl
		if err := db.Where("organization_id = ? AND business_unit_id = ?", orgID, buID).First(&dc).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		if err := utils.ParseBodyAndValidate(w, r, &dc); err != nil {
			return
		}

		if err := db.Save(&dc).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, dc)
	}
}
