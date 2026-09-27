package realtimehandler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	lastEventIDHeader  = "Last-Event-ID"
	retryAfterSeconds  = 5
	shipmentCommentsNS = "shipment-comments:"
)

type Params struct {
	fx.In

	Gateway              services.RealtimeGateway
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	Config               *config.Config
	Logger               *zap.Logger
}

type Handler struct {
	gateway     services.RealtimeGateway
	eh          *helpers.ErrorHandler
	pm          *middleware.PermissionMiddleware
	maxLifetime time.Duration
	l           *zap.Logger
}

func New(p Params) *Handler {
	return &Handler{
		gateway:     p.Gateway,
		eh:          p.ErrorHandler,
		pm:          p.PermissionMiddleware,
		maxLifetime: p.Config.GetRealtimeConfig().GetMaxStreamLifetime(),
		l:           p.Logger.Named("handler.realtime"),
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/realtime/stream/", h.stream)

	comments := rg.Group("/shipments/:shipmentID/comments")
	comments.POST(
		"/presence/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.joinShipmentComments,
	)
	comments.DELETE(
		"/presence/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.leaveShipmentComments,
	)
	comments.POST(
		"/typing/",
		h.pm.RequirePermission(permission.ResourceShipment.String(), permission.OpRead),
		h.typingShipmentComments,
	)
}

type presenceRequest struct {
	ConnectionID string `json:"connectionId"`
}

type typingRequest struct {
	ConnectionID string `json:"connectionId"`
	Stop         bool   `json:"stop"`
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
	rt, err := h.gateway.Open(ctx, &services.OpenRealtimeStreamRequest{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
		Portal:         authCtx.IsPortalUser,
		JoinUsers:      c.Query("presence") == services.RealtimeScopeUsers,
		LastEventID:    cursorFrom(c),
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

	frames := rt.Frames()
	for {
		select {
		case <-ctx.Done():
			return
		case <-rotate.C:
			h.emitClose(stream, services.RealtimeCloseRotate)
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
	if err := stream.Emit(services.RealtimeEventClose, services.RealtimeCloseEvent{
		Reason: reason,
	}); err != nil {
		h.l.Debug("could not send realtime close notice", zap.Error(err))
	}
}

// @Summary Join shipment comment presence
// @Description Marks the caller's live connection as viewing a shipment's comments and
// @Description returns everyone else viewing them.
// @ID joinShipmentCommentPresence
// @Tags Realtime
// @Accept json
// @Produce json
// @Param shipmentID path string true "Shipment ID"
// @Param request body presenceRequest true "Live connection"
// @Success 200 {object} services.RealtimePresenceSnapshot
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/comments/presence/ [post]
func (h *Handler) joinShipmentComments(c *gin.Context) {
	scope, ok := h.shipmentCommentsScope(c)
	if !ok {
		return
	}

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

	c.JSON(http.StatusOK, snapshot)
}

// @Summary Leave shipment comment presence
// @ID leaveShipmentCommentPresence
// @Tags Realtime
// @Param shipmentID path string true "Shipment ID"
// @Param connectionId query string true "Live connection ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/comments/presence/ [delete]
func (h *Handler) leaveShipmentComments(c *gin.Context) {
	scope, ok := h.shipmentCommentsScope(c)
	if !ok {
		return
	}

	if err := h.gateway.LeavePresence(
		c.Request.Context(),
		scopeRequest(c, c.Query("connectionId"), scope),
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Signal typing in shipment comments
// @Description Tells the other viewers of a shipment's comments that the caller is typing,
// @Description or has stopped.
// @ID signalShipmentCommentTyping
// @Tags Realtime
// @Accept json
// @Param shipmentID path string true "Shipment ID"
// @Param request body typingRequest true "Typing signal"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /shipments/{shipmentID}/comments/typing/ [post]
func (h *Handler) typingShipmentComments(c *gin.Context) {
	scope, ok := h.shipmentCommentsScope(c)
	if !ok {
		return
	}

	var body typingRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err := h.gateway.SignalTyping(c.Request.Context(), &services.RealtimeTypingRequest{
		RealtimeScopeRequest: *scopeRequest(c, body.ConnectionID, scope),
		Stop:                 body.Stop,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) shipmentCommentsScope(c *gin.Context) (string, bool) {
	shipmentID := c.Param("shipmentID")
	if !pulid.LooksLike(shipmentID) {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"shipmentId", errortypes.ErrInvalid, "Shipment ID is invalid",
		))
		return "", false
	}

	return shipmentCommentsNS + shipmentID, true
}

func scopeRequest(c *gin.Context, connectionID, scope string) *services.RealtimeScopeRequest {
	authCtx := authctx.GetAuthContext(c)

	return &services.RealtimeScopeRequest{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
		ConnectionID:   connectionID,
		Scope:          scope,
	}
}

// cursorFrom reads where the reader left off. The header is what a browser's
// reader sends after a dropped connection; the query parameter serves a client
// that cannot set headers on the request that opens the stream.
func cursorFrom(c *gin.Context) string {
	if cursor := c.GetHeader(lastEventIDHeader); cursor != "" {
		return cursor
	}

	return c.Query("lastEventId")
}
