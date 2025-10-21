package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/services/classification"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type ClassificationHandlerParams struct {
	fx.In

	Service      *classification.Service
	ErrorHandler *helpers.ErrorHandler
}

type ClassificationHandler struct {
	service      *classification.Service
	errorHandler *helpers.ErrorHandler
}

func NewClassificationHandler(p ClassificationHandlerParams) *ClassificationHandler {
	return &ClassificationHandler{
		service:      p.Service,
		errorHandler: p.ErrorHandler,
	}
}

func (h *ClassificationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/classification/")
	api.POST("location/", h.classifyLocation)
}

func (h *ClassificationHandler) classifyLocation(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var req classification.LocationClassificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, &req.TenantOpts)

	response, err := h.service.ClassifyLocation(c.Request.Context(), &req)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}
