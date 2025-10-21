package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	shipmentservice "github.com/emoss08/trenova/internal/core/services/shipment"
	"github.com/emoss08/trenova/internal/core/services/shipmentcomment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type ShipmentHandlerParams struct {
	fx.In

	Service        *shipmentservice.Service
	CommentService *shipmentcomment.Service
	PM             *middleware.PermissionMiddleware
	ErrorHandler   *helpers.ErrorHandler
}

type ShipmentHandler struct {
	service        *shipmentservice.Service
	commentService *shipmentcomment.Service
	pm             *middleware.PermissionMiddleware
	errorHandler   *helpers.ErrorHandler
}

func NewShipmentHandler(p ShipmentHandlerParams) *ShipmentHandler {
	return &ShipmentHandler{
		service:        p.Service,
		commentService: p.CommentService,
		pm:             p.PM,
		errorHandler:   p.ErrorHandler,
	}
}

func (h *ShipmentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/shipments/")
	api.GET("", h.pm.RequirePermission(permission.ResourceShipment, "read"), h.list)
	api.POST("", h.pm.RequirePermission(permission.ResourceShipment, "create"), h.create)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceShipment, "read"), h.get)
	api.POST(
		"duplicate/",
		h.pm.RequirePermission(permission.ResourceShipment, "create"),
		h.duplicate,
	)
	api.POST(
		"previous-rates/",
		h.pm.RequirePermission(permission.ResourceShipment, "read"),
		h.getPreviousRates,
	)
	api.POST(
		"calculate-totals/",
		h.calculateTotals,
	)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceShipment, "update"), h.update)
	api.POST("cancel/", h.pm.RequirePermission(permission.ResourceShipment, "update"), h.cancel)
	api.POST("uncancel/", h.pm.RequirePermission(permission.ResourceShipment, "update"), h.unCancel)
	api.POST(
		"check-for-duplicate-bols/",
		h.pm.RequirePermission(permission.ResourceShipment, "create"),
		h.checkForDuplicateBOLs,
	)
	api.PUT(
		":id/transfer-ownership/",
		h.pm.RequirePermission(permission.ResourceShipment, "update"),
		h.transferOwnership,
	)

	// Live Stream
	api.GET("/live/", h.stream)

	// Shipment Comments
	api.GET(
		":id/comments/count/",
		h.pm.RequirePermission(permission.ResourceShipment, "read"),
		h.getCommentCount,
	)
	api.GET(
		":id/comments/",
		h.pm.RequirePermission(permission.ResourceShipment, "read"),
		h.listComments,
	)
	api.POST(
		":id/comments/",
		h.pm.RequirePermission(permission.ResourceShipment, "create"),
		h.addComment,
	)
	api.PUT(
		":id/comments/:commentID/",
		h.pm.RequirePermission(permission.ResourceShipment, "update"),
		h.updateComment,
	)
	api.DELETE(
		":id/comments/:commentID/",
		h.pm.RequirePermission(permission.ResourceShipment, "delete"),
		h.deleteComment,
	)
}

func (h *ShipmentHandler) list(c *gin.Context) {
	var req repositories.ShipmentOptions

	pagination.Handle[*shipment.Shipment](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		WithExtraParams(&req).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*shipment.Shipment], error) {
			return h.service.List(c.Request.Context(), &repositories.ListShipmentRequest{
				Filter:          opts,
				ShipmentOptions: req,
			})
		})
}

func (h *ShipmentHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetShipmentByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
			ShipmentOptions: repositories.ShipmentOptions{
				ExpandShipmentDetails: helpers.QueryBool(c, "expandShipmentDetails", false),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *ShipmentHandler) getPreviousRates(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	req := new(repositories.GetPreviousRatesRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, req)
	entities, err := h.service.GetPreviousRates(c.Request.Context(), req)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entities)
}

