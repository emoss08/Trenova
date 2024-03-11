package handlers

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"trenova/app/middleware"
	"trenova/app/models"
	"trenova/utils"
)

func GetEmailProfiles(db *gorm.DB) http.HandlerFunc {
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

		var ep models.EmailProfile
		emailProfiles, totalRows, err := ep.FetchEmailProfilesForOrg(db, orgID, buID, offset, limit)

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
			Results:  emailProfiles,
			Count:    int(totalRows),
			Next:     nextURL,
			Previous: prevURL,
		})
	}
}

func CreateEmailProfile(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ep models.EmailProfile

		// Retrieve the orgID and buID from the request's context
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		ep.BusinessUnitID = buID
		ep.OrganizationID = orgID

		if err := utils.ParseBodyAndValidate(w, r, &ep); err != nil {
			return
		}

		if err := db.Create(&ep).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusCreated, ep)
	}
}

func UpdateEmailProfile(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the orgID and buID from the request's context
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		emailProfileID := utils.GetMuxVar(w, r, "emailProfileID")
		if emailProfileID == "" {
			return
		}

		var ep models.EmailProfile

		// Let's make sure we're updating the right revenue code, for the right organization and business unit
		if err := db.Where("id = ? AND organization_id = ? AND business_unit_id = ?", emailProfileID, orgID, buID).First(&ep).Error; err != nil {
			utils.ResponseWithError(w, http.StatusNotFound, models.ValidationErrorDetail{
				Code:   "notFound",
				Detail: "Revenue code not found",
				Attr:   "id",
			})

			return
		}
		if err := utils.ParseBodyAndValidate(w, r, &ep); err != nil {

			return

		}

		if err := db.Save(&ep).Error; err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, ep)
	}
}
