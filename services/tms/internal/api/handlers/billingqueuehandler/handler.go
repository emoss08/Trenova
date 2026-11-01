package billingqueuehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Packets              billingqueueports.PacketPreviewer `optional:"true"`
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	packets billingqueueports.PacketPreviewer
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		packets: p.Packets,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/billing-queue")
	api.GET(
		"/:itemID/packet-preview/",
		h.pm.RequirePermission(
			permission.ResourceBillingQueue.String(),
			permission.OpRead,
		),
		h.packetPreview,
	)
}

// @Summary Preview an item's invoice packet
// @Description The one PDF approving the item would send: the invoice it would make, followed by the documents it carries. Nothing is saved.
// @ID previewBillingQueuePacket
// @Tags Billing Queue
// @Produce application/pdf
// @Param itemID path string true "Billing queue item ID"
// @Success 200 {file} binary
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /billing-queue/{itemID}/packet-preview/ [get]
func (h *Handler) packetPreview(c *gin.Context) {
	if h.packets == nil {
		h.eh.HandleError(
			c,
			errortypes.NewBusinessError("Invoice packets are not configured on this deployment"),
		)
		return
	}

	authCtx := authctx.GetAuthContext(c)
	itemID, err := pulid.MustParse(c.Param("itemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	preview, err := h.packets.PreviewPacket(
		c.Request.Context(),
		&billingqueueports.PacketPreviewRequest{
			ItemID:     itemID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Content-Disposition", `inline; filename="`+preview.FileName+`"`)
	c.Data(http.StatusOK, "application/pdf", preview.Content)
}
