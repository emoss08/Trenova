package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/services"
	tools "github.com/emoss08/trenova/util"
	"github.com/emoss08/trenova/util/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GetServiceTypes gets the service types for an organization.
func GetServiceTypes(w http.ResponseWriter, r *http.Request) {
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

	serviceTypes, count, err := services.NewServiceTypeOps().GetServiceTypes(r.Context(), limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  serviceTypes,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateServiceType creates a new service type.
func CreateServiceType(w http.ResponseWriter, r *http.Request) {
	var newServiceType ent.ServiceType

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

	newServiceType.BusinessUnitID = buID
	newServiceType.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newServiceType); err != nil {
		return
	}

	createServiceType, err := services.NewServiceTypeOps().CreateServiceType(r.Context(), newServiceType)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createServiceType)
}

// UpdateServiceType updates a service type by ID.
func UpdateServiceType(w http.ResponseWriter, r *http.Request) {
	serviceTypeID := chi.URLParam(r, "serviceTypeID")
	if serviceTypeID == "" {
		return
	}

	var serviceTypeData ent.ServiceType

	if err := tools.ParseBodyAndValidate(w, r, &serviceTypeData); err != nil {
		return
	}

	serviceTypeData.ID = uuid.MustParse(serviceTypeID)

	serviceType, err := services.NewServiceTypeOps().UpdateServiceType(r.Context(), serviceTypeData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, serviceType)
}
