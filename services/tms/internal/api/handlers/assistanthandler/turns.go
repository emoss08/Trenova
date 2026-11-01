package assistanthandler

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/sessionwatch"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantturnservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantturnports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// streamTurn relays one turn's events to a reader.
//
// The turn is read scoped to the person asking before anything is streamed,
// so a guessed id reads nothing. That check is this endpoint's whole security
// surface: everything after it is a pipe.
func (h *Handler) streamTurn(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	turnID, err := pulid.Parse(c.Param("turnID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	turn, err := h.turns.Get(c.Request.Context(), &assistantturnports.GetAssistantTurnRequest{
		ID:         turnID,
		TenantInfo: tenantFromAuthContext(authCtx),
		UserID:     authCtx.UserID,
	})
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

	ctx, stopWatching := h.sessions.Context(c.Request.Context(), authCtx.SessionID)
	defer stopWatching()

	err = h.turns.Relay(ctx, assistantturnservice.RelayRequest{
		Turn:   turn,
		Cursor: helpers.EventCursor(c),
	}, func(frame assistantstreamports.TurnStreamFrame) error {
		// The frame's payload arrives already encoded and leaves the same
		// way: decoding it here to encode it again would spend the stream's
		// throughput learning nothing.
		stream.EmitRaw(frame.ID, frame.Event, frame.Data)

		return nil
	})
	if errors.Is(context.Cause(ctx), sessionwatch.ErrSessionEnded) {
		return
	}
	if err != nil {
		h.logger.Error("assistant turn relay failed",
			zap.String("turn", turnID.String()),
			zap.Error(err),
		)
		_ = stream.Emit(assistantstreamports.AssistantEventError, gin.H{
			"message": "The live view of this reply ended. It will appear in the conversation.",
		})
	}
}
