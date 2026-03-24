package googlemapshandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	AutoCompleteService services.AutoCompleteService
	ErrorHandler        *helpers.ErrorHandler
}

type Handler struct {
	service services.AutoCompleteService
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.AutoCompleteService,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/google-maps")
	api.POST("/autocomplete/", h.autocomplete)
}

// @Summary Autocomplete a location
// @Description Resolves Google Maps place details for the provided autocomplete input.
// @ID autocompleteLocation
// @Tags Google Maps
// @Accept json
// @Produce json
// @Param request body services.AutoCompleteRequest true "Autocomplete request"
// @Success 200 {object} services.AutocompleteLocationResult
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /google-maps/autocomplete/ [post]
func (h *Handler) autocomplete(c *gin.Context) {
	ac := authctx.GetAuthContext(c)

	var req services.AutoCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  ac.OrganizationID,
		BuID:   ac.BusinessUnitID,
		UserID: ac.UserID,
	}

	resp, err := h.service.GetPlaceDetails(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
