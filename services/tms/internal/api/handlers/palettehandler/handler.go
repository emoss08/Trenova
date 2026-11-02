package palettehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/sessionwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Palette              paletteports.PaletteService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	Sessions             *sessionwatch.Watch
	Logger               *zap.Logger
}

// Handler serves command palette questions. Every route names the question
// by the person in the session, so a guessed id reaches nothing of anyone
// else's.
type Handler struct {
	palette  paletteports.PaletteService
	eh       *helpers.ErrorHandler
	pm       *middleware.PermissionMiddleware
	sessions *sessionwatch.Watch
	logger   *zap.Logger
}

func New(p Params) *Handler {
	return &Handler{
		palette:  p.Palette,
		eh:       p.ErrorHandler,
		pm:       p.PermissionMiddleware,
		sessions: p.Sessions,
		logger:   p.Logger.Named("palettehandler"),
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/assistant/ask")
	resource := permission.ResourceAssistant.String()

	api.POST("/", h.pm.RequirePermission(resource, permission.OpRead), h.ask)
	api.GET("/:askID/stream/", h.pm.RequirePermission(resource, permission.OpRead), h.stream)
	api.DELETE("/:askID/", h.pm.RequirePermission(resource, permission.OpRead), h.stop)
	// Carrying a question into the Desk starts a conversation, so it needs
	// what starting one needs.
	api.POST("/:askID/keep/", h.pm.RequirePermission(resource, permission.OpCreate), h.keep)
}

type askRequest struct {
	Content  string                        `json:"content"`
	Context  *agentpagecontext.PageContext `json:"context"`
	Mentions []agentpagecontext.EntityRef  `json:"mentions"`
}

func (h *Handler) ask(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var body askRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if body.Context != nil {
		body.Context.Draft = nil
	}

	ask, err := h.palette.Ask(c.Request.Context(), &paletteports.AskRequest{
		Question: paletteports.Question{
			Content:  body.Content,
			Page:     body.Context,
			Mentions: body.Mentions,
		},
		TenantInfo: actorutil.TenantInfoFrom(authCtx),
	}, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusAccepted, ask)
}

func (h *Handler) stream(c *gin.Context) {
	ref, err := askRef(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	stream, err := helpers.OpenEventStream(c, helpers.EventStreamOptions{})
	if err != nil {
		h.eh.HandleError(c, errortypes.NewBusinessError("Streaming is not supported"))
		return
	}
	defer stream.Close()

	ctx, stopWatching := h.sessions.Context(
		c.Request.Context(),
		authctx.GetAuthContext(c).SessionID,
		authctx.GetSessionSecretHash(c),
	)
	defer stopWatching()

	err = h.palette.Relay(ctx, &paletteports.RelayRequest{
		Ref:    ref,
		Cursor: helpers.EventCursor(c),
		OnFrame: func(frame assistantstreamports.TurnStreamFrame) error {
			stream.EmitRaw(frame.ID, frame.Event, frame.Data)

			return nil
		},
	})
	if sessionwatch.Ended(ctx) {
		return
	}
	if err != nil {
		h.logger.Warn("palette answer relay failed",
			zap.String("askId", ref.AskID.String()), zap.Error(err))
		_ = stream.Emit(assistantstreamports.AssistantEventError, gin.H{
			"message": "The answer stopped arriving. Ask again, or open it in the Desk.",
		})
	}
}

func (h *Handler) stop(c *gin.Context) {
	ref, err := askRef(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if err = h.palette.Stop(c.Request.Context(), ref); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) keep(c *gin.Context) {
	ref, err := askRef(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	kept, err := h.palette.Keep(
		c.Request.Context(),
		ref,
		actorutil.FromAuthContext(authctx.GetAuthContext(c)),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusCreated, kept)
}

// askRef names the question by the session's person, never by anything the
// request says about whose it is.
func askRef(c *gin.Context) (paletteports.AskRef, error) {
	askID, err := pulid.Parse(c.Param("askID"))
	if err != nil {
		return paletteports.AskRef{}, err
	}

	authCtx := authctx.GetAuthContext(c)
	tenant := actorutil.TenantInfoFrom(authCtx)
	tenant.UserID = authCtx.UserID

	return paletteports.AskRef{AskID: askID, TenantInfo: tenant}, nil
}
