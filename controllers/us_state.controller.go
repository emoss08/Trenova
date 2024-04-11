package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/services"
	tools "github.com/emoss08/trenova/util"
	"github.com/emoss08/trenova/util/types"
)

// GetUsStates gets the us states.
func GetUsStates(w http.ResponseWriter, r *http.Request) {
	usStates, err := services.NewUsStateOps().GetUsStates(r.Context())
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  usStates,
		Count:    0,
		Next:     "",
		Previous: "",
	})
}
