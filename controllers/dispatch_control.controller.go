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

// GetDispatchControl gets the accounting control settings for an organization.
func GetDispatchControl(w http.ResponseWriter, r *http.Request) {
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

	accountingControl, err := services.NewDispatchControlOps(r.Context()).GetDispatchControl(orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, accountingControl)
}

func UpdateDispatchControl(w http.ResponseWriter, r *http.Request) {
	dispatchControlID := chi.URLParam(r, "dispatchControlID")
	if dispatchControlID == "" {
		return
	}

	var dControlData ent.DispatchControl

	if err := tools.ParseBodyAndValidate(w, r, &dControlData); err != nil {
		return
	}

	dControlData.ID = uuid.MustParse(dispatchControlID)

	dispatchControl, err := services.NewDispatchControlOps(r.Context()).UpdateDispatchControl(dControlData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, dispatchControl)
}
