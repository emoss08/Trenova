package agentdefinitionhandler

import (
	"context"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type dryRunRequest struct {
	// AgentID is the agent the draft edits; empty for a new one.
	AgentID pulid.ID         `json:"agentId"`
	Draft   saveAgentRequest `json:"draft"`
	Prompt  string           `json:"prompt"`
}

// dryRun tries a draft agent once against live data, with every write
// simulated, and streams what it does as server-sent events: the assistant's
// own events as they happen, then dry_run_steps naming what each tool call
// would have come to, then done or error. Nothing is saved.
//
// The run is started and read on this one request, so the stream can only
// ever be read by the person who asked for it.
func (h *Handler) dryRun(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	ctx := c.Request.Context()

	var body dryRunRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	tenant := tenantFromAuthContext(authCtx)
	draft, err := h.service.Draft(ctx, body.Draft.toServiceRequest(body.AgentID, tenant))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authCtx)
	runID := pulid.MustNew(agent.EvaluationIDPrefix)
	run, err := assistantjobs.StartDryRunWorkflow(ctx, h.workflows, &assistantjobs.AgentDryRunPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			UserID:         tenant.UserID,
		},
		RunID:      runID,
		Actor:      actor,
		Definition: draft,
		Prompt:     body.Prompt,
	})
	if err != nil {
		h.eh.HandleError(c, errortypes.NewBusinessError("The dry run could not start").WithInternal(err))
		return
	}

	stream, err := helpers.OpenEventStream(c, helpers.EventStreamOptions{})
	if err != nil {
		h.eh.HandleError(c, errortypes.NewBusinessError("Streaming is not supported"))
		return
	}
	defer stream.Close()

	sawTerminal := false
	err = h.streams.Read(ctx, serviceports.ReadTurnStreamRequest{
		Ref: serviceports.TurnStreamRef{
			TenantInfo: tenant,
			TurnID:     runID,
			WorkflowID: run.GetID(),
		},
		OnFrame: func(frame serviceports.TurnStreamFrame) error {
			if frame.Terminal() {
				sawTerminal = true
			}
			stream.EmitRaw(frame.ID, frame.Event, frame.Data)
			return nil
		},
	})
	if ctx.Err() != nil && !sawTerminal {
		// The person left. What is left of the run would be paid for and read
		// by nobody.
		if cancelErr := h.workflows.CancelWorkflow(
			context.WithoutCancel(ctx), run.GetID(), run.GetRunID(),
		); cancelErr != nil {
			h.logger.Warn("could not stop an abandoned dry run", zap.Error(cancelErr))
		}
		return
	}
	if err != nil || !sawTerminal {
		if err != nil {
			h.logger.Warn("dry run stream ended early", zap.Error(err))
		}
		_ = stream.Emit(serviceports.AssistantEventError, gin.H{
			"message": "The dry run stopped before it finished. Try again in a moment.",
		})
	}
}
