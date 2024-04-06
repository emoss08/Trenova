package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GetWorkers gets the workers for an organization.
func GetWorkers(w http.ResponseWriter, r *http.Request) {
	offset, limit, err := tools.PaginationParams(r)
	if err != nil {
		tools.ResponseWithError(w, http.StatusBadRequest, types.ValidationErrorResponse{
			Type: "invalidRequest",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "invalidRequest",
					Detail: err.Error(),
					Attr:   "offset, limit",
				},
			},
		})

		return
	}

	fleetCodeID := uuid.MustParse(r.URL.Query().Get("fleet_code_id"))

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

	workers, count, err := services.NewWorkerOps().GetWorkers(
		r.Context(), limit, offset, orgID, buID, fleetCodeID,
	)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  workers,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateWorker creates a new worker.
func CreateWorker(w http.ResponseWriter, r *http.Request) {
	var newWorker services.WorkerRequest

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

	newWorker.BusinessUnitID = buID
	newWorker.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newWorker); err != nil {
		return
	}

	newRecord, err := services.NewWorkerOps().CreateWorker(r.Context(), newWorker)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, newRecord)
}

// UpdateWorker updates a worker by ID.
func UpdateWorker(w http.ResponseWriter, r *http.Request) {
	workerID := chi.URLParam(r, "workerID")
	if workerID == "" {
		return
	}

	var workerData services.WorkerUpdateRequest

	if err := tools.ParseBodyAndValidate(w, r, &workerData); err != nil {
		return
	}

	workerData.ID = uuid.MustParse(workerID)

	tractor, err := services.NewWorkerOps().UpdateWorker(r.Context(), workerData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, tractor)
}
