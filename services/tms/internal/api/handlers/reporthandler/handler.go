package reporthandler

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	reportingservice "github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *reportingservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *reportingservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/reports")
	api.GET(
		"/runs/:runID/download/",
		h.pm.RequirePermission(permission.ResourceReport.String(), permission.OpExport),
		h.downloadRun,
	)
	// A dashboard export is streamed back directly rather than presigned: it
	// is generated per request from the caller's filter choices, so there is
	// no stored artifact to point at.
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
	Filters []report.DashboardFilter `json:"filters"`
	Values  map[string]any           `json:"values"`
}

// exportDashboard runs every tile with the caller's own permissions and
// returns one workbook. Tile reports are re-authorized during resolution, so a
// shared dashboard never widens what the caller may read.
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

	export, err := h.service.ExportDashboard(
		c.Request.Context(),
		&reportingservice.ExportDashboardRequest{
			Request: reportingservice.Request{
				TenantInfo: pagination.TenantInfo{
					OrgID:  authCtx.OrganizationID,
					BuID:   authCtx.BusinessUnitID,
					UserID: authCtx.UserID,
				},
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
) map[string]reportingservice.TileFilterOverride {
	if len(body) == 0 {
		return nil
	}
	overrides := make(map[string]reportingservice.TileFilterOverride, len(body))
	for tileID, override := range body {
		overrides[tileID] = reportingservice.TileFilterOverride{
			Filters: override.Filters,
			Values:  override.Values,
		}
	}
	return overrides
}

// downloadRun re-authorizes at download time, audits the access, and redirects
// to a short-lived presigned URL with a forced attachment disposition.
// Presigned URLs are never stored or returned through any other surface.
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
				TenantInfo: pagination.TenantInfo{
					OrgID:  authCtx.OrganizationID,
					BuID:   authCtx.BusinessUnitID,
					UserID: authCtx.UserID,
				},
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
