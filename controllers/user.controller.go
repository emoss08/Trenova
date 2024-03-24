package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/types"
	"github.com/google/uuid"
)

// GetAuthenticatedUser returns the authenticated user.
func GetAuthenticatedUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextKeyUserID).(uuid.UUID)

	if !ok {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorDetail{
			Code:   "internalError",
			Detail: "User ID not found in the request context",
			Attr:   "userID",
		})

		return
	}

	user, err := services.NewUserOps(r.Context()).GetAuthenticatedUser(userID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, user)
}
