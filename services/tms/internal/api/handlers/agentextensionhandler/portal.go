package agentextensionhandler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

const maxRunsPerPage = 200

func (h *Handler) listSites(c *gin.Context) {
	sites, err := h.service.ListSites(c.Request.Context(), tenantOf(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": sites})
}

func (h *Handler) createSite(c *gin.Context) {
	req := new(agentextensionports.SavePortalSiteRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(c)

	site, err := h.service.CreateSite(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, site)
}

func (h *Handler) updateSite(c *gin.Context) {
	id, err := pulid.MustParse(c.Param("siteID"))
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError("siteId", errortypes.ErrInvalid,
			"The site id is not valid"))
		return
	}
	req := new(agentextensionports.SavePortalSiteRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(c)
	req.ID = id

	site, err := h.service.UpdateSite(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, site)
}

func (h *Handler) deleteSite(c *gin.Context) {
	id, err := pulid.MustParse(c.Param("siteID"))
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError("siteId", errortypes.ErrInvalid,
			"The site id is not valid"))
		return
	}

	if err = h.service.DeleteSite(c.Request.Context(), tenantOf(c), id); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) listRuns(c *gin.Context) {
	req := &agentextensionports.ListPortalRunsRequest{TenantInfo: tenantOf(c)}
	if raw := strings.TrimSpace(c.Query("siteId")); raw != "" {
		id, err := pulid.MustParse(raw)
		if err != nil {
			h.eh.HandleError(c, errortypes.NewValidationError("siteId", errortypes.ErrInvalid,
				"The site id is not valid"))
			return
		}
		req.SiteID = id
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
		if err != nil || limit < 1 || limit > maxRunsPerPage {
			h.eh.HandleError(c, errortypes.NewValidationError("limit", errortypes.ErrInvalid,
				"limit must be a whole number from 1 to 200"))
			return
		}
		req.Limit = limit
	}

	runs, err := h.service.ListRuns(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, runs)
}

func (h *Handler) runRecording(c *gin.Context) {
	id, err := pulid.MustParse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError("runId", errortypes.ErrInvalid,
			"The portal run id is not valid"))
		return
	}

	recording, err := h.service.RunRecording(c.Request.Context(), tenantOf(c), id)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, recording)
}

// portalRun is one run, for the person it was asked for or someone who
// administers agents: the Desk's card follows it while it runs.
func (h *Handler) portalRun(c *gin.Context) {
	req, ok := h.runAccess(c, permission.OpRead)
	if !ok {
		return
	}

	run, err := h.service.PortalRunFor(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, run)
}

// portalRunLive is the address of the open browser a run is in, where the
// person watches it or signs in themselves. Taking over another person's
// browser needs the right to change agent extensions.
func (h *Handler) portalRunLive(c *gin.Context) {
	req, ok := h.runAccess(c, permission.OpUpdate)
	if !ok {
		return
	}

	live, err := h.service.PortalRunLive(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, live)
}

func (h *Handler) runAccess(
	c *gin.Context,
	administer permission.Operation,
) (*agentextensionports.PortalRunAccessRequest, bool) {
	id, err := pulid.MustParse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError("runId", errortypes.ErrInvalid,
			"The portal run id is not valid"))
		return nil, false
	}
	administers, err := h.pm.Allowed(c, permission.ResourceAgentExtension.String(), administer)
	if err != nil {
		h.eh.HandleError(c, err)
		return nil, false
	}

	return &agentextensionports.PortalRunAccessRequest{
		TenantInfo:  tenantOf(c),
		RunID:       id,
		Administers: administers,
	}, true
}

type signInCodeRequest struct {
	Code     string `json:"code"`
	SignedIn bool   `json:"signedIn"`
}

// provideSignInCode continues a portal task waiting on a sign-in code with the
// code the person was sent. The person the task was asked for enters it from
// their conversation; someone who administers agents can enter it for any.
func (h *Handler) provideSignInCode(c *gin.Context) {
	id, err := pulid.MustParse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError("runId", errortypes.ErrInvalid,
			"The portal run id is not valid"))
		return
	}
	body := new(signInCodeRequest)
	if err = c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
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

	run, err := h.service.ProvidePortalCode(
		c.Request.Context(),
		&agentextensionports.ProvidePortalCodeRequest{
			TenantInfo:  tenantOf(c),
			RunID:       id,
			Code:        body.Code,
			SignedIn:    body.SignedIn,
			Administers: administers,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, run)
}
