package agentrunhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              serviceports.AgentRunService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service serviceports.AgentRunService
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

func requestActorFromAuthContext(authCtx *authctx.AuthContext) serviceports.RequestActor {
	return serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalType(authCtx.PrincipalType),
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
}

type startRunRequest struct {
	SubjectID pulid.ID `json:"subjectId"`
}

func (h *Handler) start(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var body startRunRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	run, err := h.service.Start(
		c.Request.Context(),
		&serviceports.StartAgentRunRequest{
			AgentType:   agent.TypeBillingException,
			SubjectType: agent.SubjectBillingQueueItem,
			SubjectID:   body.SubjectID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
		},
		&actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, run)
}

func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	runID, err := pulid.Parse(c.Param("runID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}

	run, err := h.service.GetByID(c.Request.Context(), repositories.GetAgentRunByIDRequest{
		ID:         runID,
		TenantInfo: &tenantInfo,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, run)
}
