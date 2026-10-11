package agenttoolstest

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/pulid"
)

func Approved(params agentports.ToolExecuteParams) agentports.ToolExecuteParams {
	params.ProposalID = pulid.MustNew("ap_")
	params.IdempotencyKey = params.ProposalID.String()

	return params
}
