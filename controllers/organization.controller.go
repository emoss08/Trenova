package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/types"
	"github.com/google/uuid"
)

// GetUserOrganization returns the organization of the user.
func GetUserOrganization(w http.ResponseWriter, r *http.Request) {
	buID, buOK := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)
	orgID, orgOK := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

	if !buOK || !orgOK {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorResponse{
			Type: "internalError",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "internalError",
					Detail: "User ID, Business Unit ID, or Organization ID not found in the request context",
					Attr:   "userID, buID, orgID",
				},
			},
		})

		return
	}

	user, err := services.NewOrganizationOps(r.Context()).GetUserOrganization(buID, orgID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, user)
}
