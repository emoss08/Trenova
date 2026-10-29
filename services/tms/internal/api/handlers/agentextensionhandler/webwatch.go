package agentextensionhandler

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

const (
	maxWebWatchesPerPage  = 100
	maxWatchDeliveryBytes = 1 << 20
)

// RegisterPublicRoutes takes vendors' deliveries for web watches. They carry
// no session: each is checked against its watch's secret before anything is
// done, and a valid one only asks the watch to be checked now.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST(
		extensionwebwatch.WebWatchWebhookPrefix+"/:orgID/:buID/:watchID/",
		h.receiveWatchDelivery,
	)
}

func (h *Handler) listWebWatches(c *gin.Context) {
	req := &agentextensionports.ListWebWatchesRequest{TenantInfo: tenantOf(c)}
	if raw := strings.TrimSpace(c.Query("status")); raw != "" {
		status := extensionwebwatch.WebWatchStatus(raw)
		if !status.IsValid() {
			h.eh.HandleError(c, errortypes.NewValidationError("status", errortypes.ErrInvalid,
				"status must be Active, Ended, Stopped or Failed"))
			return
		}
		req.Status = status
	}
	if raw := strings.TrimSpace(c.Query("before")); raw != "" {
		before, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || before < 0 {
			h.eh.HandleError(c, errortypes.NewValidationError("before", errortypes.ErrInvalid,
				"before must be a Unix time in seconds"))
			return
		}
		req.Before = before
	}
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > maxWebWatchesPerPage {
			h.eh.HandleError(c, errortypes.NewValidationError("limit", errortypes.ErrInvalid,
				"limit must be a whole number from 1 to 100"))
			return
		}
		req.Limit = limit
	}

	watches, err := h.service.ListWebWatches(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, watches)
}

func (h *Handler) watchIDOf(c *gin.Context) (pulid.ID, bool) {
	id, err := pulid.MustParse(c.Param("watchID"))
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError("watchId", errortypes.ErrInvalid,
			"The web watch id is not valid"))
		return pulid.Nil, false
	}

	return id, true
}

func (h *Handler) webWatchFindings(c *gin.Context) {
	id, ok := h.watchIDOf(c)
	if !ok {
		return
	}

	findings, err := h.service.WebWatchFindings(c.Request.Context(), tenantOf(c), id)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, findings)
}

// stopWebWatch stops a watch for the person it was set for, or for anyone
// who administers agents, and closes the wait the work had on it.
func (h *Handler) stopWebWatch(c *gin.Context) {
	id, ok := h.watchIDOf(c)
	if !ok {
		return
	}

	administers, err := h.pm.Allowed(
		c,
		permission.ResourceAgentExtension.String(),
		permission.OpUpdate,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	tenant := tenantOf(c)
	watch, err := h.service.StopWebWatch(
		c.Request.Context(),
		&agentextensionports.StopWebWatchRequest{
			TenantInfo:  tenant,
			WatchID:     id,
			StoppedBy:   tenant.UserID,
			Administers: administers,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if h.waits != nil {
		h.waits.EndWebWatchWaits(c.Request.Context(), tenant, &extensionwebwatch.WebWatch{
			ID:     watch.ID,
			Status: watch.Status,
		})
	}

	helpers.JSON(c, http.StatusOK, watch)
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

	err = h.service.ReceiveWebWatchDelivery(
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
