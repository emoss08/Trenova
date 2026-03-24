package searchhandler

import (
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/types/search"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      serviceports.GlobalSearchService
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service serviceports.GlobalSearchService
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/search")
	api.GET("/global/", h.globalSearch)
}

func (h *Handler) globalSearch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	entityTypes, err := parseEntityTypes(c.Query("entityTypes"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.Search(c.Request.Context(), &serviceports.GlobalSearchRequest{
		Query: strings.TrimSpace(c.Query("query")),
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
		Principal: serviceports.PrincipalInfo{
			Type:     serviceports.PrincipalType(authCtx.PrincipalType),
			ID:       authCtx.PrincipalID,
			UserID:   authCtx.UserID,
			APIKeyID: authCtx.APIKeyID,
		},
		Limit:       helpers.QueryInt(c, "limit", 8),
		EntityTypes: entityTypes,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func parseEntityTypes(value string) ([]search.EntityType, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	parts := strings.Split(value, ",")
	entityTypes := make([]search.EntityType, 0, len(parts))
	seen := make(map[search.EntityType]struct{}, len(parts))
	for _, part := range parts {
		entityType := search.EntityType(strings.TrimSpace(part))
		if entityType == "" {
			continue
		}
		if !isValidEntityType(entityType) {
			return nil, errortypes.NewValidationError(
				"entityTypes",
				errortypes.ErrInvalid,
				"Invalid global search entity type",
			)
		}
		if _, ok := seen[entityType]; ok {
			continue
		}
		seen[entityType] = struct{}{}
		entityTypes = append(entityTypes, entityType)
	}

	return entityTypes, nil
}

func isValidEntityType(entityType search.EntityType) bool {
	switch entityType {
	case search.EntityTypeShipment,
		search.EntityTypeCustomer,
		search.EntityTypeWorker,
		search.EntityTypeDocument:
		return true
	default:
		return false
	}
}
