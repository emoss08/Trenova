package assistanthandler

import (
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/gin-gonic/gin"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
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
		workflowID := assistantjobs.WorkflowIDFor(turn.ID)

		_, err := h.workflows.StartWorkflow(c.Request.Context(), client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: temporaltype.TaskQueueAgentChat.String(),
			// One turn, one execution. A duplicate start is a bug rather than
			// a second question, and rejecting it is how it stays visible.
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		}, assistantjobs.AssistantTurnWorkflowName, &assistantjobs.AssistantTurnPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: authCtx.OrganizationID,
				BusinessUnitID: authCtx.BusinessUnitID,
				UserID:         authCtx.UserID,
				Timestamp:      timeutils.NowUnix(),
			},
			TurnID:   turn.ID,
			ThreadID: threadID,
			Actor:    requestActorFromAuthContext(authCtx),
			Content:  body.Content,
			Request: assistantjobs.AssistantTurnRequest{
				Page:                  body.page(),
				Mentions:              body.Mentions,
				AttachmentDocumentIDs: body.AttachmentDocumentIDs,
				PreferredProviderID:   providerID,
				ProviderChosen:        providerChosen,
			},
		})
		if err != nil {
			return "", err
		}

		return workflowID, nil
	}
}

// cancelTurn builds the call that stops a turn.
func (h *Handler) cancelTurn(c *gin.Context) func(string) error {
	return func(workflowID string) error {
		return h.workflows.CancelWorkflow(c.Request.Context(), workflowID, "")
	}
}
