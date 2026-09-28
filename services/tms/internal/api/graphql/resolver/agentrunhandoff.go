package resolver

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func agentRunParentOwnerKind(run *agent.AgentRun) (*gqlmodel.AgentRunEventOwnerKind, error) {
	if run.ParentOwnerKind == "" {
		return nil, nil
	}

	kind := gqlmodel.AgentRunEventOwnerKind(run.ParentOwnerKind)
	if !kind.IsValid() {
		return nil, fmt.Errorf("unknown agent run parent owner kind: %s", run.ParentOwnerKind)
	}

	return &kind, nil
}

func agentRunHandedBy(
	ctx context.Context,
	run *agent.AgentRun,
) (*agentdefinition.Definition, error) {
	if run.ParentOwnerKind != agent.RunOwnerAssistantTurn ||
		run.SubjectType != agent.SubjectAssistantThread || run.SubjectID.IsNil() {
		return nil, nil
	}

	l, ok := loaders.FromContext(ctx)
	if !ok {
		return nil, nil
	}

	thread, err := l.ThreadAgentByID.Load(ctx, run.SubjectID.String())
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}
	if thread.AgentDefinitionID.IsNil() || thread.AgentDefinitionID == run.AgentDefinitionID {
		return nil, nil
	}

	definition, err := l.AgentDefinitionByID.Load(ctx, thread.AgentDefinitionID.String())
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	return definition, nil
}
