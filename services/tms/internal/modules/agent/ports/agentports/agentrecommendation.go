package agentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentsuggestion"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

const AgentMatchRateScope = "agent-match"

const (
	MaxAgentRecommendationTextRunes = 4000
	MaxAgentRecommendationTools     = 3
	MaxAgentMatchContextKinds       = 11
)

type AgentMatchRequest struct {
	Actor          *permissionports.RequestActor
	Text           string
	CurrentAgentID pulid.ID
	ContextKinds   []string
}

type AgentMatchTool struct {
	Name    string
	Summary string
}

type AgentMatch struct {
	Outcome  agentsuggestion.MatchOutcome
	Agent    *agentdefinition.Definition
	Delegate *agentdefinition.Definition
	Tools    []AgentMatchTool
}

type AgentRecommender interface {
	MatchAgent(ctx context.Context, req *AgentMatchRequest) (*AgentMatch, error)
}
