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

// GetUserFavorites returns the user's favorite pages
func GetUserFavorites(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.ContextKeyUserID).(uuid.UUID)

	if !ok {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorDetail{
			Code:   "internalError",
			Detail: "User ID not found in the request context",
			Attr:   "userID",
		})

		return
	}

	favorites, favCount, err := services.NewUserFavoriteOps(r.Context()).GetUserFavorites(userID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results: favorites,
		Count:   favCount,
	})
}

// CreateUserFavorite creates a new user favorite
func CreateUserFavorite(w http.ResponseWriter, r *http.Request) {
	var uf ent.UserFavorite

	buID, ok := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)
	orgID, orgOK := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
	if !ok || !orgOK {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorDetail{
			Code:   "internalError",
			Detail: "Business Unit ID or Organization ID not found in the request context",
			Attr:   "buID, orgID",
		})

		return
	}

	err := tools.ParseBody(r, &uf)
	if err != nil {
		tools.ResponseWithError(w, http.StatusBadRequest, types.ValidationErrorDetail{
			Code:   "invalidRequest",
			Detail: "Invalid request body",
			Attr:   "requestBody",
		})

		return
	}

	uf.OrganizationID = orgID
	uf.BusinessUnitID = buID

	createdFavorite, createErr := services.NewUserFavoriteOps(r.Context()).UserFavoriteCreate(uf)
	if createErr != nil {
		errorResponse := tools.CreateDBErrorResponse(createErr)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createdFavorite)
}

// DeleteUserFavorite deletes a user favorite
func DeleteUserFavorite(w http.ResponseWriter, r *http.Request) {
	var uf ent.UserFavorite

	err := tools.ParseBody(r, &uf)
	if err != nil {
		tools.ResponseWithError(w, http.StatusBadRequest, types.ValidationErrorDetail{
			Code:   "invalidRequest",
			Detail: "Invalid request body",
			Attr:   "requestBody",
		})

		return
	}

	if deleteErr := services.NewUserFavoriteOps(r.Context()).UserFavoriteDelete(uf.UserID, uf.PageLink); deleteErr != nil {
		errorResponse := tools.CreateDBErrorResponse(deleteErr)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)
	}

	tools.ResponseWithJSON(w, http.StatusNoContent, nil)
}
