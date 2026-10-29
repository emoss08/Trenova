package agenttoolstest

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/shared/pulid"
)

func TexasState() *FakeStates {
	return &FakeStates{State: &usstate.UsState{
		ID:           pulid.MustNew("us_"),
		Abbreviation: "TX",
		Name:         "Texas",
	}}
}

func IdempotentParams(params map[string]any) *agenttoolports.ToolExecuteParams {
	execute := ExecuteParams(params)
	execute.IdempotencyKey = pulid.MustNew("idem_").String()

	return execute
}
