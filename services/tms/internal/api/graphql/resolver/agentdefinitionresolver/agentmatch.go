package agentdefinitionresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type agentMatchInput struct {
	text           string
	currentAgentID *string
	contextKinds   []string
}

func parseCurrentAgentID(raw *string) (pulid.ID, error) {
	if raw == nil || *raw == "" {
		return pulid.Nil, nil
	}
	id, err := pulid.MustParse(*raw)
	if err != nil {
		return pulid.Nil, errortypes.NewValidationError(
			"currentAgentId",
			errortypes.ErrInvalid,
			"The agent the question would go to must be an agent ID.",
		)
	}

	return id, nil
}

func matchAgent(
	ctx context.Context,
	recommender agentports.AgentRecommender,
	actor *permissionports.RequestActor,
	input agentMatchInput,
) (*agentports.AgentMatch, error) {
	currentAgentID, err := parseCurrentAgentID(input.currentAgentID)
	if err != nil {
		return nil, err
	}

	return recommender.MatchAgent(ctx, &agentports.AgentMatchRequest{
		Actor:          actor,
		Text:           input.text,
		CurrentAgentID: currentAgentID,
		ContextKinds:   input.contextKinds,
	})
}

func toMyAgentMatch(found *agentports.AgentMatch) *gqlmodel.MyAgentMatch {
	tools := make([]*gqlmodel.AgentMatchTool, 0, len(found.Tools))
	for _, tool := range found.Tools {
		tools = append(tools, &gqlmodel.AgentMatchTool{Name: tool.Name, Summary: tool.Summary})
	}

	return &gqlmodel.MyAgentMatch{
		Outcome: found.Outcome,
		Agent:   found.Agent,
		Tools:   tools,
	}
}
