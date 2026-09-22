package assistanthandler

import (
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
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
	Turns                *assistantturnservice.Service
	Workflows            serviceports.WorkflowStarter
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	Logger               *zap.Logger
}

type Handler struct {
	service   serviceports.AssistantService
	turns     *assistantturnservice.Service
	workflows serviceports.WorkflowStarter
	eh        *helpers.ErrorHandler
	pm        *middleware.PermissionMiddleware
	logger    *zap.Logger
}

func New(p Params) *Handler {
	return &Handler{
		service:   p.Service,
		turns:     p.Turns,
		workflows: p.Workflows,
		eh:        p.ErrorHandler,
		pm:        p.PermissionMiddleware,
		logger:    p.Logger.Named("assistanthandler"),
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
	// A quick question makes a thread of its own, so it needs what starting
	// one needs.
	api.POST("/ask/", h.pm.RequirePermission(resource, permission.OpCreate), h.ask)
	api.GET("/threads/:threadID/", h.pm.RequirePermission(resource, permission.OpRead), h.getThread)
	// Renaming, pinning and deleting a conversation need no more than being
	// allowed to use the assistant.
	//
	// Every route here is read under the caller's own user id, so someone
	// else's conversation is not found rather than refused — ownership is the
	// authorization, and it cannot be got around. Asking for assistant:update
	// on top of that gated a person's own desk behind an organization-wide
	// grant most people have no reason to hold, so they could hold a
	// conversation and not name it. There is nothing here but their own
	// workspace, and arranging it is not an act over other people's records.
	api.PATCH(
		"/threads/:threadID/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.updateThread,
	)
	api.DELETE(
		"/threads/:threadID/",
		h.pm.RequirePermission(resource, permission.OpRead),
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
	// A reply is watched through the turn producing it rather than through the
	// request that asked for one. That is what lets a reader who closed the tab
	// come back to a reply still being written: the events are somewhere other
	// than in the connection that was lost.
	api.GET(
		"/threads/:threadID/turns/active/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.activeTurn,
	)
	api.GET(
		"/turns/:turnID/stream/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.streamTurn,
	)
	// Asking a question durably: the worker answers it, this returns the turn
	// to watch. Creating a turn is creating a message, so it is gated the same
	// way as sending one.
	api.POST(
		"/threads/:threadID/turns/",
		h.pm.RequirePermission(resource, permission.OpCreate),
		h.startTurn,
	)
	// Stopping a reply is arranging one's own conversation, like naming or
	// deleting it, and every turn here is read under the caller's own user id.
	api.POST(
		"/turns/:turnID/stop/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.stopTurn,
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
	api.GET(
		"/threads/:threadID/plans/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listThreadPlans,
	)
	// Artifacts are part of what the conversation produced, read with it;
	// pinning one is the reader arranging their own pane, which is the same
	// kind of act as naming the conversation and gated the same way.
	api.GET(
		"/threads/:threadID/artifacts/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listThreadArtifacts,
	)
	api.POST(
		"/threads/:threadID/artifacts/:artifactID/pin/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.pinArtifact,
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
	// Origin is where the conversation begins; empty means the panel.
	Origin conversation.ThreadOrigin `json:"origin"`
	// SubjectType and SubjectID name the record the conversation is about,
	// when it was opened from one.
	SubjectType agent.SubjectType `json:"subjectType"`
	SubjectID   *pulid.ID         `json:"subjectId"`
}

func (r *startThreadRequest) subjectID() pulid.ID {
	if r.SubjectID == nil {
		return pulid.Nil
	}

	return *r.SubjectID
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
		Origin:            body.Origin,
		SubjectType:       body.SubjectType,
		SubjectID:         body.subjectID(),
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

type updateThreadRequest struct {
	Title  *string `json:"title"`
	Pinned *bool   `json:"pinned"`
	// Keep lists a quick question as a conversation.
	Keep bool `json:"keep"`
}

func (h *Handler) updateThread(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body updateThreadRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authctx.GetAuthContext(c))
	thread, err := h.service.UpdateThread(c.Request.Context(), &serviceports.UpdateThreadRequest{
		ThreadID:   req.ID,
		TenantInfo: req.TenantInfo,
		Title:      body.Title,
		Pinned:     body.Pinned,
		Keep:       body.Keep,
	}, &actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, thread)
}

func (h *Handler) listThreadArtifacts(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	artifacts, err := h.service.ListThreadArtifacts(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": artifacts})
}

type pinArtifactRequest struct {
	Pinned bool `json:"pinned"`
}

func (h *Handler) pinArtifact(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	artifactID, err := pulid.Parse(c.Param("artifactID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body pinArtifactRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	artifact, err := h.service.PinArtifact(c.Request.Context(), req, artifactID, body.Pinned)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, artifact)
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

func (h *Handler) listThreadPlans(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	plans, err := h.service.ListThreadPlans(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": plans})
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
	// View is the table the page was showing, when it was one.
	View *agent.PageView `json:"view"`
}

func (r *pageContextRequest) page() *agent.PageContext {
	if r == nil {
		return nil
	}

	return &agent.PageContext{
		Path:       r.Path,
		EntityType: r.EntityType,
		EntityID:   r.EntityID,
		Title:      r.Title,
		View:       r.View,
	}
}

type sendMessageRequest struct {
	Content string              `json:"content"`
	Context *pageContextRequest `json:"context"`
	// ProviderID is the model the person picked in the composer. Empty leaves
	// the choice to the organization's priority order. It is resolved against
	// the providers this organization has assigned to the assistant before it
	// is used or stored, so an unknown id is dropped rather than trusted.
	ProviderID *pulid.ID `json:"providerId"`
	// AttachmentDocumentIDs are the files uploaded for this message, and
	// Mentions the records named from the composer.
	AttachmentDocumentIDs []pulid.ID        `json:"attachmentDocumentIds"`
	Mentions              []agent.EntityRef `json:"mentions"`
}

// askRequest is a quick question from anywhere: the words, the page, and the
// records named. There is no thread yet; the answer makes one.
type askRequest struct {
	Content  string              `json:"content"`
	Context  *pageContextRequest `json:"context"`
	Mentions []agent.EntityRef   `json:"mentions"`
}

func (r *sendMessageRequest) provider() (pulid.ID, bool) {
	if r.ProviderID == nil {
		return pulid.Nil, false
	}

	return *r.ProviderID, true
}

func (r *sendMessageRequest) page() *agent.PageContext {
	return r.Context.page()
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
	providerID, providerChosen := body.provider()
	result, err := h.service.SendMessage(c.Request.Context(), &serviceports.SendMessageRequest{
		ThreadID:              threadID,
		Content:               body.Content,
		Page:                  body.page(),
		TenantInfo:            tenantFromAuthContext(authCtx),
		PreferredProviderID:   providerID,
		ProviderChosen:        providerChosen,
		AttachmentDocumentIDs: body.AttachmentDocumentIDs,
		Mentions:              body.Mentions,
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

	emit := func(event serviceports.StreamEvent) {
		if emitErr := stream.Emit(event.Event, event.Data); emitErr != nil {
			h.logger.Error("assistant stream event lost",
				zap.String("event", event.Event),
				zap.Error(emitErr),
			)
		}
	}

	// The turn is recorded before it runs, and everything it says is published
	// under that id as well as written here. The connection stays the fast
	// path; the stream is what a reader who lost it can come back to.
	turn, err := h.turns.Start(c.Request.Context(), assistantturnservice.StartRequest{
		ThreadID:   threadID,
		UserID:     authCtx.UserID,
		TenantInfo: tenantFromAuthContext(authCtx),
	})
	if err != nil {
		emit(serviceports.StreamEvent{
			Event: serviceports.AssistantEventError,
			Data:  gin.H{"message": h.streamErrorMessage(err)},
		})
		return
	}
	emit(serviceports.StreamEvent{
		Event: serviceports.AssistantEventTurn,
		Data:  serviceports.AssistantTurnEvent{TurnID: turn.ID, ThreadID: threadID},
	})

	observed, closeStream := h.turns.Observe(c.Request.Context(), turn, emit)

	actor := requestActorFromAuthContext(authCtx)
	providerID, providerChosen := body.provider()
	result, err := h.service.SendMessageStream(c.Request.Context(), &serviceports.SendMessageRequest{
		ThreadID:              threadID,
		Content:               body.Content,
		Page:                  body.page(),
		TenantInfo:            tenantFromAuthContext(authCtx),
		PreferredProviderID:   providerID,
		ProviderChosen:        providerChosen,
		AttachmentDocumentIDs: body.AttachmentDocumentIDs,
		Mentions:              body.Mentions,
	}, &actor, observed)
	if err != nil {
		ending := serviceports.StreamEvent{
			Event: serviceports.AssistantEventError,
			Data:  gin.H{"message": h.streamErrorMessage(err)},
		}
		emit(ending)
		closeStream(ending)
		h.turns.Complete(c.Request.Context(), turn, assistantturnservice.StatusFor(false, err), err)

		return
	}

	ending := serviceports.StreamEvent{Event: serviceports.AssistantEventDone, Data: result}
	emit(ending)
	closeStream(ending)
	h.turns.Complete(
		c.Request.Context(), turn, assistantturnservice.StatusFor(result.Refused, nil), nil,
	)
}

// ask streams a quick question's answer. The thread it runs on is announced
// first, so the reader can open it in the Desk even if the answer fails; the
// rest of the stream is the same as a message on a thread.
func (h *Handler) ask(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var body askRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	stream, err := helpers.OpenEventStream(c, helpers.EventStreamOptions{})
	if err != nil {
		h.eh.HandleError(c, errortypes.NewBusinessError("Streaming is not supported"))
		return
	}
	defer stream.Close()

	emit := func(event serviceports.StreamEvent) {
		if emitErr := stream.Emit(event.Event, event.Data); emitErr != nil {
			h.logger.Error("assistant ask event lost",
				zap.String("event", event.Event),
				zap.Error(emitErr),
			)
		}
	}

	actor := requestActorFromAuthContext(authCtx)
	result, err := h.service.Ask(c.Request.Context(), &serviceports.AskRequest{
		Content:    body.Content,
		Page:       body.Context.page(),
		Mentions:   body.Mentions,
		TenantInfo: tenantFromAuthContext(authCtx),
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
