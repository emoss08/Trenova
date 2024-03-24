package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/types"
)

// GetUsStates gets the us states.
func GetUsStates(w http.ResponseWriter, r *http.Request) {
	usStates, err := services.NewUsStateOps(r.Context()).GetUsStates()
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
