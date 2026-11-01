package agentrunhandler

import (
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Transcripts          agentruntimeports.AgentRunTranscriptService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	transcripts agentruntimeports.AgentRunTranscriptService
	eh          *helpers.ErrorHandler
	pm          *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		transcripts: p.Transcripts,
		eh:          p.ErrorHandler,
		pm:          p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/agent-runs")
	api.GET(
		"/:runID/transcript/",
		h.pm.RequirePermission(permission.ResourceAgentRun.String(), permission.OpRead),
		h.downloadTranscript,
	)
}

func runRequest(c *gin.Context) (agentruntimeports.GetAgentRunByIDRequest, error) {
	runID, err := pulid.Parse(c.Param("runID"))
	if err != nil {
		return agentruntimeports.GetAgentRunByIDRequest{}, err
	}

	authCtx := authctx.GetAuthContext(c)

	return agentruntimeports.GetAgentRunByIDRequest{
		ID: runID,
		TenantInfo: &pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
		Reader: agentruntimeports.PersonRunRead(authCtx.UserID),
	}, nil
}

func (h *Handler) downloadTranscript(c *gin.Context) {
	req, err := runRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	transcript, err := h.transcripts.RunTranscript(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", transcript.FileName))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(transcript.Body))
}
