package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/types"
)

func GetBusinessUnits(w http.ResponseWriter, r *http.Request) {
	offset, limit, err := tools.PaginationParams(r)
	if err != nil {
		tools.ResponseWithError(w, http.StatusBadRequest, types.ValidationErrorDetail{
			Code:   "invalid",
			Detail: err.Error(),
			Attr:   "offset, limit",
		})
		return
	}

	businessUnits, buCount, err := services.NewBusinessUnitOps(r.Context()).GetBusinessUnits(limit, offset)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, offset, limit, buCount)
	prevURL := tools.GetPrevPageURL(r, offset, limit)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  businessUnits,
		Count:    buCount,
		Next:     nextURL,
		Previous: prevURL,
	})
}
