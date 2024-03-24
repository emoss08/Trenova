package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GetFeasibilityToolControl gets the accounting control settings for an organization.
func GetFeasibilityToolControl(w http.ResponseWriter, r *http.Request) {
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

	feasibilityToolControl, err := services.NewFeasibilityControlOps(r.Context()).GetFeasibilityToolControl(orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, feasibilityToolControl)
}

func UpdateFeasibilityToolControl(w http.ResponseWriter, r *http.Request) {
	feasibilityToolControlID := chi.URLParam(r, "feasibilityToolControlID")
	if feasibilityToolControlID == "" {
		return
	}

	var ftControlData ent.FeasibilityToolControl

	if err := tools.ParseBodyAndValidate(w, r, &ftControlData); err != nil {
		return
	}

	ftControlData.ID = uuid.MustParse(feasibilityToolControlID)

	feasibilityToolControl, err := services.NewFeasibilityControlOps(r.Context()).UpdateFeasibilityToolControl(ftControlData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, feasibilityToolControl)
}