func (h *ShipmentHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(shipment.Shipment)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, entity)
	entity, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *ShipmentHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(shipment.Shipment)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity.ID = id
	context.AddContextToRequest(authCtx, entity)

	entity, err = h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *ShipmentHandler) duplicate(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	req := new(repositories.DuplicateShipmentRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, req)
	if err := h.service.Duplicate(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Shipment duplicate job started",
	})
}

func (h *ShipmentHandler) checkForDuplicateBOLs(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	type bolCheckRequest struct {
		BOL        string    `json:"bol"`
		ShipmentID *pulid.ID `json:"shipmentId,omitempty"`
	}

	req := new(bolCheckRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	if req.BOL == "" {
		c.JSON(http.StatusOK, gin.H{
			"valid": true,
		})
		return
	}

	shp := new(shipment.Shipment)
	shp.BOL = req.BOL
	context.AddContextToRequest(authCtx, req)

	if req.ShipmentID != nil && !req.ShipmentID.IsNil() {
		shp.ID = *req.ShipmentID
	}

	if err := h.service.CheckForDuplicateBOLs(c.Request.Context(), shp); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	type bolCheckResponse struct {
		Valid bool `json:"valid"`
	}

	c.JSON(http.StatusOK, bolCheckResponse{
		Valid: true,
	})
}

func (h *ShipmentHandler) cancel(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	req := new(repositories.CancelShipmentRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, req)
	entity, err := h.service.Cancel(c.Request.Context(), req)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *ShipmentHandler) unCancel(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	req := new(repositories.UnCancelShipmentRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, req)
	entity, err := h.service.UnCancel(c.Request.Context(), req)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *ShipmentHandler) getCommentCount(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	shipmentID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	count, err := h.commentService.GetCountByShipmentID(
		c.Request.Context(),
		repositories.GetShipmentCommentCountRequest{
			ShipmentID: shipmentID,
			OrgID:      authCtx.OrganizationID,
			BuID:       authCtx.BusinessUnitID,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}

func (h *ShipmentHandler) listComments(c *gin.Context) {
	shipmentID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	pagination.Handle[*shipment.ShipmentComment](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*shipment.ShipmentComment], error) {
			return h.commentService.ListByShipmentID(
				c.Request.Context(),
				repositories.GetCommentsByShipmentIDRequest{
					Filter:     opts,
					ShipmentID: shipmentID,
				},
			)
		})
}

func (h *ShipmentHandler) addComment(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	shipmentID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(shipment.ShipmentComment)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity.ShipmentID = shipmentID
	context.AddContextToRequest(authCtx, entity)
	entity, err = h.commentService.Create(c.Request.Context(), entity)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *ShipmentHandler) updateComment(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	commentID, err := pulid.MustParse(c.Param("commentID"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	shipmentID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(shipment.ShipmentComment)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity.ID = commentID
	entity.ShipmentID = shipmentID
	context.AddContextToRequest(authCtx, entity)

	entity, err = h.commentService.Update(c.Request.Context(), entity)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *ShipmentHandler) deleteComment(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	commentID, err := pulid.MustParse(c.Param("commentID"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	shipmentID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	req := new(repositories.DeleteCommentRequest)
	req.CommentID = commentID
	req.ShipmentID = shipmentID
	context.AddContextToRequest(authCtx, req)

	if err = h.commentService.Delete(c.Request.Context(), req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ShipmentHandler) transferOwnership(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	shipmentID, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	req := new(repositories.TransferOwnershipRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	req.ShipmentID = shipmentID
	context.AddContextToRequest(authCtx, req)

	entity, err := h.service.TransferOwnership(c, req)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *ShipmentHandler) calculateTotals(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	shp := new(shipment.Shipment)
	if err := c.ShouldBindJSON(shp); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, shp)
	resp, err := h.service.CalculateTotals(c.Request.Context(), shp, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ShipmentHandler) stream(c *gin.Context) {
	h.service.Stream(c)
}
