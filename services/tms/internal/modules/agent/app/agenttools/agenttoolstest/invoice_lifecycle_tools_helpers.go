package agenttoolstest

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/shared/pulid"
)

func Approved(params agenttoolports.ToolExecuteParams) agenttoolports.ToolExecuteParams {
	params.ProposalID = pulid.MustNew("ap_")
	params.IdempotencyKey = params.ProposalID.String()

	return params
}
