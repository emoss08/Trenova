package services

import (
	"github.com/google/uuid"
	"net/http"
	"trenova/app/middleware"
	"trenova/app/models"
	"trenova/utils"
)

type CRUDService[T any] interface {
	GetAll(orgID, buID uuid.UUID, offset, limit int) ([]T, int64, error)
	GetByID(orgID, buID uuid.UUID, id string) (T, error)
	Create(orgID, buID uuid.UUID, entity T) error
}

func GetEntityHandler[T any](service CRUDService[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)
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

		nextURL := utils.GetNextPageUrl(r, offset, limit, totalRows)
		prevURL := utils.GetPrevPageUrl(r, offset, limit)

		utils.ResponseWithJSON(w, http.StatusOK, models.HTTPResponse{
			Results:  entities,
			Count:    int(totalRows),
			Next:     nextURL,
			Previous: prevURL,
		})
	}
}

func CreateEntityHandler[T any](service CRUDService[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var entity T

		// Retrieve the orgID and buID from the request's context
		orgID := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
		buID := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

		if err := utils.ParseBodyAndValidate(w, r, &entity); err != nil {
			// The ParseBodyAndValidate function will handle the response
			return
		}

		if err := service.Create(orgID, buID, entity); err != nil {
			errorResponse := utils.FormatDatabaseError(err)
			utils.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

			return
		}

		utils.ResponseWithJSON(w, http.StatusCreated, entity)
	}
}
