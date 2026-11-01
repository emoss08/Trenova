package editransferhandler

import (
	"context"
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferchangeservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerTenderChangeRoutes(changes *gin.RouterGroup) {
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
