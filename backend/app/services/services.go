package services

import (
	"net/http"

	"github.com/google/uuid"

	"trenova/app/middleware"
	"trenova/app/models"
	"trenova/utils"
)

type CRUDService[T any] interface {
	GetAll(orgID, buID uuid.UUID, offset, limit int) ([]T, int64, error)
	GetByID(orgID, buID uuid.UUID, id string) (T, error)
	Create(orgID, buID uuid.UUID, entity T) error
	Update(orgID, buID uuid.UUID, id string, entity T) error
}

func GetEntityHandler[T any](service CRUDService[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		buID, ok := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Business Unit ID not found in the request context",
				Attr:   "businessUnitId",
			})
		}

		offset, limit, err := utils.PaginationParams(r)
		if err != nil {
			utils.ResponseWithError(w, http.StatusBadRequest, models.ValidationErrorDetail{
				Code:   "invalid",
				Detail: err.Error(),
				Attr:   "offset, limit",
			})

			return
		}

		entities, totalRows, err := service.GetAll(orgID, buID, offset, limit)
		if err != nil {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "databaseError",
				Detail: err.Error(),
				Attr:   "all",
			})

			return
		}

		nextURL := utils.GetNextPageURL(r, offset, limit, totalRows)
		prevURL := utils.GetPrevPageURL(r, offset, limit)

		utils.ResponseWithJSON(w, http.StatusOK, models.HTTPResponse{
			Results:  entities,
			Count:    int(totalRows),
			Next:     nextURL,
			Previous: prevURL,
		})
	}
}

func CreateEntityHandler[T any](service CRUDService[T], validator *utils.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var entity T

		// Retrieve the orgID and buID from the request's context
		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		buID, ok := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Business Unit ID not found in the request context",
				Attr:   "businessUnitId",
			})
		}

		if err := utils.ParseBodyAndValidate(validator, w, r, &entity); err != nil {
			// The ParseBodyAndValidate function will handle the response
			return
		}

		if err := service.Create(orgID, buID, entity); err != nil {
			errorResponse := utils.CreateDBErrorResponse(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusCreated, entity)
	}
}

func GetEntityByIDHandler[T any](service CRUDService[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entityID := utils.GetMuxVar(w, r, "entityID")

		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		buID, ok := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Business Unit ID not found in the request context",
				Attr:   "businessUnitId",
			})
		}

		entity, err := service.GetByID(orgID, buID, entityID)
		if err != nil {
			errorResponse := utils.CreateDBErrorResponse(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, entity)
	}
}

func UpdateEntityHandler[T any](service CRUDService[T], validator *utils.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entityID := utils.GetMuxVar(w, r, "entityID")

		orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Organization ID not found in the request context",
				Attr:   "organizationId",
			})
		}

		buID, ok := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if !ok {
			utils.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
				Code:   "internalError",
				Detail: "Business Unit ID not found in the request context",
				Attr:   "businessUnitId",
			})
		}

		var entity T
		if err := utils.ParseBodyAndValidate(validator, w, r, &entity); err != nil {
			return
		}

		if err := service.Update(orgID, buID, entityID, entity); err != nil {
			errorResponse := utils.CreateDBErrorResponse(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, entity)
	}
}
