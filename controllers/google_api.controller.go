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

// GetGoogleAPI gets the accounting control settings for an organization.
func GetGoogleAPI(w http.ResponseWriter, r *http.Request) {
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

	googleAPI, err := services.NewGoogleAPIOps().
		GetGoogleAPI(r.Context(), orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, googleAPI)
}

func UpdateGoogleAPI(w http.ResponseWriter, r *http.Request) {
	googleAPIID := chi.URLParam(r, "googleAPIID")
	if googleAPIID == "" {
		return
	}

	var googleAPIData ent.GoogleApi

	if err := tools.ParseBodyAndValidate(w, r, &googleAPIData); err != nil {
		return
	}

	googleAPIData.ID = uuid.MustParse(googleAPIID)

	googleAPI, err := services.NewGoogleAPIOps().
		UpdateGoogleAPI(r.Context(), googleAPIData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, googleAPI)
}
