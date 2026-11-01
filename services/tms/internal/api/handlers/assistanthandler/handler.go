package assistanthandler

import (
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/sessionwatch"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantturnservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Artifacts            assistantports.AssistantArtifactService
	Transcripts          assistantports.AssistantTranscriptService
	Turns                *assistantturnservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	Sessions             *sessionwatch.Watch
	Logger               *zap.Logger
}

type Handler struct {
	artifacts   assistantports.AssistantArtifactService
	transcripts assistantports.AssistantTranscriptService
	turns       *assistantturnservice.Service
	eh          *helpers.ErrorHandler
	pm          *middleware.PermissionMiddleware
	sessions    *sessionwatch.Watch
	logger      *zap.Logger
}

func New(p Params) *Handler {
	return &Handler{
		artifacts:   p.Artifacts,
		transcripts: p.Transcripts,
		turns:       p.Turns,
		eh:          p.ErrorHandler,
		pm:          p.PermissionMiddleware,
		sessions:    p.Sessions,
		logger:      p.Logger.Named("assistanthandler"),
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/assistant")
	read := h.pm.RequirePermission(permission.ResourceAssistant.String(), permission.OpRead)

	api.GET("/threads/:threadID/transcript/", read, h.downloadTranscript)
	api.GET("/turns/:turnID/stream/", read, h.streamTurn)
	api.GET("/threads/:threadID/artifacts/:artifactID/export.csv", read, h.exportArtifactCSV)
	api.GET("/threads/:threadID/artifacts/:artifactID/export/", read, h.exportDocument)
}

func tenantFromAuthContext(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

func threadRequest(c *gin.Context) (conversationports.GetThreadRequest, error) {
	authCtx := authctx.GetAuthContext(c)

	threadID, err := pulid.Parse(c.Param("threadID"))
	if err != nil {
		return conversationports.GetThreadRequest{}, err
	}

	return conversationports.GetThreadRequest{
		ID:         threadID,
		UserID:     authCtx.UserID,
		TenantInfo: tenantFromAuthContext(authCtx),
	}, nil
}

func (h *Handler) downloadTranscript(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	transcript, err := h.transcripts.Transcript(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", transcript.FileName))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(transcript.Body))
}
