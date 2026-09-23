package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

/*
What leaves the organization is a person's decision.

A desk earns autonomy one clean approval at a time, and an organization can
raise an agent's ceiling as far as it likes. Neither can carry these tools
past approval: each sends something outside, to a customer, a driver, a
carrier or whoever wrote in, and the run that composed it may have been
started by an email an outsider wrote. A model can be talked into sending
anything it is allowed to send; the only control that does not depend on the
model is a person reading it first.
*/

const egressCeiling = agent.TierActWithApproval

func (t *emailCustomerTool) TierCeiling() agent.AutonomyTier            { return egressCeiling }
func (t *sendDetentionNoticeTool) TierCeiling() agent.AutonomyTier      { return egressCeiling }
func (t *notifyDriverTool) TierCeiling() agent.AutonomyTier             { return egressCeiling }
func (t *requestMissingDocsTool) TierCeiling() agent.AutonomyTier       { return egressCeiling }
func (t *requestCredentialRenewalTool) TierCeiling() agent.AutonomyTier { return egressCeiling }
func (t *tenderToRoutingGuideTool) TierCeiling() agent.AutonomyTier     { return egressCeiling }
func (t *tenderToCarriersTool) TierCeiling() agent.AutonomyTier         { return egressCeiling }
func (t *replyToInboundMessageTool) TierCeiling() agent.AutonomyTier    { return egressCeiling }

// TierLimit holds a note a customer or a driver will read for approval. An
// internal note stays automatic: it changes nothing and can be deleted.
func (t *addShipmentCommentTool) TierLimit(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) agent.AutonomyTier {
	switch commentVisibility(optionalString(params.Params, "visibility")) {
	case shipment.CommentVisibilityCustomer, shipment.CommentVisibilityDriver:
		return egressCeiling
	default:
		return agent.TierAutoExecute
	}
}

var (
	_ serviceports.ToolTierCeiling = (*replyToInboundMessageTool)(nil)
	_ serviceports.ToolTierLimiter = (*addShipmentCommentTool)(nil)
)
