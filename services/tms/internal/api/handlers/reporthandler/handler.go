package reporthandler

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reportdashboardservice"
	reportingservice "github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *reportingservice.Service
	DashboardService     *reportdashboardservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service    *reportingservice.Service
	dashboards *reportdashboardservice.Service
	eh         *helpers.ErrorHandler
	pm         *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service:    p.Service,
		dashboards: p.DashboardService,
		eh:         p.ErrorHandler,
		pm:         p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/reports")
	api.GET(
		"/runs/:runID/download/",
		h.pm.RequirePermission(permission.ResourceReport.String(), permission.OpExport),
		h.downloadRun,
	)
	api.POST(
		"/dashboards/:dashboardID/export/",
		h.pm.RequirePermission(permission.ResourceReport.String(), permission.OpExport),
		h.exportDashboard,
	)
}

type exportDashboardBody struct {
	FilterValues map[string]any                  `json:"filterValues"`
	ParamValues  map[string]any                  `json:"paramValues"`
	TileFilters  map[string]exportTileFilterBody `json:"tileFilters"`
}

type exportTileFilterBody struct {
	Filters []ir.DashboardFilter `json:"filters"`
	Values  map[string]any       `json:"values"`
}

func (h *Handler) exportDashboard(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	dashboardID, err := pulid.MustParse(c.Param("dashboardID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body exportDashboardBody
	if err = c.ShouldBindJSON(&body); err != nil && !errors.Is(err, io.EOF) {
		h.eh.HandleError(c, err)
		return
	}

	export, err := h.dashboards.ExportDashboard(
		c.Request.Context(),
		&reportdashboardservice.ExportDashboardRequest{
			Request: reportingservice.Request{
				TenantInfo: pagination.FromAuthAsUser(authCtx),
			},
			DashboardID:  dashboardID,
			FilterValues: body.FilterValues,
			ParamValues:  body.ParamValues,
			TileFilters:  exportTileFilters(body.TileFilters),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", export.FileName))
	c.Data(http.StatusOK, export.ContentType, export.Body)
}

func exportTileFilters(
	body map[string]exportTileFilterBody,
) map[string]reportdashboardservice.TileFilterOverride {
	if len(body) == 0 {
		return nil
	}
	overrides := make(map[string]reportdashboardservice.TileFilterOverride, len(body))
	for tileID, override := range body {
		overrides[tileID] = reportdashboardservice.TileFilterOverride{
			Filters: override.Filters,
			Values:  override.Values,
		}
	}
	return overrides
}

func (h *Handler) downloadRun(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	runID, err := pulid.MustParse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	download, err := h.service.DownloadRun(
		c.Request.Context(),
		&reportingservice.GetRunRequest{
			Request: reportingservice.Request{
				TenantInfo: pagination.FromAuthAsUser(authCtx),
			},
			RunID: runID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Redirect(http.StatusFound, download.URL)
}
