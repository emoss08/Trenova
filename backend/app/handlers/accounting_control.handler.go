package handlers

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"trenova/app/middleware"
	"trenova/app/models"
	"trenova/utils"
)

func GetAccountingControl(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		var ac models.AccountingControl
		if err := db.Model(&models.AccountingControl{}).Where("organization_id = ?", orgID).First(&ac).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, ac)
	}
}

func UpdateAccountingControl(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		var ac models.AccountingControl
		if err := db.Where("organization_id = ? AND business_unit_id = ?", orgID, buID).First(&ac).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		if err := utils.ParseBodyAndValidate(w, r, &ac); err != nil {
			return
		}

		if err := db.Save(&ac).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, ac)
	}
}
