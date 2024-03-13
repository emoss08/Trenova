package services

import (
	"github.com/emoss08/trenova/tools"
	"log"
	"net/http"

	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/models"
	"github.com/google/uuid"
)

type OrgBuSetter interface {
	SetOrgID(uuid.UUID)
	SetBuID(uuid.UUID)
}
type CRUDService[T any] interface {
	GetAll(orgID, buID uuid.UUID, offset, limit int) ([]T, int64, error)
	GetByID(orgID, buID uuid.UUID, id string) (T, error)
	Create(orgID, buID uuid.UUID, entity T) error
	Update(orgID, buID uuid.UUID, id string, entity T) error
}

func getOrgAndBuIDFromContext(r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}

	buID, ok := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}

	return orgID, buID, true
}

// func GetEntityHandler[T any](service CRUDService[T]) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		orgID, buID, ok := getOrgAndBuIDFromContext(r)
// 		if !ok {
// 			tools.ResponseWithError(w, http.StatusBadRequest, models.ValidationErrorDetail{
// 				Code:   "contextError",
// 				Detail: "Organization ID or Business Unit ID not found in the request context",
// 			})
// 			return
// 		}

// 		offset, limit, err := tools.PaginationParams(r)
// 		if err != nil {
// 			tools.ResponseWithError(w, http.StatusBadRequest, models.ValidationErrorDetail{
// 				Code:   "invalid",
// 				Detail: err.Error(),
// 				Attr:   "offset, limit",
// 			})
// 			return
// 		}

// 		entities, totalRows, err := service.GetAll(orgID, buID, offset, limit)
// 		if err != nil {
// 			tools.ResponseWithError(w, http.StatusInternalServerError, models.ValidationErrorDetail{
// 				Code:   "databaseError",
// 				Detail: err.Error(),
// 			})
// 			return
// 		}

// 		nextURL := tools.GetNextPageURL(r, offset, limit, totalRows)
// 		prevURL := tools.GetPrevPageURL(r, offset, limit)

// 		tools.ResponseWithJSON(w, http.StatusOK, models.HTTPResponse{
// 			Results:  entities,
// 			Count:    int(totalRows),
// 			Next:     nextURL,
// 			Previous: prevURL,
// 		})
// 	}
// }

func GetEntityByIDHandler[T any](service CRUDService[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entityID := tools.GetMuxVar(w, r, "entityID")

		orgID, buID, ok := getOrgAndBuIDFromContext(r)
		if !ok {
			tools.ResponseWithError(w, http.StatusBadRequest, models.ValidationErrorDetail{
				Code:   "contextError",
				Detail: "Organization ID or Business Unit ID not found in the request context",
			})
			return
		}

		entity, err := service.GetByID(orgID, buID, entityID)
		if err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)
			return
		}

		tools.ResponseWithJSON(w, http.StatusOK, entity)
	}
}

func CreateEntityHandler[T any](service CRUDService[T], validator *tools.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var entity T
		entityPtr := &entity // Work with a pointer to T

		orgID, buID, ok := getOrgAndBuIDFromContext(r)
		if !ok {
			tools.ResponseWithError(w, http.StatusBadRequest, models.ValidationErrorDetail{
				Code:   "contextError",
				Detail: "Organization ID or Business Unit ID not found in the request context",
			})
			return
		}

		// After parsing, check if the entity implements OrgBuSetter and set the IDs
		if setter, setterOK := any(entityPtr).(OrgBuSetter); setterOK {
			setter.SetOrgID(orgID)
			setter.SetBuID(buID)
		} else {
			log.Printf("Entity of type %T does not implement OrgBuSetter", entity)
		}

		if err := tools.ParseBodyAndValidate(validator, w, r, entityPtr); err != nil {
			// The ParseBodyAndValidate function will handle the response
			return
		}

		if err := service.Create(orgID, buID, *entityPtr); err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)
			return
		}

		tools.ResponseWithJSON(w, http.StatusCreated, entity)
	}
}

func UpdateEntityHandler[T any](service CRUDService[T], validator *tools.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entityID := tools.GetMuxVar(w, r, "entityID")

		var entity T
		entityPtr := &entity // Work with a pointer to T

		orgID, buID, ok := getOrgAndBuIDFromContext(r)
		if !ok {
			tools.ResponseWithError(w, http.StatusBadRequest, models.ValidationErrorDetail{
				Code:   "contextError",
				Detail: "Organization ID or Business Unit ID not found in the request context",
			})
			return
		}

		// After parsing, check if the entity implements OrgBuSetter and set the IDs
		if setter, setterOK := any(entityPtr).(OrgBuSetter); setterOK {
			setter.SetOrgID(orgID)
			setter.SetBuID(buID)
		} else {
			log.Printf("Entity of type %T does not implement OrgBuSetter", entity)
		}

		if err := tools.ParseBodyAndValidate(validator, w, r, entityPtr); err != nil {
			// The ParseBodyAndValidate function will handle the response
			return
		}

		if err := service.Update(orgID, buID, entityID, *entityPtr); err != nil {
			errorResponse := tools.CreateDBErrorResponse(err)
			tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)
			return
		}

		// Return 204 No Content
		w.WriteHeader(http.StatusNoContent)
	}
}
