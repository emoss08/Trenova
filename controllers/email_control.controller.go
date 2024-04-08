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

// GetEmailControl gets the email control settings for an organization.
func GetEmailControl(w http.ResponseWriter, r *http.Request) {
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

	emailControl, err := services.NewEmailControlOps().GetEmailControl(r.Context(), orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, emailControl)
}

// UpdateEmailControl updates the email control settings for an organization.
func UpdateEmailControl(w http.ResponseWriter, r *http.Request) {
	emailControlID := chi.URLParam(r, "emailControlID")
	if emailControlID == "" {
		return
	}

	var emailControlData ent.EmailControl

	if err := tools.ParseBodyAndValidate(w, r, &emailControlData); err != nil {
		return
	}

	emailControlData.ID = uuid.MustParse(emailControlID)

	emailControl, err := services.NewEmailControlOps().UpdateEmailControl(r.Context(), emailControlData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, emailControl)
}
