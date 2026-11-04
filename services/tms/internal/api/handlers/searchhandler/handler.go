package searchhandler

import (
	"net/http"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/types/search"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      kernelports.GlobalSearchService
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service kernelports.GlobalSearchService
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
	result, err := h.service.Search(c.Request.Context(), &kernelports.GlobalSearchRequest{
		Query:      strings.TrimSpace(c.Query("query")),
		TenantInfo: pagination.FromAuth(authCtx),
		Principal: kernelports.PrincipalInfo{
			Type:     kernelports.PrincipalType(authCtx.PrincipalType),
			ID:       authCtx.PrincipalID,
			UserID:   authCtx.UserID,
			APIKeyID: authCtx.APIKeyID,
		},
		Limit:       helpers.QueryInt(c, "limit", 8),
		EntityTypes: parseEntityTypes(c.Query("entityTypes")),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}

// parseEntityTypes reads the comma-separated kinds, each once. Whether the
// search finds a kind is the search service's to say.
func parseEntityTypes(value string) []search.EntityType {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	entityTypes := make([]search.EntityType, 0, len(parts))
	for _, part := range parts {
		entityType := search.EntityType(strings.TrimSpace(part))
		if entityType == "" || slices.Contains(entityTypes, entityType) {
			continue
		}
		entityTypes = append(entityTypes, entityType)
	}

	return entityTypes
}
