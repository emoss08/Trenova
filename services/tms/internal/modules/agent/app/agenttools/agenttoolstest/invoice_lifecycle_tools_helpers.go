package agenttoolstest

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/shared/pulid"
)

func Approved(params *agenttoolports.ToolExecuteParams) *agenttoolports.ToolExecuteParams {
	approved := *params
	approved.ProposalID = pulid.MustNew("ap_")
	approved.IdempotencyKey = approved.ProposalID.String()

	return &approved
}
