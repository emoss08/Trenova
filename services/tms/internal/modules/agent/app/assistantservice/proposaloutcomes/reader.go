package proposaloutcomes

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/proposalstores"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"go.uber.org/zap"
)

type Reader struct {
	Logger    *zap.Logger
	Proposals proposalstores.ProposalStore
	Decisions agentdecisionports.AgentDecisionRepository
}
