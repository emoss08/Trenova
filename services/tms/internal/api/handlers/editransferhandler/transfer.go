package editransferhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerTransferRoutes(transfers *gin.RouterGroup) {
	transfers.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTransfers,
	)
	transfers.GET(
		"/:transferID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTransfer,
	)
	transfers.GET(
		"/:transferID/mapping-preview/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.mappingPreview,
	)
	transfers.POST(
		"/:transferID/approve/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.approveTransfer,
	)
	transfers.POST(
		"/:transferID/reject/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.rejectTransfer,
	)
	transfers.POST(
		"/bulk-approve/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.bulkApproveTransfers,
	)
	transfers.POST(
		"/bulk-reject/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.bulkRejectTransfers,
	)
	transfers.POST(
		"/:transferID/cancel/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.cancelTransfer,
	)
	transfers.POST(
		"/:transferID/expire/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.expireTransfer,
	)
}

func (h *Handler) resetControlNumber(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(edipartnerports.ResetEDIControlNumberRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	sequence, err := h.setup.ResetControlNumber(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, sequence)
}

func (h *Handler) bulkApproveTransfers(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(editransferservice.BulkApproveTransfersRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	result, err := h.transfers.BulkApproveTransfers(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) bulkRejectTransfers(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(editransferservice.BulkRejectTransfersRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	result, err := h.transfers.BulkRejectTransfers(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) submitLoadTender(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(editenderservice.SubmitLoadTenderRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	transfer, err := h.tenders.SubmitLoadTender(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusCreated, transfer)
}

func (h *Handler) listInboundTransfers(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	inboundFileID := pulid.Nil
	if raw := c.Query("inboundFileId"); raw != "" {
		parsed, err := pulid.Parse(raw)
		if err != nil {
			h.eh.HandleError(c, errortypes.NewValidationError(
				"inboundFileId",
				errortypes.ErrInvalid,
				"Inbound file ID is not a valid ID",
			))
			return
		}
		inboundFileID = parsed
	}

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*editransfer.EDITransfer], error) {
		return h.transfers.ListInboundTransfers(
			c.Request.Context(),
			&editenderports.ListEDITransfersRequest{Filter: req, InboundFileID: inboundFileID},
		)
	})
}

func (h *Handler) listTransfers(c *gin.Context) {
	direction := c.Query("direction")
	switch direction {
	case "inbound":
		h.listInboundTransfers(c)
	case "outbound":
		h.listOutboundTransfers(c)
	default:
		h.eh.HandleError(c, errortypes.NewValidationError(
			"direction",
			errortypes.ErrInvalid,
			"Direction must be inbound or outbound",
		))
	}
}

func (h *Handler) listOutboundTransfers(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*editransfer.EDITransfer], error) {
		return h.transfers.ListOutboundTransfers(
			c.Request.Context(),
			&editenderports.ListEDITransfersRequest{Filter: req},
		)
	})
}

func (h *Handler) getTransfer(c *gin.Context) {
	h.withTransfer(c, "", func(gCtx *gin.Context, transfer *editransfer.EDITransfer) {
		helpers.JSON(gCtx, http.StatusOK, transfer)
	})
}

func (h *Handler) mappingPreview(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	transferID, err := pulid.MustParse(c.Param("transferID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	preview, err := h.transfers.MappingPreview(
		c.Request.Context(),
		&editenderports.GetEDITransferByIDRequest{
			ID:         transferID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, preview)
}

func (h *Handler) approveTransfer(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	transferID, err := pulid.MustParse(c.Param("transferID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(editransferservice.ApproveTransferRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TransferID = transferID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	transfer, err := h.transfers.ApproveTransfer(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusAccepted, transfer)
}

func (h *Handler) rejectTransfer(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	transferID, err := pulid.MustParse(c.Param("transferID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(editransferservice.RejectTransferRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TransferID = transferID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	transfer, err := h.transfers.RejectTransfer(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, transfer)
}

func (h *Handler) cancelTransfer(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	transferID, err := pulid.MustParse(c.Param("transferID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	transfer, err := h.transfers.CancelTransfer(
		c.Request.Context(),
		&editransferservice.CancelTransferRequest{
			TransferID: transferID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, transfer)
}

func (h *Handler) expireTransfer(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	transferID, err := pulid.MustParse(c.Param("transferID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	transfer, err := h.transfers.ExpireTransfer(
		c.Request.Context(),
		&editransferservice.ExpireTransferRequest{
			TransferID: transferID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, transfer)
}
