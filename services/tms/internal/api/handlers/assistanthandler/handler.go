package assistanthandler

import (
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/internal/core/domain/agent"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Service              serviceports.AssistantService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	Logger               *zap.Logger
}

type Handler struct {
	service serviceports.AssistantService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
	logger  *zap.Logger
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
		logger:  p.Logger.Named("assistanthandler"),
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/assistant")
	resource := permission.ResourceAssistant.String()

	// Anyone who may use the assistant may see which models they can pick; the
	// response is a projection, so this does not widen access to the provider
	// records themselves.
	api.GET("/providers/", h.pm.RequirePermission(resource, permission.OpRead), h.listProviders)
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
	api.GET(
		"/threads/:threadID/transcript/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.downloadTranscript,
	)
	api.POST(
		"/threads/:threadID/messages/",
		h.pm.RequirePermission(resource, permission.OpCreate),
		h.sendMessage,
	)
	api.POST(
		"/threads/:threadID/messages/stream/",
		h.pm.RequirePermission(resource, permission.OpCreate),
		h.sendMessageStream,
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

type listMessagesQuery struct {
	Limit int `form:"limit"`
	// Before is the sequence of the oldest message the client already has.
	Before *int `form:"before"`
}

func (h *Handler) listMessages(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var query listMessagesQuery
	if err = c.ShouldBindQuery(&query); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	page, err := h.service.ListMessages(c.Request.Context(), serviceports.ListThreadMessagesRequest{
		Thread:         req,
		Limit:          query.Limit,
		BeforeSequence: query.Before,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, page)
}

// downloadTranscript hands the conversation over as a Markdown file. It is a
// download rather than a JSON body because what it is for is reading away
// from the panel: a text file opens anywhere and pastes anywhere.
func (h *Handler) downloadTranscript(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	transcript, err := h.service.Transcript(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", transcript.FileName))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(transcript.Body))
}

type pageContextRequest struct {
	Path       string `json:"path"`
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
	Title      string `json:"title"`
}

type sendMessageRequest struct {
	Content string              `json:"content"`
	Context *pageContextRequest `json:"context"`
	// ProviderID is the model the person picked in the composer. Empty leaves
	// the choice to the organization's priority order. It is resolved against
	// the providers this organization has assigned to the assistant before it
	// is used or stored, so an unknown id is dropped rather than trusted.
	ProviderID pulid.ID `json:"providerId"`
}

func (r *sendMessageRequest) page() *agent.PageContext {
	if r.Context == nil {
		return nil
	}

	return &agent.PageContext{
		Path:       r.Context.Path,
		EntityType: r.Context.EntityType,
		EntityID:   r.Context.EntityID,
		Title:      r.Context.Title,
	}
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
		ThreadID:            threadID,
		Content:             body.Content,
		Page:                body.page(),
		TenantInfo:          tenantFromAuthContext(authCtx),
		PreferredProviderID: body.ProviderID,
	}, &actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	// A refusal is a successful request with a declined answer, not an error: the
	// turn was processed, recorded, and explained.
	c.JSON(http.StatusOK, result)
}

// sendMessageStream runs the same turn as sendMessage but reports it as
// server-sent events while it happens: the guard's decision, each piece of the
// reply, each tool as it starts and finishes, and finally the saved result. A
// failure after the stream has opened is reported as an error event, since the
// status line has already been sent.
func (h *Handler) sendMessageStream(c *gin.Context) {
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

	stream, err := helpers.OpenEventStream(c, helpers.EventStreamOptions{})
	if err != nil {
		h.eh.HandleError(c, errortypes.NewBusinessError("Streaming is not supported"))
		return
	}
	defer stream.Close()

	emit := func(event serviceports.StreamEvent) { stream.Emit(event.Event, event.Data) }

	actor := requestActorFromAuthContext(authCtx)
	result, err := h.service.SendMessageStream(c.Request.Context(), &serviceports.SendMessageRequest{
		ThreadID:            threadID,
		Content:             body.Content,
		Page:                body.page(),
		TenantInfo:          tenantFromAuthContext(authCtx),
		PreferredProviderID: body.ProviderID,
	}, &actor, emit)
	if err != nil {
		emit(serviceports.StreamEvent{
			Event: "error",
			Data:  gin.H{"message": h.streamErrorMessage(err)},
		})
		return
	}

	emit(serviceports.StreamEvent{Event: serviceports.AssistantEventDone, Data: result})
}

// streamErrorMessage picks what the reader may see. A business or validation
// error is written for them; anything else is an internal fault whose wording
// belongs in the log, not on their screen.
func (h *Handler) streamErrorMessage(err error) string {
	if errortypes.IsBusinessError(err) || errortypes.IsMultiError(err) {
		return err.Error()
	}

	h.logger.Error("assistant stream failed", zap.Error(err))

	return "The assistant could not finish this reply. Try again in a moment."
}

func (h *Handler) listProviders(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	actor := requestActorFromAuthContext(authCtx)

	options, err := h.service.SelectableProviders(c.Request.Context(), actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": options})
}
