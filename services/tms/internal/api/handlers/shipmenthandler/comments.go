package shipmenthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	_ "github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

type shipmentCommentCountResponse struct {
	Count int `json:"count"`
}

// @Summary Get shipment comment count
// @ID getShipmentCommentCount
// @Tags Shipments
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Success 200 {object} shipmentCommentCountResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/comments/count/ [get]
func (h *Handler) getCommentCount(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	count, err := h.commentService.GetCountByShipmentID(
		c.Request.Context(),
		&repositories.GetShipmentCommentCountRequest{
			ShipmentID: shipmentID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, shipmentCommentCountResponse{Count: count})
}

// @Summary List shipment comments
// @ID listShipmentComments
// @Tags Shipments
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]shipment.ShipmentComment]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/comments/ [get]
func (h *Handler) listComments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := pagination.NewQueryOptions(c, authCtx)
	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*shipment.ShipmentComment], error) {
			return h.commentService.ListByShipmentID(
				c.Request.Context(),
				&repositories.ListShipmentCommentsRequest{
					Filter:     req,
					ShipmentID: shipmentID,
				},
			)
		},
	)
}

// @Summary Create a shipment comment
// @ID createShipmentComment
// @Tags Shipments
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param request body shipment.ShipmentComment true "Shipment comment payload"
// @Success 201 {object} shipment.ShipmentComment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/comments/ [post]
func (h *Handler) createComment(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(shipment.ShipmentComment)
	entity.ShipmentID = shipmentID
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.commentService.Create(
		c.Request.Context(),
		entity,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// @Summary Update a shipment comment
// @ID updateShipmentComment
// @Tags Shipments
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param commentID path string true "Comment ID"
// @Param request body shipment.ShipmentComment true "Shipment comment payload"
// @Success 200 {object} shipment.ShipmentComment
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/comments/{commentID}/ [put]
func (h *Handler) updateComment(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	commentID, err := pulid.MustParse(c.Param("commentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(shipment.ShipmentComment)
	entity.ID = commentID
	entity.ShipmentID = shipmentID
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.commentService.Update(
		c.Request.Context(),
		entity,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Delete a shipment comment
// @ID deleteShipmentComment
// @Tags Shipments
// @Param shipmentID path string true "Shipment ID"
// @Param commentID path string true "Comment ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/comments/{commentID}/ [delete]
func (h *Handler) deleteComment(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	shipmentID, err := pulid.MustParse(c.Param("shipmentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	commentID, err := pulid.MustParse(c.Param("commentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.commentService.Delete(
		c.Request.Context(),
		&repositories.DeleteShipmentCommentRequest{
			ShipmentID: shipmentID,
			CommentID:  commentID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
		actorutil.FromAuthContext(authCtx)); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
