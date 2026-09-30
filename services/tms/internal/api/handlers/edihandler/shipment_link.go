package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerShipmentLinkRoutes(links *gin.RouterGroup) {
	links.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listShipmentLinks,
	)
	links.GET(
		"/:linkID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getShipmentLink,
	)
}

func (h *Handler) listShipmentLinks(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.ShipmentLink], error) {
		return h.service.ListShipmentLinks(
			c.Request.Context(),
			&repositories.ListEDIShipmentLinksRequest{Filter: req},
		)
	})
}

func (h *Handler) getShipmentLink(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	linkID, err := pulid.MustParse(c.Param("linkID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	link, err := h.service.GetShipmentLink(
		c.Request.Context(),
		repositories.GetEDIShipmentLinkByIDRequest{
			ID:         linkID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, link)
}
