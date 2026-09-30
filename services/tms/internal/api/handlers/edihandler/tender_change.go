package edihandler

import (
	"context"
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerTenderChangeRoutes(changes *gin.RouterGroup) {
	changes.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTenderChanges,
	)
	changes.GET(
		"/:changeID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTenderChange,
	)
	changes.POST(
		"/:changeID/apply/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.applyTenderChange,
	)
	changes.POST(
		"/:changeID/reject/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.rejectTenderChange,
	)
}

func (h *Handler) listTenderChanges(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	recipientID := pulid.Nil
	if rawRecipientID := c.Query("recipientId"); rawRecipientID != "" {
		parsed, err := pulid.MustParse(rawRecipientID)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		recipientID = parsed
	}
	sourceShipmentID := pulid.Nil
	if rawShipmentID := c.Query("sourceShipmentId"); rawShipmentID != "" {
		parsed, err := pulid.MustParse(rawShipmentID)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		sourceShipmentID = parsed
	}

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.TenderChange], error) {
		return h.service.ListTenderChanges(
			c.Request.Context(),
			&repositories.ListEDITenderChangesRequest{
				Filter:           req,
				RecipientID:      recipientID,
				SourceShipmentID: sourceShipmentID,
				Status:           edi.TenderChangeStatus(helpers.QueryString(c, "status", "")),
			},
		)
	})
}

func (h *Handler) getTenderChange(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	changeID, err := pulid.MustParse(c.Param("changeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	change, err := h.service.GetTenderChange(
		c.Request.Context(),
		repositories.GetEDITenderChangeByIDRequest{
			ID:         changeID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, change)
}

func (h *Handler) applyTenderChange(c *gin.Context) {
	h.tenderChangeAction(c, h.service.ApplyTenderChange)
}

func (h *Handler) rejectTenderChange(c *gin.Context) {
	h.tenderChangeAction(c, h.service.RejectTenderChange)
}

func (h *Handler) tenderChangeAction(
	c *gin.Context,
	fn func(
		context.Context,
		*ediservice.TenderChangeActionRequest,
		*services.RequestActor,
	) (*edi.TenderChange, error),
) {
	authCtx := authctx.GetAuthContext(c)
	changeID, err := pulid.MustParse(c.Param("changeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(ediservice.TenderChangeActionRequest)
	if c.Request.ContentLength > 0 {
		if err = c.ShouldBindJSON(req); err != nil {
			h.eh.HandleError(c, err)
			return
		}
	}
	req.ChangeID = changeID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	change, err := fn(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, change)
}

func (h *Handler) transferChangeAction(
	c *gin.Context,
	fn func(
		context.Context,
		*ediservice.TransferChangeActionRequest,
		*services.RequestActor,
	) (*edi.TransferChange, error),
) {
	authCtx := authctx.GetAuthContext(c)
	changeID, err := pulid.MustParse(c.Param("changeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(ediservice.TransferChangeActionRequest)
	if c.Request.ContentLength > 0 {
		if err = c.ShouldBindJSON(req); err != nil {
			h.eh.HandleError(c, err)
			return
		}
	}
	req.ChangeID = changeID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	change, err := fn(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, change)
}

func (h *Handler) withTransfer(
	c *gin.Context,
	direction string,
	fn func(*gin.Context, *edi.EDITransfer),
) {
	authCtx := authctx.GetAuthContext(c)
	transferID, err := pulid.MustParse(c.Param("transferID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	transfer, err := h.service.GetTransfer(
		c.Request.Context(),
		repositories.GetEDITransferByIDRequest{
			ID:         transferID,
			TenantInfo: pagination.FromAuth(authCtx),
			Direction:  direction,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	fn(c, transfer)
}
