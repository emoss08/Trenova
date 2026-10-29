package editransferhandler

import (
	"context"
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderchangeservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferchangeservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
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

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*editenderchange.TenderChange], error) {
			return h.tenderChanges.ListTenderChanges(
				c.Request.Context(),
				&editenderports.ListEDITenderChangesRequest{
					Filter:           req,
					RecipientID:      recipientID,
					SourceShipmentID: sourceShipmentID,
					Status: editenderchange.TenderChangeStatus(
						helpers.QueryString(c, "status", ""),
					),
				},
			)
		},
	)
}

func (h *Handler) getTenderChange(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	changeID, err := pulid.MustParse(c.Param("changeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	change, err := h.tenderChanges.GetTenderChange(
		c.Request.Context(),
		editenderports.GetEDITenderChangeByIDRequest{
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

func (h *Handler) applyTenderChange(c *gin.Context) {
	h.tenderChangeAction(c, h.tenderChanges.ApplyTenderChange)
}

func (h *Handler) rejectTenderChange(c *gin.Context) {
	h.tenderChangeAction(c, h.tenderChanges.RejectTenderChange)
}

func (h *Handler) tenderChangeAction(
	c *gin.Context,
	fn func(
		context.Context,
		*editenderchangeservice.TenderChangeActionRequest,
		*permissionports.RequestActor,
	) (*editenderchange.TenderChange, error),
) {
	authCtx := authctx.GetAuthContext(c)
	changeID, err := pulid.MustParse(c.Param("changeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(editenderchangeservice.TenderChangeActionRequest)
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

	helpers.JSON(c, http.StatusOK, change)
}

func (h *Handler) transferChangeAction(
	c *gin.Context,
	fn func(
		context.Context,
		*editransferchangeservice.TransferChangeActionRequest,
		*permissionports.RequestActor,
	) (*edisync.TransferChange, error),
) {
	authCtx := authctx.GetAuthContext(c)
	changeID, err := pulid.MustParse(c.Param("changeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(editransferchangeservice.TransferChangeActionRequest)
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

	helpers.JSON(c, http.StatusOK, change)
}

func (h *Handler) withTransfer(
	c *gin.Context,
	direction string,
	fn func(*gin.Context, *editransfer.EDITransfer),
) {
	authCtx := authctx.GetAuthContext(c)
	transferID, err := pulid.MustParse(c.Param("transferID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	transfer, err := h.transfers.GetTransfer(
		c.Request.Context(),
		&editenderports.GetEDITransferByIDRequest{
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
