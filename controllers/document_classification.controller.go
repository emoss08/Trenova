package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/services"
	tools "github.com/emoss08/trenova/util"
	"github.com/emoss08/trenova/util/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GetDocumentClassifications gets the document classification for an organization.
func GetDocumentClassifications(w http.ResponseWriter, r *http.Request) {
	offset, limit, err := tools.PaginationParams(r)
	if err != nil {
		tools.ResponseWithError(w, http.StatusBadRequest, types.ValidationErrorResponse{
			Type: "invalidRequest",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "invalidRequest",
					Detail: err.Error(),
					Attr:   "offset, limit",
				},
			},
		})

		return
	}

	orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
	buID, buOK := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

	if !ok || !buOK {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorResponse{
			Type: "internalError",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "internalError",
					Detail: "Organization ID or Business Unit ID not found in the request context",
					Attr:   "orgID, buID",
				},
			},
		})

		return
	}

	documentClassifications, count, err := services.NewDocumentClassificationOps().GetDocumentClassification(r.Context(), limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  documentClassifications,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateDocumentClassification creates a new document classification.
func CreateDocumentClassification(w http.ResponseWriter, r *http.Request) {
	var newDocClass ent.DocumentClassification

	orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
	buID, buOK := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

	if !ok || !buOK {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorResponse{
			Type: "internalError",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "internalError",
					Detail: "Organization ID or Business Unit ID not found in the request context",
					Attr:   "orgID, buID",
				},
			},
		})

		return
	}

	newDocClass.BusinessUnitID = buID
	newDocClass.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newDocClass); err != nil {
		return
	}

	createDocumentClassification, err := services.NewDocumentClassificationOps().CreateDocumentClassification(r.Context(), newDocClass)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createDocumentClassification)
}

// UpdateDocumentClassification updates a document classification by ID.
func UpdateDocumentClassification(w http.ResponseWriter, r *http.Request) {
	docClassID := chi.URLParam(r, "docClassID")
	if docClassID == "" {
		return
	}

	var docClassData ent.DocumentClassification

	if err := tools.ParseBodyAndValidate(w, r, &docClassData); err != nil {
		return
	}

	docClassData.ID = uuid.MustParse(docClassID)

	documentClassification, err := services.NewDocumentClassificationOps().UpdateDocumentClassification(r.Context(), docClassData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, documentClassification)
}
