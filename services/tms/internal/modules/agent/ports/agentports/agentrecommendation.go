package agentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MaxAgentRecommendationTextRunes = 4000
	MaxAgentRecommendationTools     = 3
)

type AgentRecommendationRequest struct {
	Actor          *permissionports.RequestActor
	Text           string
	CurrentAgentID pulid.ID
}

type AgentRecommendation struct {
	Agent   *agentdefinition.Definition
	Current bool
	Tools   []string
}

type AgentRecommender interface {
	RecommendAgent(
		ctx context.Context,
		req *AgentRecommendationRequest,
	) (*AgentRecommendation, error)
}
