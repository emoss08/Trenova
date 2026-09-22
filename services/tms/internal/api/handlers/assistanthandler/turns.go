package assistanthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// lastEventIDHeader is where a reconnecting browser puts the cursor it left
// off at, as the server-sent events specification says it should.
const lastEventIDHeader = "Last-Event-ID"

// activeTurn names the reply a conversation is still producing, so reopening
// it rejoins rather than showing a thread that looks finished.
func (h *Handler) activeTurn(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	turn, err := h.turns.Active(c.Request.Context(), repositories.ActiveAssistantTurnRequest{
		ThreadID:   req.ID,
		UserID:     req.UserID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	// Nothing in progress is the ordinary state of a conversation, not a
	// missing record: a 404 here would have every client treating the common
	// case as an error.
	c.JSON(http.StatusOK, gin.H{"turn": turn})
}

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

	turn, err := h.turns.Get(c.Request.Context(), repositories.GetAssistantTurnRequest{
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

	err = h.turns.Relay(c.Request.Context(), assistantturnservice.RelayRequest{
		Turn:   turn,
		Cursor: cursorFrom(c),
	}, func(frame serviceports.TurnStreamFrame) error {
		// The frame's payload came out of redis already encoded and leaves
		// the same way: decoding it here to encode it again would spend the
		// stream's throughput learning nothing, and would let a frame this
		// relay cannot parse kill a turn it was only meant to carry.
		stream.EmitRaw(frame.ID, frame.Event, frame.Data)

		return nil
	})
	if err != nil {
		h.logger.Error("assistant turn relay failed",
			zap.String("turn", turnID.String()),
			zap.Error(err),
		)
		_ = stream.Emit(serviceports.AssistantEventError, gin.H{
			"message": "The live view of this reply ended. It will appear in the conversation.",
		})
	}
}

// cursorFrom reads where the reader left off.
//
// The header is what a browser sends on its own after a dropped connection.
// The query parameter is for a client that opened the stream by hand and
// therefore never had one set for it.
func cursorFrom(c *gin.Context) string {
	if cursor := c.GetHeader(lastEventIDHeader); cursor != "" {
		return cursor
	}

	return c.Query("after")
}
