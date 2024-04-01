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

// GetAccountingControl gets the accounting control settings for an organization.
func GetAccountingControl(w http.ResponseWriter, r *http.Request) {
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

	accountingControl, err := services.NewAccountingControlOps().GetAccountingControl(r.Context(), orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, accountingControl)
}

func UpdateAccountingControl(w http.ResponseWriter, r *http.Request) {
	accountingControlID := chi.URLParam(r, "accountingControlID")
	if accountingControlID == "" {
		return
	}

	var aControlData ent.AccountingControl

	if err := tools.ParseBodyAndValidate(w, r, &aControlData); err != nil {
		return
	}

	aControlData.ID = uuid.MustParse(accountingControlID)

	accountingControl, err := services.NewAccountingControlOps().UpdateAccountingControl(r.Context(), aControlData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, accountingControl)
}
