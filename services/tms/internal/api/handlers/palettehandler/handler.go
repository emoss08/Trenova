package palettehandler

import (
	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
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
	Logger               *zap.Logger
}

// Handler serves command palette questions. Every route names the question
// by the person in the session, so a guessed id reaches nothing of anyone
// else's.
type Handler struct {
	palette paletteports.PaletteService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
	logger  *zap.Logger
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Handler {
	return &Handler{
		palette: p.Palette,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
		logger:  p.Logger.Named("palettehandler"),
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/assistant/ask")
	resource := permission.ResourceAssistant.String()

	api.GET("/:askID/stream/", h.pm.RequirePermission(resource, permission.OpRead), h.stream)
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

	err = h.palette.Relay(c.Request.Context(), &paletteports.RelayRequest{
		Ref:    ref,
		Cursor: helpers.EventCursor(c),
		OnFrame: func(frame assistantstreamports.TurnStreamFrame) error {
			stream.EmitRaw(frame.ID, frame.Event, frame.Data)

			return nil
		},
	})
	if err != nil {
		h.logger.Warn("palette answer relay failed",
			zap.String("askId", ref.AskID.String()), zap.Error(err))
		_ = stream.Emit(assistantstreamports.AssistantEventError, gin.H{
			"message": "The answer stopped arriving. Ask again, or open it in the Desk.",
		})
	}
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
