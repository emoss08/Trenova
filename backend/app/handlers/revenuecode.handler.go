package handlers

import (
	"net/http"
	"trenova-go-backend/app/models"
	"trenova-go-backend/utils"

	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

func GetRevenueCodes(db *gorm.DB, store *gormstore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := utils.GetUserOrgFromSession(r, store)

		if !ok {
			utils.ResponseWithError(w, http.StatusUnauthorized, utils.ValidationErrorDetail{
				Code:   "unauthorized",
				Detail: "Unauthorized",
				Attr:   "organizationId",
			})
			return
		}

		offset, limit, err := utils.PaginationParams(r)
		if err != nil {
			utils.ResponseWithError(w, http.StatusBadRequest, utils.ValidationErrorDetail{
				Code:   "invalid",
				Detail: err.Error(),
				Attr:   "offset, limit",
			})
			return
		}

		var rc models.RevenueCode
		revenueCodes, totalRows, err := rc.GetRevenueCodesByOrgID(db, orgID, offset, limit)
		if err != nil {
			utils.ResponseWithError(w, http.StatusInternalServerError, utils.ValidationErrorDetail{
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

		if revenueCodeId == "" {
			return
		}

		var rc models.RevenueCode

		if err := db.Model(&models.RevenueCode{}).Where("id = ?", revenueCodeId).First(&rc).Error; err != nil {
			utils.ResponseWithError(w, http.StatusNotFound, utils.ValidationErrorDetail{
				Code:   "notFound",
				Detail: "Revenue code not found",
				Attr:   "id",
			})
			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, rc)
	}
}

func UpdateRevenueCode(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get revenue code ID from the URL
		revenueCodeId := utils.GetMuxVar(w, r, "revenueCodeID")
		if revenueCodeId == "" {
			// The GetMuxVar has already handled the response if the ID is missing
			return
		}

		var rc models.RevenueCode
		if err := db.Where("id = ?", revenueCodeId).First(&rc).Error; err != nil {
			utils.ResponseWithError(w, http.StatusNotFound, utils.ValidationErrorDetail{
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
