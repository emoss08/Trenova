package tooldispatch

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/argcheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/recordnumbers"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

func (s *Dispatcher) contract(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	call *llmports.ToolCall,
	schema map[string]any,
) (argcheck.Contract, error) {
	contract, err := argcheck.ContractCall(
		s.arguments, call.Name, schema, call.Arguments, organizationToday(req),
	)
	if err != nil || len(contract.Numbers()) == 0 {
		return contract, err
	}

	return s.numbers.Resolve(ctx, &recordnumbers.Request{
		Actor:      req.Actor,
		DataAccess: req.Definition.DataAccessSensitivity(),
		Contract:   contract,
	})
}
