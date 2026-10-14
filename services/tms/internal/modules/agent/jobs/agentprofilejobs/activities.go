package agentprofilejobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Profiles   agentprofileports.Generator
	Facts      agentprofileports.FactsReader
	Completion agentports.CompletionService
	Tenants    tenantports.TenantSyncRepository
	Logger     *zap.Logger
}

type Activities struct {
	profiles   agentprofileports.Generator
	facts      agentprofileports.FactsReader
	completion agentports.CompletionService
	tenants    tenantports.TenantSyncRepository
	l          *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		profiles:   p.Profiles,
		facts:      p.Facts,
		completion: p.Completion,
		tenants:    p.Tenants,
		l:          p.Logger.Named("job.agent-profile"),
	}
}

func (a *Activities) ListOrganizationsActivity(
	ctx context.Context,
	input *ListOrganizationsInput,
) (*temporaljobs.TenantPage, error) {
	organizations, err := a.tenants.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	return temporaljobs.OrganizationPage(organizations, input.After, input.Limit), nil
}

func (a *Activities) ListActiveUsersActivity(
	ctx context.Context,
	input *ActiveUsersInput,
) ([]pulid.ID, error) {
	users, err := a.facts.ActiveUsers(ctx, agentprofileports.ActiveUsersRequest{
		TenantInfo: input.TenantInfo(),
		Since:      input.Since,
		Limit:      agentprofileports.MaxActiveUsers,
	})
	if err != nil {
		return nil, fmt.Errorf("list the people who talked to agents: %w", err)
	}

	return users, nil
}

func (a *Activities) PrepareProfileActivity(
	ctx context.Context,
	req *agentprofileports.GenerateRequest,
) (*agentprofileports.GenerationPlan, error) {
	plan, err := a.profiles.Prepare(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("work out the profile's facts: %w", err)
	}

	return plan, nil
}

// ProfileModelActivity writes the profile. It heartbeats on a timer, because a
// model call can outlast the heartbeat timeout without producing anything.
func (a *Activities) ProfileModelActivity(
	ctx context.Context,
	plan *agentprofileports.GenerationPlan,
) (*ModelReply, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	result, err := a.completion.CompleteStructured(ctx, plan.Request)
	if err != nil {
		if errors.Is(err, agentports.ErrNoProviderConfigured) {
			return &ModelReply{NoProvider: true}, nil
		}

		return nil, modelcall.Classify(err)
	}

	return &ModelReply{Result: result}, nil
}

func (a *Activities) FinishProfileActivity(
	ctx context.Context,
	req *agentprofileports.FinishGenerationRequest,
) (*agentprofileports.GenerationOutcome, error) {
	outcome, err := a.profiles.Finish(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("keep the profile: %w", err)
	}

	return outcome, nil
}

func (a *Activities) PruneProfileActivity(
	ctx context.Context,
	ref *agentprofileports.SubjectRef,
) error {
	if err := a.profiles.Prune(ctx, *ref); err != nil {
		return fmt.Errorf("prune the profile's old versions: %w", err)
	}

	return nil
}
