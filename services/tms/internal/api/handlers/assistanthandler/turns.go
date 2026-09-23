package assistanthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
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
		// The frame's payload arrives already encoded and leaves the same
		// way: decoding it here to encode it again would spend the stream's
		// throughput learning nothing.
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

// startTurnResponse is a question handed to a worker: the turn to watch and
// where to watch it.
type startTurnResponse struct {
	TurnID    pulid.ID                         `json:"turnId"`
	ThreadID  pulid.ID                         `json:"threadId"`
	StreamURL string                           `json:"streamUrl"`
	Status    conversation.AssistantTurnStatus `json:"status"`
	// Thread is set when asking the question made the thread, as a quick
	// question does.
	Thread *conversation.Thread `json:"thread,omitempty"`
}

func turnStarted(turn *conversation.AssistantTurn) startTurnResponse {
	return startTurnResponse{
		TurnID:    turn.ID,
		ThreadID:  turn.ThreadID,
		StreamURL: "/api/v1/assistant/turns/" + turn.ID.String() + "/stream/",
		Status:    turn.Status,
	}
}

// startTurn hands a question to a worker and returns immediately.
//
// The reply is not on this response. It arrives on the turn's stream, which
// the caller attaches to with the id returned here — so the answer survives
// this request ending, whether that is a deploy, a dropped connection or
// somebody closing the tab.
func (h *Handler) startTurn(c *gin.Context) {
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

	turn, _, err := h.askWorker(c, threadID, &body)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, turnStarted(turn))
}

// sendMessage asks a question and answers with the saved turn, for a caller
// that would rather wait than follow a stream. The turn is the same one a
// streaming reader follows; this request just waits for its workflow.
func (h *Handler) sendMessage(c *gin.Context) {
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

	_, run, err := h.askWorker(c, threadID, &body)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var outcome assistantjobs.AssistantTurnResult
	if err = run.Get(c.Request.Context(), &outcome); err != nil {
		if temporal.IsCanceledError(err) {
			h.eh.HandleError(c, errortypes.NewBusinessError(
				"This reply was stopped. What was said so far has been kept in the conversation.",
			))
			return
		}
		h.eh.HandleError(c, err)
		return
	}
	if outcome.Result == nil {
		h.eh.HandleError(c, errortypes.NewBusinessError(outcome.Message))
		return
	}

	// A refusal is a successful request with a declined answer, not an error: the
	// turn was processed, recorded, and explained.
	c.JSON(http.StatusOK, outcome.Result)
}

// ask starts a quick question: the hidden thread it is answered on, and the
// turn answering it. The thread comes back first so the reader can open it in
// the Desk even if the answer fails.
func (h *Handler) ask(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var body askRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	thread, err := h.service.StartAsk(c.Request.Context(), &serviceports.AskRequest{
		Content:    body.Content,
		Page:       body.Context.page(),
		Mentions:   body.Mentions,
		TenantInfo: tenantFromAuthContext(authCtx),
	}, &actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	turn, _, err := h.askWorker(c, thread.ID, &sendMessageRequest{
		Content:  body.Content,
		Context:  body.Context,
		Mentions: body.Mentions,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	response := turnStarted(turn)
	response.Thread = thread
	c.JSON(http.StatusAccepted, response)
}

// askWorker records a turn and starts the workflow that answers it.
func (h *Handler) askWorker(
	c *gin.Context,
	threadID pulid.ID,
	body *sendMessageRequest,
) (*conversation.AssistantTurn, client.WorkflowRun, error) {
	authCtx := authctx.GetAuthContext(c)

	var run client.WorkflowRun
	turn, err := h.turns.StartTurn(
		c.Request.Context(),
		assistantturnservice.StartRequest{
			ThreadID:   threadID,
			UserID:     authCtx.UserID,
			TenantInfo: tenantFromAuthContext(authCtx),
			Origin:     conversation.AssistantTurnOriginPerson,
			Input:      body.Content,
		},
		func(turn *conversation.AssistantTurn) (string, error) {
			started, err := h.startWorkflow(c, turn, authCtx, body)
			if err != nil {
				return "", err
			}
			run = started

			return started.GetID(), nil
		},
	)
	if err != nil {
		return nil, nil, err
	}

	return turn, run, nil
}

// startWorkflow starts the execution answering a turn.
//
// It is kept apart from the service that records the turn so that service
// stays ignorant of Temporal: the record is the product's, the execution is an
// implementation of it, and the one that outlives the other is the record.
func (h *Handler) startWorkflow(
	c *gin.Context,
	turn *conversation.AssistantTurn,
	authCtx *authctx.AuthContext,
	body *sendMessageRequest,
) (client.WorkflowRun, error) {
	providerID, providerChosen := body.provider()

	return assistantjobs.StartTurnWorkflow(c.Request.Context(), h.workflows, turn,
		assistantjobs.TurnStart{
			Actor:   requestActorFromAuthContext(authCtx),
			Content: body.Content,
			Request: assistantjobs.AssistantTurnRequest{
				Page:                  body.page(),
				Mentions:              body.Mentions,
				AttachmentDocumentIDs: body.AttachmentDocumentIDs,
				PreferredProviderID:   providerID,
				ProviderChosen:        providerChosen,
				FollowUpProposalID:    body.FollowUpProposalID,
			},
		})
}

// stopTurn ends a reply nobody is waiting for any more.
func (h *Handler) stopTurn(c *gin.Context) {
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

	err = h.turns.Stop(c.Request.Context(), turn, func(workflowID string) error {
		return h.workflows.CancelWorkflow(c.Request.Context(), workflowID, "")
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusAccepted)
}
