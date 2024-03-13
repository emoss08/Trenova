package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/types"
	"github.com/google/uuid"
)

func GetBillingControl(w http.ResponseWriter, r *http.Request) {
	orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

	if !ok {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorDetail{
			Code:   "internalError",
			Detail: "Organization ID not found in the request context",
			Attr:   "organizationId",
		})

		return
	}

	billingControl, err := services.NewBillingControlOps(r.Context()).GetBillingControlByOrgID(orgID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, billingControl)
}

// func GetBillingControl(db *gorm.DB) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

// 		if !ok {
// 			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
// 				Code:   "internalError",
// 				Detail: "Organization ID not found in the request context",
// 				Attr:   "organizationId",
// 			})
// 		}

// 		var bc models.BillingControl
// 		if err := db.Model(&models.BillingControl{}).Where("organization_id = ?", orgID).First(&bc).Error; err != nil {
// 			errorResponse := tools.CreateDBErrorResponse(err)
// 			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

// 			return
// 		}

// 		tools.ResponseWithJSON(w, http.StatusOK, bc)
// 	}
// }

// func UpdateBillingControl(db *gorm.DB, validator *tools.Validator) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

// 		if !ok {
// 			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
// 				Code:   "internalError",
// 				Detail: "Organization ID not found in the request context",
// 				Attr:   "organizationId",
// 			})
// 		}

// 		buID, ok := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

// 		if !ok {
// 			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
// 				Code:   "internalError",
// 				Detail: "Business Unit ID not found in the request context",
// 				Attr:   "businessUnitId",
// 			})
// 		}

// 		var bc models.BillingControl
// 		if err := db.Where("organization_id = ? AND business_unit_id = ?", orgID, buID).First(&bc).Error; err != nil {
// 			errorResponse := tools.CreateDBErrorResponse(err)
// 			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

// 			return
// 		}

// 		if err := tools.ParseBodyAndValidate(validator, w, r, &bc); err != nil {
// 			return
// 		}

// 		if err := db.Save(&bc).Error; err != nil {
// 			errorResponse := tools.CreateDBErrorResponse(err)
// 			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

// 			return
// 		}

// 		tools.ResponseWithJSON(w, http.StatusOK, bc)
// 	}
// }
