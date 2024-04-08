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

// GetUserFavorites returns the user's favorite pages.
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

	favorites, favCount, err := services.NewUserFavoriteOps().GetUserFavorites(r.Context(), userID)
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
	var userFavorite ent.UserFavorite

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

	if err := tools.ParseBodyAndValidate(w, r, &userFavorite); err != nil {
		return
	}

	userFavorite.OrganizationID = orgID
	userFavorite.BusinessUnitID = buID

	createdFavorite, createErr := services.NewUserFavoriteOps().UserFavoriteCreate(r.Context(), userFavorite)
	if createErr != nil {
		errorResponse := tools.CreateDBErrorResponse(createErr)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createdFavorite)
}

// DeleteUserFavorite deletes a user favorite.
func DeleteUserFavorite(w http.ResponseWriter, r *http.Request) {
	var userFavorite ent.UserFavorite

	if err := tools.ParseBodyAndValidate(w, r, &userFavorite); err != nil {
		return
	}

	if deleteErr := services.NewUserFavoriteOps().
		UserFavoriteDelete(r.Context(), userFavorite.UserID, userFavorite.PageLink); deleteErr != nil {
		errorResponse := tools.CreateDBErrorResponse(deleteErr)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)
	}

	tools.ResponseWithJSON(w, http.StatusNoContent, nil)
}
