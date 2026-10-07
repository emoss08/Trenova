package aicontrolresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aituneup"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

func tuneUpModel(view *services.AITuneUpView) *gqlmodel.AITuneUp {
	tuneUp := view.TuneUp
	out := &gqlmodel.AITuneUp{
		ID:            tuneUp.ID.String(),
		Kind:          tuneUp.Kind,
		Agent:         view.Agent,
		Provider:      view.Provider,
		OtherProvider: view.OtherProvider,
		Evidence:      &tuneUp.Evidence,
		Status:        tuneUp.Status,
		ComputedAt:    int(tuneUp.ComputedAt),
		Version:       int(tuneUp.Version),
	}
	if out.Evidence.Tasks == nil {
		out.Evidence.Tasks = []aiprovider.Task{}
	}
	if tuneUp.ToolName != "" {
		toolName := tuneUp.ToolName
		out.ToolName = &toolName
	}
	if tuneUp.Task != "" {
		task := tuneUp.Task
		out.Task = &task
	}
	if tuneUp.DismissedUntil != nil {
		until := int(*tuneUp.DismissedUntil)
		out.DismissedUntil = &until
	}
	return out
}

func optionalTier(tier agent.AutonomyTier) *agent.AutonomyTier {
	if tier == "" {
		return nil
	}
	return &tier
}

func changedResource(kind aituneup.Kind) permission.Resource {
	switch kind {
	case aituneup.KindReorderProviders, aituneup.KindAssignTask:
		return permission.ResourceAIProvider
	default:
		return permission.ResourceAgentDefinition
	}
}

func (r *MutationResolver) requireApplyPermission(ctx context.Context, kind aituneup.Kind) error {
	_, err := r.RequirePermission(ctx, changedResource(kind), permission.OpUpdate)
	return err
}
