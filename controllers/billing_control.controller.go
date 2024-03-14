package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/types"
	"github.com/google/uuid"
)

// GetBillingControl gets the invoice control settings for an organization
func GetBillingControl(w http.ResponseWriter, r *http.Request) {
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

	billingControl, err := services.NewBillingControlOps(r.Context()).GetBillingControl(orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, billingControl)
}

func UpdateBillingControl(w http.ResponseWriter, r *http.Request) {
	billingControlID := tools.GetMuxVar(w, r, "billingControlID")
	if billingControlID == "" {
		return
	}

	var bControlData ent.BillingControl

	err := tools.ParseBody(r, &bControlData)
	if err != nil {
		tools.ResponseWithError(w, http.StatusBadRequest, types.ValidationErrorResponse{
			Type: "validationError",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "invalidRequest",
					Detail: "Invalid request body",
					Attr:   "requestBody",
				},
			},
		})

		return
	}

	bControlData.ID = uuid.MustParse(billingControlID)

	billingControl, err := services.NewBillingControlOps(r.Context()).UpdateBillingControl(bControlData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, billingControl)
}
