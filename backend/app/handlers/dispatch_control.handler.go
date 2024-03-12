package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"trenova/app/middleware"
	"trenova/app/models"
	"trenova/utils"
)

func GetDispatchControl(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		var dc models.DispatchControl
		if err := db.Model(&models.DispatchControl{}).Where("organization_id = ?", orgID).First(&dc).Error; err != nil {
			errorResponse := utils.CreateDBErrorResponse(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, dc)
	}
}

func UpdateDispatchControl(db *gorm.DB, validator *utils.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		buID, ok := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Business Unit ID not found in the request context",
				Attr:   "businessUnitId",
			})
		}

		var dc models.DispatchControl
		if err := db.Where("organization_id = ? AND business_unit_id = ?", orgID, buID).First(&dc).Error; err != nil {
			errorResponse := utils.CreateDBErrorResponse(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		if err := utils.ParseBodyAndValidate(validator, w, r, &dc); err != nil {
			return
		}

		if err := db.Save(&dc).Error; err != nil {
			errorResponse := utils.CreateDBErrorResponse(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, dc)
	}
}
