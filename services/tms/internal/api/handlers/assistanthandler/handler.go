package assistanthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
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

	Service              serviceports.AssistantService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service serviceports.AssistantService
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
	api := rg.Group("/assistant")
	resource := permission.ResourceAssistant.String()

	api.GET("/threads/", h.pm.RequirePermission(resource, permission.OpRead), h.listThreads)
	api.POST("/threads/", h.pm.RequirePermission(resource, permission.OpCreate), h.startThread)
	api.GET("/threads/:threadID/", h.pm.RequirePermission(resource, permission.OpRead), h.getThread)
	api.DELETE(
		"/threads/:threadID/",
		h.pm.RequirePermission(resource, permission.OpDelete),
		h.deleteThread,
	)
	api.GET(
		"/threads/:threadID/messages/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listMessages,
	)
	api.POST(
		"/threads/:threadID/messages/",
		h.pm.RequirePermission(resource, permission.OpCreate),
		h.sendMessage,
	)
	// Reading a conversation's proposals needs no more than reading the
	// conversation: they are part of what was said. Acting on one goes through the
	// agent proposal endpoint, which requires permission over proposals and, at
	// execution, over whatever the tool touches.
	api.GET(
		"/threads/:threadID/proposals/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listThreadProposals,
	)
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

func tenantFromAuthContext(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

// threadRequest builds the per-user scoped read used by every thread route. The
// user comes from the auth context rather than the path, so one person cannot
// read another's conversation by guessing an id.
func threadRequest(c *gin.Context) (repositories.GetThreadRequest, error) {
	authCtx := authctx.GetAuthContext(c)

	threadID, err := pulid.Parse(c.Param("threadID"))
	if err != nil {
		return repositories.GetThreadRequest{}, err
	}

	return repositories.GetThreadRequest{
		ID:         threadID,
		UserID:     authCtx.UserID,
		TenantInfo: tenantFromAuthContext(authCtx),
	}, nil
}

type listThreadsQuery struct {
	Limit  int `form:"limit"`
	Offset int `form:"offset"`
}

func (h *Handler) listThreads(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var query listThreadsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.ListThreads(c.Request.Context(), repositories.ListThreadsRequest{
		UserID:     authCtx.UserID,
		TenantInfo: tenantFromAuthContext(authCtx),
		Limit:      query.Limit,
		Offset:     query.Offset,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

type startThreadRequest struct {
	AgentDefinitionID string `json:"agentDefinitionId"`
	Title             string `json:"title"`
}

func (h *Handler) startThread(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var body startThreadRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	agentID, err := pulid.Parse(body.AgentDefinitionID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	thread, err := h.service.StartThread(c.Request.Context(), &serviceports.StartThreadRequest{
		AgentDefinitionID: agentID,
		Title:             body.Title,
		TenantInfo:        tenantFromAuthContext(authCtx),
	}, &actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, thread)
}

func (h *Handler) getThread(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	thread, err := h.service.GetThread(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, thread)
}

func (h *Handler) deleteThread(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.DeleteThread(c.Request.Context(), req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) listThreadProposals(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	proposals, err := h.service.ListThreadProposals(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": proposals})
}

func (h *Handler) listMessages(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	messages, err := h.service.ListMessages(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": messages})
}

type sendMessageRequest struct {
	Content string `json:"content"`
}

func (h *Handler) sendMessage(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	threadID, err := pulid.Parse(c.Param("threadID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body sendMessageRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	result, err := h.service.SendMessage(c.Request.Context(), &serviceports.SendMessageRequest{
		ThreadID:   threadID,
		Content:    body.Content,
		TenantInfo: tenantFromAuthContext(authCtx),
	}, &actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	// A refusal is a successful request with a declined answer, not an error: the
	// turn was processed, recorded, and explained.
	c.JSON(http.StatusOK, result)
}
