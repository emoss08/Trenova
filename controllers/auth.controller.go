package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/session"
	"github.com/emoss08/trenova/tools/types"
)

// LoginHandler handles the login request and authenticates the user.
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var loginRequest struct {
		Username string `json:"username" validate:"required"`
		Password string `json:"password" validate:"required"`
	}

	store, storeErr := session.GetStore()
	if storeErr != nil {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorDetail{
			Code:   "sessionError",
			Detail: storeErr.Error(),
			Attr:   "session",
		})
		return
	}

	if err := tools.ParseBodyAndValidate(w, r, &loginRequest); err != nil {
		return
	}

	user, err := services.NewLoginOps().AuthenticateUser(r.Context(), loginRequest.Username, loginRequest.Password)
	if err != nil {
		tools.ResponseWithError(w, http.StatusUnauthorized, types.ValidationErrorResponse{
			Type: "validationError",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "authenticationError",
					Detail: "Invalid username or password",
					Attr:   "username",
				},
				{
					Code:   "authenticationError",
					Detail: "Invalid username or password",
					Attr:   "password",
				},
			},
		})
		return
	}

	// Get the session ID from the system
	sessionName := tools.GetSystemSessionName()
	session, sessionErr := store.Get(r, sessionName)
	if sessionErr != nil {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorResponse{
			Type: "severError",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "sessionError",
					Detail: sessionErr.Error(),
					Attr:   "session",
				},
			},
		})
		return
	}

	// Set the session values
	session.Values["userID"] = user.ID
	session.Values["organizationID"] = user.OrganizationID
	session.Values["businessUnitID"] = user.BusinessUnitID

	// Save the session
	if saveErr := session.Save(r, w); saveErr != nil {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorResponse{
			Type: "serverError",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "sessionError",
					Detail: "Failed to save session",
					Attr:   "session",
				},
			},
		})
		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, "Login successful")
}

// LogoutHandler handles the logout request and invalidates the session.
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	store, storeErr := session.GetStore()
	if storeErr != nil {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorDetail{
			Code:   "sessionError",
			Detail: storeErr.Error(),
			Attr:   "session",
		})
		return
	}

	sessionName := tools.GetSystemSessionName()
	session, sessionErr := store.Get(r, sessionName)

	if sessionErr != nil {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorResponse{
			Type: "severError",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "sessionError",
					Detail: "Failed to retrieve session",
					Attr:   "session",
				},
			},
		})

		return
	}

	// Invalidate the session by setting the MaxAge to -1
	session.Options.MaxAge = -1

	// Save the session to update the session in the database and delete client's cookie
	if err := session.Save(r, w); err != nil {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorResponse{
			Type: "serverError",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "sessionError",
					Detail: "Failed to save session",
					Attr:   "session",
				},
			},
		})

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, "Logout successful")
}
