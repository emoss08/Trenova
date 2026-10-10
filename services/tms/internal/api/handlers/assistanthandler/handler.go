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
	"github.com/emoss08/trenova/internal/core/services/agentwaitservice"
	"github.com/emoss08/trenova/internal/core/services/assistantqueueservice"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/services/conversationscheduleservice"
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
	Schedules            *conversationscheduleservice.Service
	Handoffs             serviceports.AssistantHandoffService `optional:"true"`
	Queue                *assistantqueueservice.Service
	Waits                *agentwaitservice.Service
	Cases                serviceports.AssistantCaseService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	Logger               *zap.Logger
}

type Handler struct {
	service   serviceports.AssistantService
	turns     *assistantturnservice.Service
	workflows serviceports.WorkflowStarter
	schedules scheduleService
	handoffs  serviceports.AssistantHandoffService
	queue     *assistantqueueservice.Service
	waits     *agentwaitservice.Service
	cases     serviceports.AssistantCaseService
	eh        *helpers.ErrorHandler
	pm        *middleware.PermissionMiddleware
	logger    *zap.Logger
}

func New(p Params) *Handler {
	h := &Handler{
		service:   p.Service,
		turns:     p.Turns,
		workflows: p.Workflows,
		handoffs:  p.Handoffs,
		queue:     p.Queue,
		waits:     p.Waits,
		cases:     p.Cases,
		eh:        p.ErrorHandler,
		pm:        p.PermissionMiddleware,
		logger:    p.Logger.Named("assistanthandler"),
	}
	// A nil service put in the interface would no longer compare equal to
	// nil, and the schedule routes would call through it.
	if p.Schedules != nil {
		h.schedules = p.Schedules
	}

	return h
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/assistant")
	resource := permission.ResourceAssistant.String()

	// Anyone who may use the assistant may see which models they can pick; the
	// response is a projection, so this does not widen access to the provider
	// records themselves.
	api.GET("/providers/", h.pm.RequirePermission(resource, permission.OpRead), h.listProviders)
	api.GET("/mentions/", h.pm.RequirePermission(resource, permission.OpRead), h.searchMentions)
	api.GET("/search/", h.pm.RequirePermission(resource, permission.OpRead), h.searchDesk)
	api.GET("/threads/", h.pm.RequirePermission(resource, permission.OpRead), h.listThreads)
	api.POST("/threads/", h.pm.RequirePermission(resource, permission.OpCreate), h.startThread)
	// A quick question makes a thread of its own, so it needs what starting
	// one needs.
	api.POST("/ask/", h.pm.RequirePermission(resource, permission.OpCreate), h.ask)
	api.GET("/threads/:threadID/", h.pm.RequirePermission(resource, permission.OpRead), h.getThread)
	api.GET(
		"/threads/:threadID/budget/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.threadBudget,
	)
	api.POST(
		"/threads/:threadID/requests/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.requestMore,
	)
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
	// Handing a conversation to another agent starts a conversation, so it
	// needs what starting one needs.
	api.POST(
		"/threads/:threadID/handoff/",
		h.pm.RequirePermission(resource, permission.OpCreate),
		h.handoff,
	)
	api.POST(
		"/threads/:threadID/read/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.markThreadRead,
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
	// A reply is watched through the turn producing it rather than through the
	// request that asked for one. That is what lets a reader who closed the tab
	// come back to a reply still being written: the events are somewhere other
	// than in the connection that was lost.
	api.GET(
		"/threads/:threadID/turns/active/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.activeTurn,
	)
	// Every reply the caller has in progress, across their conversations, so
	// any tab can show one another tab started. The static segment is matched
	// ahead of the turn id below, so "active" is never read as a turn.
	api.GET(
		"/turns/active/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.activeTurns,
	)
	api.GET(
		"/turns/:turnID/stream/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.streamTurn,
	)
	// Asking a question: a worker answers it, and this returns the turn to
	// watch. Creating a turn is creating a message, so it is gated the same
	// way as sending one.
	api.POST(
		"/threads/:threadID/turns/",
		h.pm.RequirePermission(resource, permission.OpCreate),
		h.startTurn,
	)
	// Compacting summarizes the older part of a conversation into a message
	// of its own, which is writing to it like asking a question.
	api.POST(
		"/threads/:threadID/compact/",
		h.pm.RequirePermission(resource, permission.OpCreate),
		h.compactThread,
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
	// Saving the wording a person changed on a pending proposal is drafting
	// in their own conversation, not deciding: the approval is still the
	// decision endpoint's, and it checks the values again.
	api.PUT(
		"/threads/:threadID/proposals/:proposalID/edits/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.saveProposalEdits,
	)
	api.GET(
		"/threads/:threadID/plans/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listThreadPlans,
	)
	// Artifacts are part of what the conversation produced, read with it;
	// pinning one is the reader arranging their own pane, which is the same
	// kind of act as naming the conversation and gated the same way.
	api.POST(
		"/threads/:threadID/artifacts/:artifactID/pin/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.pinArtifact,
	)
	h.registerScheduleRoutes(api, resource)
	h.registerArtifactRoutes(api, resource)
	h.registerQueueRoutes(api, resource)
	h.registerWaitRoutes(api, resource)
	h.registerCaseRoutes(api, resource)
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
	Limit  int    `form:"limit"`
	Cursor string `form:"cursor"`
	Until  string `form:"until"`
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
		Cursor:     query.Cursor,
		Until:      query.Until,
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

func (h *Handler) threadBudget(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	budget, err := h.service.ThreadBudget(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, budget)
}

func (h *Handler) requestMore(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	var body struct {
		Kind string `json:"kind"`
	}
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"kind", errortypes.ErrInvalid, "Say what is being asked for",
		))
		return
	}

	result, err := h.service.RequestMore(c.Request.Context(), serviceports.RequestMoreRequest{
		Thread: req,
		Kind:   body.Kind,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) markThreadRead(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.MarkThreadRead(c.Request.Context(), req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

type updateThreadRequest struct {
	Title  *string `json:"title"`
	Pinned *bool   `json:"pinned"`
	// PinnedFacts replaces the facts the agents keep in mind, when sent.
	PinnedFacts *[]string `json:"pinnedFacts"`
	// Keep lists a quick question as a conversation.
	Keep bool `json:"keep"`
	// AutoCompact turns the conversation's compacting itself on or off.
	AutoCompact *bool `json:"autoCompact"`
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
		ThreadID:    req.ID,
		TenantInfo:  req.TenantInfo,
		Title:       body.Title,
		Pinned:      body.Pinned,
		PinnedFacts: body.PinnedFacts,
		Keep:        body.Keep,
		AutoCompact: body.AutoCompact,
	}, &actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, thread)
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

// saveProposalEditsRequest is the values a person changed, keyed by
// parameter; an empty or absent set clears what was saved.
type saveProposalEditsRequest struct {
	Modifications map[string]any `json:"modifications"`
}

func (h *Handler) saveProposalEdits(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	proposalID, err := pulid.Parse(c.Param("proposalID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body saveProposalEditsRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	edits, err := h.service.SaveProposalEdits(
		c.Request.Context(),
		req,
		proposalID,
		body.Modifications,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, edits)
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
	Surface agent.Surface       `json:"surface"`
	// ProviderID is the model the person picked in the composer. Empty leaves
	// the choice to the organization's priority order. It is resolved against
	// the providers this organization has assigned to the assistant before it
	// is used or stored, so an unknown id is dropped rather than trusted.
	ProviderID *pulid.ID `json:"providerId"`
	// AttachmentDocumentIDs are the files uploaded for this message, and
	// Mentions the records named from the composer.
	AttachmentDocumentIDs []pulid.ID        `json:"attachmentDocumentIds"`
	Mentions              []agent.EntityRef `json:"mentions"`
	// FollowUpProposalID asks for the turn that follows a decision on one of
	// the thread's proposals, in place of content.
	FollowUpProposalID pulid.ID `json:"followUpProposalId"`
	// DirectedAgentID hands the message to another agent the person may use,
	// run from this conversation in place of its own agent's reply.
	DirectedAgentID pulid.ID `json:"directedAgentId"`
	// awaited is set by the route that waits for the saved turn rather than
	// following its stream. It is never read from the request body.
	awaited bool
}

// askRequest is a quick question from anywhere: the words, the page, and the
// records named. There is no thread yet; the answer makes one.
type askRequest struct {
	Content  string              `json:"content"`
	Context  *pageContextRequest `json:"context"`
	Surface  agent.Surface       `json:"surface"`
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

// searchMentionsQuery is the @ search's tab, or with a limit one page of a
// single kind for a list that scrolls through every record of it.
type searchMentionsQuery struct {
	Query  string `form:"query"`
	Type   string `form:"type"`
	Kind   string `form:"kind"`
	Offset int    `form:"offset"`
	Limit  int    `form:"limit"`
}

func (h *Handler) searchMentions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var query searchMentionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if query.Limit > 0 {
		page, err := h.service.SearchMentionPage(
			c.Request.Context(),
			requestActorFromAuthContext(authCtx),
			serviceports.MentionPageRequest{
				Query:  query.Query,
				Kind:   query.Kind,
				Offset: query.Offset,
				Limit:  query.Limit,
			},
		)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		c.JSON(http.StatusOK, page)
		return
	}

	results, err := h.service.SearchMentions(
		c.Request.Context(),
		requestActorFromAuthContext(authCtx),
		serviceports.MentionSearchRequest{Query: query.Query, Kind: query.Type},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

type searchDeskQuery struct {
	Query string `form:"query"`
	Kind  string `form:"kind"`
}

func (h *Handler) searchDesk(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var query searchDeskQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	results, err := h.service.SearchDesk(
		c.Request.Context(),
		requestActorFromAuthContext(authCtx),
		serviceports.DeskSearchRequest{Query: query.Query, Kind: query.Kind},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
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
