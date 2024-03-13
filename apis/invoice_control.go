package handlers

import (
	"github.com/emoss08/trenova/tools"
	"net/http"

	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetInvoiceControl(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		var ic models.InvoiceControl
		if err := db.
			Model(&models.InvoiceControl{}).
			Where("organization_id = ?", orgID).First(&ic).Error; err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		tools.ResponseWithJSON(w, http.StatusOK, ic)
	}
}

func UpdateInvoiceControl(db *gorm.DB, validator *tools.Validator) http.HandlerFunc {
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

		var ic models.InvoiceControl
		if err := db.Where("organization_id = ? AND business_unit_id = ?", orgID, buID).First(&ic).Error; err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		if err := tools.ParseBodyAndValidate(validator, w, r, &ic); err != nil {
			return
		}

		if err := db.Save(&ic).Error; err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		tools.ResponseWithJSON(w, http.StatusOK, ic)
	}
}
