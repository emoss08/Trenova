package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetDispatchControl(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		var dc models.DispatchControl
		if err := db.Model(&models.DispatchControl{}).Where("organization_id = ?", orgID).First(&dc).Error; err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		tools.ResponseWithJSON(w, http.StatusOK, dc)
	}
}

func UpdateDispatchControl(db *gorm.DB, validator *tools.Validator) http.HandlerFunc {
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

		var dc models.DispatchControl
		if err := db.Where("organization_id = ? AND business_unit_id = ?", orgID, buID).First(&dc).Error; err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		if err := tools.ParseBodyAndValidate(validator, w, r, &dc); err != nil {
			return
		}

		if err := db.Save(&dc).Error; err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		tools.ResponseWithJSON(w, http.StatusOK, dc)
	}
}
