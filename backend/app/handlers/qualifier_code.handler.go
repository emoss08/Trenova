package handlers

import (
	"net/http"

	"trenova/app/middleware"
	"trenova/app/models"
	"trenova/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetQualifierCodes(db *gorm.DB) http.HandlerFunc {
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

		var qc models.QualifierCode
		qualifierCodes, totalRows, err := qc.FetchQualifierCodesForOrg(db, orgID, buID, offset, limit)
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
			Results:  qualifierCodes,
			Count:    int(totalRows),
			Next:     nextURL,
			Previous: prevURL,
		})
	}
}

func GetQualifierCodeByID(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		qualifierCodeID := utils.GetMuxVar(w, r, "qualifierCodeID")

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

		if qualifierCodeID == "" {
			return
		}

		var qc models.QualifierCode

		// Fetch the revenue code details for the organization and business unit
		qualifierCode, err := qc.FetchQualifierCodeDetails(db, orgID, buID, qualifierCodeID)
		if err != nil {
			utils.ResponseWithError(w, http.StatusNotFound, models.ValidationErrorDetail{
				Code:   "notFound",
				Detail: "Revenue code not found",
				Attr:   "id",
			})
			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, qualifierCode)
	}
}

func CreateQualifierCode(db *gorm.DB, validator *utils.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var qc models.QualifierCode

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

		qc.BusinessUnitID = buID
		qc.OrganizationID = orgID

		if err := utils.ParseBodyAndValidate(validator, w, r, &qc); err != nil {
			return
		}

		if err := db.Create(&qc).Error; err != nil {
			errorResponse := utils.CreateDBErrorResponse(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusCreated, qc)
	}
}

func UpdateQualifierCode(db *gorm.DB, validator *utils.Validator) http.HandlerFunc {
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

		qualifierCodeID := utils.GetMuxVar(w, r, "qualifierCodeID")
		if qualifierCodeID == "" {
			return
		}

		var qc models.QualifierCode

		if err := db.
			Where("id = ? AND organization_id = ? AND business_unit_id = ?", qualifierCodeID, orgID, buID).
			First(&qc).Error; err != nil {
			utils.ResponseWithError(w, http.StatusNotFound, models.ValidationErrorDetail{
				Code:   "notFound",
				Detail: "Revenue code not found",
				Attr:   "id",
			})

			return
		}

		if err := utils.ParseBodyAndValidate(validator, w, r, &qc); err != nil {
			return
		}

		if err := db.Save(&qc).Error; err != nil {
			errorResponse := utils.CreateDBErrorResponse(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, qc)
	}
}
