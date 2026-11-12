package deskcasehandler

import (
	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/sessionwatch"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordcaseports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
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

	Cases                recordcaseports.RecordCaseService
	Streams              assistantstreamports.TurnStreamReader
	Sessions             *sessionwatch.Watch
	Logger               *zap.Logger
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

// Handler serves what an agent is writing on a record's case as it writes
// it. Everything else about a case is read through GraphQL; a stream is not.
type Handler struct {
	cases    recordcaseports.RecordCaseService
	streams  assistantstreamports.TurnStreamReader
	sessions *sessionwatch.Watch
	l        *zap.Logger
	eh       *helpers.ErrorHandler
	pm       *middleware.PermissionMiddleware
}

//nolint:gocritic // fx passes params by value
func New(p Params) *Handler {
	return &Handler{
		cases:    p.Cases,
		streams:  p.Streams,
		sessions: p.Sessions,
		l:        p.Logger.Named("handler.desk-case"),
		eh:       p.ErrorHandler,
		pm:       p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/desk-cases")
	resource := permission.ResourceWatchtower.String()

	api.GET(
		"/:kind/:recordID/stream/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.stream,
	)
}

// stream follows the agent working the case: its words as server-sent
// events, then done or error. The case is read as the caller, so only someone
// who may see the case can follow it, and the run followed is the one the
// case itself says is working, never one a caller names. A case no agent is
// working answers done at once.
func (h *Handler) stream(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	ctx := c.Request.Context()

	recordID, err := pulid.MustParse(c.Param("recordID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	tenant := pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	found, err := h.cases.Case(ctx, &recordcaseports.CaseRequest{
		TenantInfo: tenant,
		Kind:       recordkind.Kind(c.Param("kind")),
		ID:         recordID,
	}, actorutil.FromAuthContext(authCtx))
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

	if found.Stream == "" {
		_ = stream.Emit(assistantstreamports.AssistantEventDone, gin.H{})
		return
	}

	watched, stopWatching := h.sessions.SessionContext(ctx, authCtx.SessionID)
	defer stopWatching()

	sawTerminal := false
	err = h.streams.Read(watched, &assistantstreamports.ReadTurnStreamRequest{
		Ref: assistantstreamports.TurnStreamRef{
			TenantInfo: tenant,
			WorkflowID: found.Stream,
		},
		Cursor: c.GetHeader("Last-Event-ID"),
		OnFrame: func(frame assistantstreamports.TurnStreamFrame) error {
			if frame.Terminal() {
				sawTerminal = true
			}
			stream.EmitRaw(frame.ID, frame.Event, frame.Data)
			return nil
		},
	})
	if watched.Err() != nil || sawTerminal {
		return
	}
	if err != nil {
		h.l.Debug("a case's stream ended before its run said so", zap.Error(err))
	}
	// The run closed, or hosts no stream, before a last word reached this
	// reader. The case itself says how it ended, so the reader is told to
	// read it again rather than that anything failed.
	_ = stream.Emit(assistantstreamports.AssistantEventDone, gin.H{})
}
