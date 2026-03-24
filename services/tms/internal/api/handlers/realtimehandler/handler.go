package realtimehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      services.RealtimeService
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service services.RealtimeService
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/realtime")
	api.GET("/token-request/", h.getTokenRequest)
}

// @Summary Get realtime token request
// @Description Returns the signed realtime token request for the authenticated actor.
// @ID getRealtimeTokenRequest
// @Tags Realtime
// @Produce json
// @Success 200 {object} services.RealtimeTokenRequest
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /realtime/token-request/ [get]
func (h *Handler) getTokenRequest(c *gin.Context) {
	authContext := authctx.GetAuthContext(c)

	resp, err := h.service.CreateTokenRequest(
		&services.CreateRealtimeTokenRequest{
			UserID:         authContext.UserID,
			OrganizationID: authContext.OrganizationID,
			BusinessUnitID: authContext.BusinessUnitID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}
