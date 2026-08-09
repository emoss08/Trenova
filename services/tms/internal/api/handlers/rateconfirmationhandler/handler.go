package rateconfirmationhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/rateconfirmationservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *rateconfirmationservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *rateconfirmationservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	moveScoped := rg.Group("/shipment-moves/:moveID/rate-confirmations")
	moveScoped.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceRateConfirmation.String(), permission.OpRead),
		h.listByMove,
	)
	moveScoped.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceRateConfirmation.String(), permission.OpCreate),
		h.generate,
	)

	api := rg.Group("/rate-confirmations")
	api.GET(
		"/:rateConfirmationID/",
		h.pm.RequirePermission(permission.ResourceRateConfirmation.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/:rateConfirmationID/send/",
		h.pm.RequirePermission(permission.ResourceRateConfirmation.String(), permission.OpUpdate),
		h.send,
	)
	api.POST(
		"/:rateConfirmationID/confirm/",
		h.pm.RequirePermission(permission.ResourceRateConfirmation.String(), permission.OpUpdate),
		h.confirm,
	)
	api.POST(
		"/:rateConfirmationID/void/",
		h.pm.RequirePermission(permission.ResourceRateConfirmation.String(), permission.OpUpdate),
		h.void,
	)
}

// @Summary List rate confirmations for a move
// @ID listMoveRateConfirmations
// @Tags RateConfirmations
// @Produce json
// @Param moveID path string true "Shipment move ID"
// @Success 200 {array} rateconfirmation.RateConfirmation
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-moves/{moveID}/rate-confirmations/ [get]
func (h *Handler) listByMove(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	moveID, err := pulid.MustParse(c.Param("moveID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entities, err := h.service.ListByMove(
		c.Request.Context(),
		&repositories.ListRateConfirmationsByMoveRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			ShipmentMoveID: moveID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entities)
}

// @Summary Generate a rate confirmation for a move's carrier assignment
// @ID generateMoveRateConfirmation
// @Tags RateConfirmations
// @Produce json
// @Param moveID path string true "Shipment move ID"
// @Success 201 {object} rateconfirmation.RateConfirmation
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipment-moves/{moveID}/rate-confirmations/ [post]
func (h *Handler) generate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	moveID, err := pulid.MustParse(c.Param("moveID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	entity, err := h.service.Generate(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		moveID,
		actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

// @Summary Get a rate confirmation
// @ID getRateConfirmation
// @Tags RateConfirmations
// @Produce json
// @Param rateConfirmationID path string true "Rate confirmation ID"
// @Success 200 {object} rateconfirmation.RateConfirmation
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-confirmations/{rateConfirmationID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	rateConfirmationID, err := pulid.MustParse(c.Param("rateConfirmationID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetRateConfirmationByIDRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			RateConfirmationID: rateConfirmationID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Send a rate confirmation to the carrier
// @ID sendRateConfirmation
// @Tags RateConfirmations
// @Produce json
// @Param rateConfirmationID path string true "Rate confirmation ID"
// @Success 200 {object} rateconfirmation.RateConfirmation
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-confirmations/{rateConfirmationID}/send/ [post]
func (h *Handler) send(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	rateConfirmationID, err := pulid.MustParse(c.Param("rateConfirmationID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Send(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		rateConfirmationID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

type confirmRequest struct {
	ConfirmedByName string `json:"confirmedByName"`
}

// @Summary Mark a rate confirmation as confirmed by the carrier
// @ID confirmRateConfirmation
// @Tags RateConfirmations
// @Accept json
// @Produce json
// @Param rateConfirmationID path string true "Rate confirmation ID"
// @Param request body confirmRequest true "Confirmation payload"
// @Success 200 {object} rateconfirmation.RateConfirmation
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-confirmations/{rateConfirmationID}/confirm/ [post]
func (h *Handler) confirm(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	rateConfirmationID, err := pulid.MustParse(c.Param("rateConfirmationID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	body := new(confirmRequest)
	if err = c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.MarkConfirmed(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		rateConfirmationID,
		body.ConfirmedByName,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

type voidRequest struct {
	Reason string `json:"reason"`
}

// @Summary Void a rate confirmation
// @ID voidRateConfirmation
// @Tags RateConfirmations
// @Accept json
// @Produce json
// @Param rateConfirmationID path string true "Rate confirmation ID"
// @Param request body voidRequest true "Void payload"
// @Success 200 {object} rateconfirmation.RateConfirmation
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-confirmations/{rateConfirmationID}/void/ [post]
func (h *Handler) void(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	rateConfirmationID, err := pulid.MustParse(c.Param("rateConfirmationID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	body := new(voidRequest)
	if err = c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Void(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		rateConfirmationID,
		body.Reason,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
