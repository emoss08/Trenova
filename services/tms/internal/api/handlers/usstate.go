package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/services/usstate"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type UsStateHandlerParams struct {
	fx.In

	Service      *usstate.Service
	ErrorHandler *helpers.ErrorHandler
}

type UsStateHandler struct {
	service *usstate.Service
	eh      *helpers.ErrorHandler
}

func NewUsStateHandler(p UsStateHandlerParams) *UsStateHandler {
	return &UsStateHandler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *UsStateHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/us-states/")
	api.GET("select-options/", h.selectOptions)
}

func (h *UsStateHandler) selectOptions(c *gin.Context) {
	options, err := h.service.SelectOptions(c.Request.Context())
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": options})
}
