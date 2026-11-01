package agentextensionhandler

import (
	"io"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

const maxWatchDeliveryBytes = 1 << 20

// RegisterPublicRoutes takes vendors' deliveries for web watches. They carry
// no session: each is checked against its watch's secret before anything is
// done, and a valid one only asks the watch to be checked now.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST(
		extensionwebwatch.WebWatchWebhookPrefix+"/:orgID/:buID/:watchID/",
		h.receiveWatchDelivery,
	)
}

func (h *Handler) receiveWatchDelivery(c *gin.Context) {
	orgID, orgErr := pulid.MustParse(c.Param("orgID"))
	buID, buErr := pulid.MustParse(c.Param("buID"))
	watchID, watchErr := pulid.MustParse(c.Param("watchID"))
	if orgErr != nil || buErr != nil || watchErr != nil {
		c.Status(http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWatchDeliveryBytes+1))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if len(body) > maxWatchDeliveryBytes {
		c.Status(http.StatusRequestEntityTooLarge)
		return
	}

	err = h.webWatches.ReceiveWebWatchDelivery(
		c.Request.Context(),
		&agentextensionports.ReceiveWebWatchDeliveryRequest{
			TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
			WatchID:    watchID,
			Delivery: agentextensionports.WebWatchDelivery{
				Header: c.Request.Header.Clone(),
				Body:   body,
				At:     time.Now(),
			},
		},
	)
	switch {
	case err == nil:
		c.Status(http.StatusOK)
	case errortypes.IsAuthenticationError(err):
		c.Status(http.StatusUnauthorized)
	case errortypes.IsNotFoundError(err):
		c.Status(http.StatusNotFound)
	default:
		h.eh.HandleError(c, err)
	}
}
