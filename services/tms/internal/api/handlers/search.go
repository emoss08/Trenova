package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/meilisearch/providers"
	"github.com/emoss08/trenova/pkg/meilisearchtype"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type SearchHandlerParams struct {
	fx.In

	SearchHelper *providers.SearchHelper
	ErrorHandler *helpers.ErrorHandler
}

type SearchHandler struct {
	searchHelper *providers.SearchHelper
	errorHandler *helpers.ErrorHandler
}

func NewSearchHandler(p SearchHandlerParams) *SearchHandler {
	return &SearchHandler{
		searchHelper: p.SearchHelper,
		errorHandler: p.ErrorHandler,
	}
}

func (h *SearchHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/search/")
	api.POST("", h.search)
}

func (h *SearchHandler) search(c *gin.Context) {
	authCtx := context.GetAuthContext(c)
	var req meilisearchtype.SearchRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, &req)
	response, err := h.searchHelper.Search(&req)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}
