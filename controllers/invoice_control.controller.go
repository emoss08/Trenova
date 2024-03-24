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

// GetInvoiceControl gets the invoice control settings for an organization.
func GetInvoiceControl(w http.ResponseWriter, r *http.Request) {
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

	invoiceControl, err := services.NewInvoiceControlOps(r.Context()).GetInvoiceControlByOrgID(orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, invoiceControl)
}

// UpdateInvoiceControl updates the invoice control settings for an organization.
func UpdateInvoiceControl(w http.ResponseWriter, r *http.Request) {
	invoiceControlID := chi.URLParam(r, "invoiceControlID")
	if invoiceControlID == "" {
		return
	}

	var iControlData ent.InvoiceControl

	if err := tools.ParseBodyAndValidate(w, r, &iControlData); err != nil {
		return
	}

	iControlData.ID = uuid.MustParse(invoiceControlID)

	invoiceControl, err := services.NewInvoiceControlOps(r.Context()).UpdateInvoiceControl(iControlData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, invoiceControl)
}
