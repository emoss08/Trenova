package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/session"
	"github.com/emoss08/trenova/tools/types"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	store, storeErr := session.GetStore()
	if storeErr != nil {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorDetail{
			Code:   "sessionError",
			Detail: storeErr.Error(),
			Attr:   "session",
		})
		return
	}

	var loginDetails struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := tools.ParseBody(r, &loginDetails); err != nil {
		tools.ResponseWithError(w, http.StatusBadRequest, types.ValidationErrorDetail{
			Code:   "invalidRequest",
			Detail: "Invalid request body",
			Attr:   "body",
		})
		return
	}

	user, err := services.NewLoginOps(r.Context()).AuthenticateUser(loginDetails.Username, loginDetails.Password)
	if err != nil {
		tools.ResponseWithError(w, http.StatusUnauthorized, types.ValidationErrorDetail{
			Code:   "invalidCredentials",
			Detail: "Invalid username or password",
			Attr:   "username",
		})
		return
	}

	// Get the session ID from the system
	sessionID := tools.GetSystemSessionID()
	session, sessionErr := store.Get(r, sessionID)
	if sessionErr != nil {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorDetail{
			Code:   "sessionError",
			Detail: "Failed to retrieve session",
			Attr:   "session",
		})
		return
	}

	// Set the session values
	session.Values["userID"] = user.ID
	session.Values["organizationID"] = user.OrganizationID
	session.Values["businessUnitID"] = user.BusinessUnitID

	// Save the session
	if err := session.Save(r, w); err != nil {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorDetail{
			Code:   "sessionError",
			Detail: "Failed to save session",
			Attr:   "session",
		})
		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, "Login successful")
}
