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

// GetRouteControl gets the route control settings for an organization.
func GetRouteControl(w http.ResponseWriter, r *http.Request) {
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

	routeControl, err := services.NewRouteControlOps(r.Context()).GetRouteControl(orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, routeControl)
}

// UpdateRouteControl updates the shipment control settings for an organization.
func UpdateRouteControl(w http.ResponseWriter, r *http.Request) {
	routeControlID := chi.URLParam(r, "routeControlID")
	if routeControlID == "" {
		return
	}

	var rControlData ent.RouteControl

	if err := tools.ParseBodyAndValidate(w, r, &rControlData); err != nil {
		return
	}

	rControlData.ID = uuid.MustParse(routeControlID)

	invoiceControl, err := services.NewRouteControlOps(r.Context()).UpdateRouteControl(rControlData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, invoiceControl)
}
