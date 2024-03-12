package handlers

import (
	"net/http"

	"trenova/app/middleware"
	"trenova/app/models"
	"trenova/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetRevenueCodes(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		buID, buOK := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if !buOK {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Business Unit ID not found in the request context",
				Attr:   "businessUnitId",
			})
		}

		offset, limit, err := utils.PaginationParams(r)
		if err != nil {
			utils.ResponseWithError(w, http.StatusBadRequest, models.ValidationErrorDetail{
				Code:   "invalid",
				Detail: err.Error(),
				Attr:   "offset, limit",
			})

			return
		}

		var rc models.RevenueCode
		revenueCodes, totalRows, err := rc.FetchRevenueCodesForOrg(db, orgID, buID, offset, limit)
		if err != nil {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "databaseError",
				Detail: err.Error(),
				Attr:   "all",
			})

			return
		}

		nextURL := utils.GetNextPageURL(r, offset, limit, totalRows)
		prevURL := utils.GetPrevPageURL(r, offset, limit)

		utils.ResponseWithJSON(w, http.StatusOK, models.HTTPResponse{
			Results:  revenueCodes,
			Count:    int(totalRows),
			Next:     nextURL,
			Previous: prevURL,
		})
	}
}

func CreateRevenueCode(db *gorm.DB, validator *utils.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rc models.RevenueCode

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

		rc.BusinessUnitID = buID
		rc.OrganizationID = orgID

		if err := utils.ParseBodyAndValidate(validator, w, r, &rc); err != nil {
			return
		}

		if err := db.Create(&rc).Error; err != nil {
			errorResponse := utils.CreateDBErrorResponse(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusCreated, rc)
	}
}

func GetRevenueCodeByID(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		revenueCodeID := utils.GetMuxVar(w, r, "revenueCodeID")

		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		buID, buOK := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if !buOK {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Business Unit ID not found in the request context",
				Attr:   "businessUnitId",
			})
		}

		if revenueCodeID == "" {
			return
		}

		var rc models.RevenueCode

		// Fetch the revenue code details for the organization and business unit
		revenueCode, err := rc.FetchRevenueCodeDetails(db, orgID, buID, revenueCodeID)
		if err != nil {
			utils.ResponseWithError(w, http.StatusNotFound, models.ValidationErrorDetail{
				Code:   "notFound",
				Detail: "Revenue code not found",
				Attr:   "id",
			})
			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, revenueCode)
	}
}

func UpdateRevenueCode(db *gorm.DB, validator *utils.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		buID, buOK := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if !buOK {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Business Unit ID not found in the request context",
				Attr:   "businessUnitId",
			})
		}

		revenueCodeID := utils.GetMuxVar(w, r, "revenueCodeID")
		if revenueCodeID == "" {
			return
		}

		var rc models.RevenueCode

		// Let's make sure we're updating the right revenue code, for the right organization and business unit
		if err := db.
			Where("id = ? AND organization_id = ? AND business_unit_id = ?", revenueCodeID, orgID, buID).
			First(&rc).Error; err != nil {
			utils.ResponseWithError(w, http.StatusNotFound, models.ValidationErrorDetail{
				Code:   "notFound",
				Detail: "Revenue code not found",
				Attr:   "id",
			})

			return
		}

		if err := utils.ParseBodyAndValidate(validator, w, r, &rc); err != nil {
			return
		}

		if err := db.Save(&rc).Error; err != nil {
			errorResponse := utils.CreateDBErrorResponse(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, rc)
	}
}
