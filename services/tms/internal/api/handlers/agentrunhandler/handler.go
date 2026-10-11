package agentrunhandler

import (
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              agentports.AgentRunService
	Transcripts          agentports.AgentRunTranscriptService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service     agentports.AgentRunService
	transcripts agentports.AgentRunTranscriptService
	eh          *helpers.ErrorHandler
	pm          *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service:     p.Service,
		transcripts: p.Transcripts,
		eh:          p.ErrorHandler,
		pm:          p.PermissionMiddleware,
	}
}

func requestActorFromAuthContext(authCtx *authctx.AuthContext) permissionports.RequestActor {
	return permissionports.RequestActor{
		PrincipalType:  kernelports.PrincipalType(authCtx.PrincipalType),
		PrincipalID:    authCtx.PrincipalID,
		UserID:         authCtx.UserID,
		APIKeyID:       authCtx.APIKeyID,
		BusinessUnitID: authCtx.BusinessUnitID,
		OrganizationID: authCtx.OrganizationID,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/agent-runs")
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceAgentRun.String(), permission.OpCreate),
		h.start,
	)
	api.GET(
		"/:runID/",
		h.pm.RequirePermission(permission.ResourceAgentRun.String(), permission.OpRead),
		h.get,
	)
	api.GET(
		"/:runID/transcript/",
		h.pm.RequirePermission(permission.ResourceAgentRun.String(), permission.OpRead),
		h.downloadTranscript,
	)
}

type startRunRequest struct {
	AgentDefinitionID pulid.ID          `json:"agentDefinitionId"`
	SystemKey         string            `json:"systemKey"`
	SubjectType       agent.SubjectType `json:"subjectType"`
	SubjectID         pulid.ID          `json:"subjectId"`
}

func (h *Handler) start(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var body startRunRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	run, err := h.service.StartForDefinition(
		c.Request.Context(),
		&agentports.StartAgentRunForDefinitionRequest{
			DefinitionID: body.AgentDefinitionID,
			SystemKey:    body.SystemKey,
			SubjectType:  body.SubjectType,
			SubjectID:    body.SubjectID,
			Trigger:      agent.RunTriggerManual,
			TenantInfo:   pagination.FromAuthAsUser(authCtx),
		},
		&actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, run)
}

func runRequest(c *gin.Context) (agentports.GetAgentRunByIDRequest, error) {
	runID, err := pulid.Parse(c.Param("runID"))
	if err != nil {
		return agentports.GetAgentRunByIDRequest{}, err
	}

	authCtx := authctx.GetAuthContext(c)

	return agentports.GetAgentRunByIDRequest{
		ID: runID,
		TenantInfo: &pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	}, nil
}

func (h *Handler) get(c *gin.Context) {
	req, err := runRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	run, err := h.service.GetByID(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, run)
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
