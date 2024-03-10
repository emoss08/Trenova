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
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

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

		nextURL := utils.GetNextPageUrl(r, offset, limit, totalRows)
		prevURL := utils.GetPrevPageUrl(r, offset, limit)

		utils.ResponseWithJSON(w, http.StatusOK, models.HTTPResponse{
			Results:  revenueCodes,
			Count:    int(totalRows),
			Next:     nextURL,
			Previous: prevURL,
		})
	}
}

func CreateRevenueCode(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rc models.RevenueCode

		// Retrieve the orgID and buID from the request's context
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		rc.BusinessUnitID = buID
		rc.OrganizationID = orgID

		if err := utils.ParseBodyAndValidate(w, r, &rc); err != nil {
			return
		}

		if err := db.Create(&rc).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)
			return
		}

		utils.ResponseWithJSON(w, http.StatusCreated, rc)
	}
}

func GetRevenueCodeByID(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		revenueCodeId := utils.GetMuxVar(w, r, "revenueCodeID")

		// Retrieve the orgID and buID from the request's context
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if revenueCodeId == "" {
			return
		}

		var rc models.RevenueCode

		// Fetch the revenue code details for the organization and business unit
		revenueCode, err := rc.FetchRevenueCodeDetails(db, orgID, buID, revenueCodeId)
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

func UpdateRevenueCode(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the orgID and buID from the request's context
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		revenueCodeId := utils.GetMuxVar(w, r, "revenueCodeID")
		if revenueCodeId == "" {
			return
		}

		var rc models.RevenueCode

		// Let's make sure we're updating the right revenue code, for the right organization and business unit
		if err := db.Where("id = ? AND organization_id = ? AND business_unit_id = ?", revenueCodeId, orgID, buID).First(&rc).Error; err != nil {
			utils.ResponseWithError(w, http.StatusNotFound, models.ValidationErrorDetail{
				Code:   "notFound",
				Detail: "Revenue code not found",
				Attr:   "id",
			})
			return
		}

		if err := utils.ParseBodyAndValidate(w, r, &rc); err != nil {
			return
		}

		if err := db.Save(&rc).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)
			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, rc)
	}
}
