package common

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

type ClassifiedTool[R, P any] struct {
	ReportingReceivableTool[R, P]
	Egress   []agent.EgressClass                                              `json:"-"`
	Classify func(agenttoolports.ToolExecuteParams) agenttoolports.CallPolicy `json:"-"`
}

func (t ClassifiedTool[R, P]) Policy() agenttoolports.ToolPolicy {
	policy := t.ReportingReceivableTool.Policy()
	policy.Egress = t.Egress
	policy.Classify = t.Classify

	return policy
}
