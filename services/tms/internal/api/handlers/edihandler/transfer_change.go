package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerTransferChangeRoutes(changes *gin.RouterGroup) {
	changes.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTransferChanges,
	)
	changes.GET(
		"/:changeID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTransferChange,
	)
	changes.POST(
		"/:changeID/apply/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.applyTransferChange,
	)
	changes.POST(
		"/:changeID/reject/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.rejectTransferChange,
	)
}

func (h *Handler) listTransferChanges(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	linkID := pulid.Nil
	if rawLinkID := c.Query("shipmentLinkId"); rawLinkID != "" {
		parsed, err := pulid.MustParse(rawLinkID)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		linkID = parsed
	}

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edisync.TransferChange], error) {
		return h.transferChanges.ListTransferChanges(
			c.Request.Context(),
			&editenderports.ListEDITransferChangesRequest{
				Filter:         req,
				ShipmentLinkID: linkID,
			},
		)
	})
}

func (h *Handler) getTransferChange(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	changeID, err := pulid.MustParse(c.Param("changeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	change, err := h.transferChanges.GetTransferChange(
		c.Request.Context(),
		editenderports.GetEDITransferChangeByIDRequest{
			ID:         changeID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, change)
}

func (h *Handler) applyTransferChange(c *gin.Context) {
	h.transferChangeAction(c, h.transferChanges.ApplyTransferChange)
}

func (h *Handler) rejectTransferChange(c *gin.Context) {
	h.transferChangeAction(c, h.transferChanges.RejectTransferChange)
}
