package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// loadAgentRun reads the run behind a proposal or plan through the
// per-request loader, so a page of rows asks once. A run that is gone leaves
// the field empty rather than failing the row.
func loadAgentRun(ctx context.Context, runID pulid.ID) (*agent.AgentRun, error) {
	if runID.IsNil() {
		return nil, nil
	}

	l, ok := loaders.FromContext(ctx)
	if !ok {
		return nil, nil
	}

	run, err := l.AgentRunByID.Load(ctx, runID.String())
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	return run, nil
}
