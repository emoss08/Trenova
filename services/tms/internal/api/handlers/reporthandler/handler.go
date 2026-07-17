package reporthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
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
