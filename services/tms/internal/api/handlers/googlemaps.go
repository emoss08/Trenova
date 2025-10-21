package handlers

import (
	"errors"
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/googlemaps"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type GoogleMapsHandlerParams struct {
	fx.In

	AutoCompleteService *googlemaps.AutoCompleteService
	Config              *config.Config
	ErrorHandler        *helpers.ErrorHandler
}

type GoogleMapsHandler struct {
	autoCompleteService *googlemaps.AutoCompleteService
	config              *config.Config
	eh                  *helpers.ErrorHandler
}

func NewGoogleMapsHandler(p GoogleMapsHandlerParams) *GoogleMapsHandler {
	return &GoogleMapsHandler{
		autoCompleteService: p.AutoCompleteService,
		config:              p.Config,
		eh:                  p.ErrorHandler,
	}
}

func (h *GoogleMapsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/google-maps/")
	api.POST("/autocomplete/", h.autocomplete)
	api.GET("/api-key/", h.getAPIKey)
}

func (h *GoogleMapsHandler) autocomplete(c *gin.Context) {
	var req googlemaps.AutoCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	response, err := h.autoCompleteService.GetPlaceDetails(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *GoogleMapsHandler) getAPIKey(c *gin.Context) {
	apiKey := h.config.Google.APIKey
	if apiKey == "" {
		h.eh.HandleError(c, errors.New("API key is not set"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"apiKey": apiKey})
}
