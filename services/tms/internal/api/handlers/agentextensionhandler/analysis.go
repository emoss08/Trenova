package agentextensionhandler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) listAnalysisRuns(c *gin.Context) {
	req := &agentextensionports.ListAnalysisRunsRequest{TenantInfo: tenantOf(c)}
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

	runs, err := h.service.ListAnalysisRuns(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, runs)
}

// analysisChart sends one of an analysis's charts as the image it is. The
// person the analysis ran for sees it beside the reply; someone who oversees
// agents sees every run's.
func (h *Handler) analysisChart(c *gin.Context) {
	runID, err := pulid.MustParse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError("runId", errortypes.ErrInvalid,
			"The analysis id is not valid"))
		return
	}
	position, err := strconv.Atoi(c.Param("position"))
	if err != nil || position < 1 || position > agentextension.MaxAnalysisCharts {
		h.eh.HandleError(c, errortypes.NewValidationError("position", errortypes.ErrInvalid,
			"The chart number is not valid"))
		return
	}

	administers, err := h.pm.Allowed(
		c,
		permission.ResourceAgentExtension.String(),
		permission.OpRead,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	chart, err := h.service.AnalysisChart(
		c.Request.Context(),
		&agentextensionports.AnalysisChartRequest{
			TenantInfo:  tenantOf(c),
			RunID:       runID,
			Position:    position,
			Administers: administers,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", `inline; filename="chart-`+strconv.Itoa(position)+`.png"`)
	c.Data(http.StatusOK, "image/png", chart.Image)
}

// analysisRun is one analysis, whole, for the person it ran for or someone
// who oversees agents.
func (h *Handler) analysisRun(c *gin.Context) {
	runID, err := pulid.MustParse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError("runId", errortypes.ErrInvalid,
			"The analysis id is not valid"))
		return
	}

	administers, err := h.pm.Allowed(
		c,
		permission.ResourceAgentExtension.String(),
		permission.OpRead,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	run, err := h.service.AnalysisRun(c.Request.Context(), &agentextensionports.AnalysisRunRequest{
		TenantInfo:  tenantOf(c),
		RunID:       runID,
		Administers: administers,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, run)
}
