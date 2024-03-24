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

// GetTableChangeAlerts gets the table change alerts for an organization.
func GetTableChangeAlerts(w http.ResponseWriter, r *http.Request) {
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

	tableChangeAlerts, count, err := services.NewTableChangeAlertOps(r.Context()).GetTableChangeAlerts(limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  tableChangeAlerts,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateTableChangeALert creates a new table change alert.
func CreateTableChangeALert(w http.ResponseWriter, r *http.Request) {
	var newTableChangeAlert ent.TableChangeAlert

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

	newTableChangeAlert.BusinessUnitID = buID
	newTableChangeAlert.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newTableChangeAlert); err != nil {
		return
	}

	createTableChangeAlert, err := services.NewTableChangeAlertOps(r.Context()).CreateTableChangeAlert(newTableChangeAlert)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createTableChangeAlert)
}

// UpdateTableChangeAlert updates a table change alert by ID.
func UpdateTableChangeAlert(w http.ResponseWriter, r *http.Request) {
	tableChangeAlertID := chi.URLParam(r, "tableChangeAlertID")
	if tableChangeAlertID == "" {
		return
	}

	var tableChangeAlertData ent.TableChangeAlert

	if err := tools.ParseBodyAndValidate(w, r, &tableChangeAlertData); err != nil {
		return
	}

	tableChangeAlertData.ID = uuid.MustParse(tableChangeAlertID)

	serviceType, err := services.NewTableChangeAlertOps(r.Context()).UpdateTableChangeAlert(tableChangeAlertData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, serviceType)
}
