package agentanalysishandler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/ports/analysisports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const maxRunsPerPage = 200

type Params struct {
	fx.In

	Runs                 analysisports.Runs
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

// Handler serves the analyses agents' scripts showed people: the history for
// those who oversee agents, and one analysis whole for the person it ran for.
type Handler struct {
	runs analysisports.Runs
	eh   *helpers.ErrorHandler
	pm   *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{runs: p.Runs, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	oversees := permission.ResourceAgentRun.String()
	analyses := rg.Group("/agent-analysis")
	analyses.GET("/runs/", h.pm.RequirePermission(oversees, permission.OpRead), h.listRuns)
	analyses.GET(
		"/runs/:runID/",
		h.pm.RequireAnyPermission(
			middleware.PermissionCheck{Resource: oversees, Operation: permission.OpRead},
			middleware.PermissionCheck{
				Resource:  permission.ResourceCodeMode.String(),
				Operation: permission.OpRead,
			},
		),
		h.run,
	)
}

func (h *Handler) listRuns(c *gin.Context) {
	req := &analysisports.ListRunsRequest{TenantInfo: tenantOf(c)}
	if raw := strings.TrimSpace(c.Query("before")); raw != "" {
		before, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || before < 0 {
			h.eh.HandleError(c, errortypes.NewValidationError("before", errortypes.ErrInvalid,
				"before must be a Unix time in seconds"))
			return
		}
		req.Before = before
	}
	if raw := strings.TrimSpace(c.Query("beforeId")); raw != "" {
		beforeID, err := pulid.MustParse(raw)
		if err != nil {
			h.eh.HandleError(c, errortypes.NewValidationError("beforeId", errortypes.ErrInvalid,
				"beforeId must be the id of the last analysis on the page before"))
			return
		}
		req.BeforeID = beforeID
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

	runs, err := h.runs.ListRuns(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, runs)
}

// run is one analysis, whole, for the person it ran for or someone who
// oversees agents.
func (h *Handler) run(c *gin.Context) {
	runID, err := pulid.MustParse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError("runId", errortypes.ErrInvalid,
			"The analysis id is not valid"))
		return
	}

	oversees, err := h.pm.Allowed(c, permission.ResourceAgentRun.String(), permission.OpRead)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	run, err := h.runs.Run(c.Request.Context(), &analysisports.RunRequest{
		TenantInfo: tenantOf(c),
		RunID:      runID,
		Oversees:   oversees,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, run)
}

func tenantOf(c *gin.Context) pagination.TenantInfo {
	authCtx := authctx.GetAuthContext(c)

	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
