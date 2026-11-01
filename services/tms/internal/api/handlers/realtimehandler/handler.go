package realtimehandler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/sessionwatch"
	"github.com/emoss08/trenova/internal/infrastructure/config/realtimeconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	retryAfterSeconds  = 5
	shipmentCommentsNS = "shipment-comments:"
	viewNS             = "view:"
	recordNS           = "record:"
)

type Params struct {
	fx.In

	Gateway              realtimeports.RealtimeGateway
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	RealtimeConfig       *realtimeconfig.Config
	Sessions             *sessionwatch.Watch
	Logger               *zap.Logger
}

type Handler struct {
	gateway     realtimeports.RealtimeGateway
	eh          *helpers.ErrorHandler
	pm          *middleware.PermissionMiddleware
	maxLifetime time.Duration
	sessions    *sessionwatch.Watch
	l           *zap.Logger
}

func New(p Params) *Handler {
	return &Handler{
		gateway:     p.Gateway,
		eh:          p.ErrorHandler,
		pm:          p.PermissionMiddleware,
		maxLifetime: p.RealtimeConfig.GetMaxStreamLifetime(),
		sessions:    p.Sessions,
		l:           p.Logger.Named("handler.realtime"),
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/realtime/stream/", h.stream)

	views := rg.Group("/realtime/presence/:resource")
	canRead := h.pm.RequireParamPermission("resource", permission.OpRead)
	views.POST("/", canRead, h.joinView)
	views.DELETE("/", canRead, h.leaveView)
	views.POST("/:recordID/", canRead, h.joinRecord)
	views.DELETE("/:recordID/", canRead, h.leaveRecord)
}

type presenceRequest struct {
	ConnectionID string `json:"connectionId"`
}

// @Summary Open the live event stream
// @Description Streams resource invalidations, presence, and typing events for the caller's
// @Description tenant as server-sent events. Send Last-Event-ID to resume after a disconnect.
// @ID openRealtimeStream
// @Tags Realtime
// @Produce text/event-stream
// @Param presence query string false "Set to 'users' to join the tenant's online-user presence"
// @Param Last-Event-ID header string false "The id of the last event applied"
// @Success 200 {string} string "text/event-stream"
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 429 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /realtime/stream/ [get]
func (h *Handler) stream(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if authCtx == nil || authCtx.IsAPIKey() || authCtx.UserID.IsNil() {
		h.eh.HandleError(c, errortypes.NewAuthorizationError(
			"Live updates are available to signed-in users only",
		))
		return
	}

	ctx := c.Request.Context()
	rt, err := h.gateway.Open(ctx, &realtimeports.OpenRealtimeStreamRequest{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
		Portal:         authCtx.IsPortalUser,
		JoinUsers:      c.Query("presence") == realtimeports.RealtimeScopeUsers,
		LastEventID:    helpers.EventCursor(c),
	})
	if err != nil {
		if errortypes.IsRateLimitError(err) {
			c.Header("Retry-After", strconv.Itoa(retryAfterSeconds))
		}
		h.eh.HandleError(c, err)
		return
	}
	defer rt.Close()

	stream, err := helpers.OpenEventStream(c, helpers.EventStreamOptions{Heartbeat: -1})
	if err != nil {
		h.eh.HandleError(c, errortypes.NewBusinessError("Streaming is not supported"))
		return
	}
	defer stream.Close()

	for _, frame := range rt.Prelude() {
		stream.EmitRaw(frame.ID, frame.Event, frame.Data)
	}

	// Streams are recycled on a jittered clock so a long session re-proves its
	// authentication, and so a fleet restart does not bring every reader back
	// in the same second.
	rotate := time.NewTimer(timeutils.Jittered(h.maxLifetime))
	defer rotate.Stop()

	sessionEnded, stopWatching := h.sessions.Watch(authCtx.SessionID)
	defer stopWatching()

	frames := rt.Frames()
	for {
		select {
		case <-ctx.Done():
			return
		case <-rotate.C:
			h.emitClose(stream, realtimeports.RealtimeCloseRotate)
			return
		case <-sessionEnded:
			h.emitClose(stream, realtimeports.RealtimeCloseSessionEnded)
			return
		case frame, ok := <-frames:
			if !ok {
				h.emitClose(stream, rt.Reason())
				return
			}
			stream.EmitRaw(frame.ID, frame.Event, frame.Data)
		}
	}
}

func (h *Handler) emitClose(stream *helpers.EventStream, reason string) {
	if err := stream.Emit(realtimeports.RealtimeEventClose, realtimeports.RealtimeCloseEvent{
		Reason: reason,
	}); err != nil {
		h.l.Debug("could not send realtime close notice", zap.Error(err))
	}
}

// @Summary Join a table's presence
// @Description Marks the caller's live connection as looking at a table and returns
// @Description everyone else looking at it.
// @ID joinViewPresence
// @Tags Realtime
// @Accept json
// @Produce json
// @Param resource path string true "Permission resource, such as shipment"
// @Param request body presenceRequest true "Live connection"
// @Success 200 {object} services.RealtimePresenceSnapshot
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /realtime/presence/{resource}/ [post]
func (h *Handler) joinView(c *gin.Context) {
	h.join(c, viewNS+c.Param("resource"))
}

// @Summary Leave a table's presence
// @ID leaveViewPresence
// @Tags Realtime
// @Param resource path string true "Permission resource"
// @Param connectionId query string true "Live connection ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /realtime/presence/{resource}/ [delete]
func (h *Handler) leaveView(c *gin.Context) {
	h.leave(c, viewNS+c.Param("resource"))
}

// @Summary Join a record's presence
// @Description Marks the caller's live connection as having one record open and returns
// @Description everyone else who has it open.
// @ID joinRecordPresence
// @Tags Realtime
// @Accept json
// @Produce json
// @Param resource path string true "Permission resource, such as shipment"
// @Param recordID path string true "Record ID"
// @Param request body presenceRequest true "Live connection"
// @Success 200 {object} services.RealtimePresenceSnapshot
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /realtime/presence/{resource}/{recordID}/ [post]
func (h *Handler) joinRecord(c *gin.Context) {
	scope, ok := h.recordScope(c)
	if !ok {
		return
	}
	h.join(c, scope)
}

// @Summary Leave a record's presence
// @ID leaveRecordPresence
// @Tags Realtime
// @Param resource path string true "Permission resource"
// @Param recordID path string true "Record ID"
// @Param connectionId query string true "Live connection ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /realtime/presence/{resource}/{recordID}/ [delete]
func (h *Handler) leaveRecord(c *gin.Context) {
	scope, ok := h.recordScope(c)
	if !ok {
		return
	}
	h.leave(c, scope)
}

func (h *Handler) join(c *gin.Context, scope string) {
	var body presenceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	snapshot, err := h.gateway.JoinPresence(
		c.Request.Context(),
		scopeRequest(c, body.ConnectionID, scope),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, snapshot)
}

func (h *Handler) leave(c *gin.Context, scope string) {
	if err := h.gateway.LeavePresence(
		c.Request.Context(),
		scopeRequest(c, c.Query("connectionId"), scope),
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) recordScope(c *gin.Context) (string, bool) {
	recordID := c.Param("recordID")
	if !pulid.LooksLike(recordID) {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"recordId", errortypes.ErrInvalid, "Record ID is invalid",
		))
		return "", false
	}

	return recordNS + c.Param("resource") + ":" + recordID, true
}

func scopeRequest(c *gin.Context, connectionID, scope string) *realtimeports.RealtimeScopeRequest {
	authCtx := authctx.GetAuthContext(c)

	return &realtimeports.RealtimeScopeRequest{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
		ConnectionID:   connectionID,
		Scope:          scope,
	}
}
