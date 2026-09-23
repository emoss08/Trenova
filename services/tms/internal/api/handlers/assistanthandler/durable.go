package assistanthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

// turnStarter builds the call that hands a turn to a worker.
//
// It is a closure rather than a method so the service that records the turn
// stays ignorant of Temporal: the record is the product's, the execution is an
// implementation of it, and the one that outlives the other is the record.
func (h *Handler) turnStarter(
	c *gin.Context,
	threadID pulid.ID,
	authCtx *authctx.AuthContext,
	body startTurnRequest,
) func(*conversation.AssistantTurn) (string, error) {
	return func(turn *conversation.AssistantTurn) (string, error) {
		providerID, providerChosen := body.provider()

		return assistantjobs.StartTurnWorkflow(c.Request.Context(), h.workflows, turn, assistantjobs.TurnStart{
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
}

// cancelTurn builds the call that stops a turn.
func (h *Handler) cancelTurn(c *gin.Context) func(string) error {
	return func(workflowID string) error {
		return h.workflows.CancelWorkflow(c.Request.Context(), workflowID, "")
	}
}

// durableTurnsAvailable reports whether a question can be handed to a worker.
//
// Two things have to be true, and the second is not implied by the first:
// somebody turned it on, and there is a Temporal client to hand it to. A
// developer with the flag set and no server would otherwise be told the
// durable path exists and then have every question fail.
func (h *Handler) durableTurnsAvailable() bool {
	return h.ai.DurableTurnsEnabled() && h.workflows != nil && h.workflows.Enabled()
}

// capabilities tells the client which way to ask a question.
//
// It exists so the two paths can roll forward independently. A client that
// asked durably against a server that cannot answer that way would fail every
// question; one that asks here first degrades to the path that always works.
func (h *Handler) capabilities(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"durableTurns": h.durableTurnsAvailable()})
}
