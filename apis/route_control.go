package handlers

import (
	"github.com/emoss08/trenova/tools"
	"net/http"

	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetRouteControl(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		var routeControl models.RouteControl
		if err := db.
			Model(&models.RouteControl{}).
			Where("organization_id = ?", orgID).First(&routeControl).Error; err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		tools.ResponseWithJSON(w, http.StatusOK, routeControl)
	}
}

func UpdateRouteControl(db *gorm.DB, validator *tools.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		buID, buOk := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if !buOk {
			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Business Unit ID not found in the request context",
				Attr:   "businessUnitId",
			})
		}

		var routeControl models.RouteControl
		if err := db.
			Where("organization_id = ? AND business_unit_id = ?", orgID, buID).
			First(&routeControl).Error; err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		if err := tools.ParseBodyAndValidate(validator, w, r, &routeControl); err != nil {
			return
		}

		if err := db.Save(&routeControl).Error; err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		tools.ResponseWithJSON(w, http.StatusOK, routeControl)
	}
}
