package agentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantproposalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type AgentsPausedLoaderFactoryParams struct {
	fx.In

	Controls tenantports.AgentControlRepository
}

type AgentsPausedLoaderFactory struct {
	controls tenantports.AgentControlRepository
}

func NewAgentsPausedLoaderFactory(p AgentsPausedLoaderFactoryParams) *AgentsPausedLoaderFactory {
	return &AgentsPausedLoaderFactory{controls: p.Controls}
}

func newAgentsPausedLoader(f *AgentsPausedLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, bool] {
	return loadercore.NewLoader(loadercore.BatchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]bool, error) {
			paused := make(map[pulid.ID]bool, len(ids))
			if f.controls == nil {
				return paused, nil
			}
			control, err := f.controls.GetOrCreate(ctx, tenantInfo)
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				paused[id] = id == tenantInfo.OrgID && control.ShadowMode
			}

			return paused, nil
		},
	))
}

type AgentPlanHoldLoaderFactoryParams struct {
	fx.In

	Plans       agentports.AgentPlanRepository
	PlanService agentports.AgentPlanService
}

type AgentPlanHoldLoaderFactory struct {
	plans       agentports.AgentPlanRepository
	planService agentports.AgentPlanService
}

func NewAgentPlanHoldLoaderFactory(p AgentPlanHoldLoaderFactoryParams) *AgentPlanHoldLoaderFactory {
	return &AgentPlanHoldLoaderFactory{plans: p.Plans, planService: p.PlanService}
}

func newAgentPlanHoldLoader(f *AgentPlanHoldLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *assistantproposalports.ProposalHold] {
	return loadercore.NewLoader(loadercore.BatchValueFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*assistantproposalports.ProposalHold, error) {
			if f.plans == nil || f.planService == nil {
				return map[pulid.ID]*assistantproposalports.ProposalHold{}, nil
			}
			plans, err := f.plans.ListByIDs(ctx, agentports.ListAgentPlansByIDsRequest{
				IDs:        ids,
				TenantInfo: tenantInfo,
			})
			if err != nil {
				return nil, err
			}

			return f.planService.PlanHolds(ctx, tenantInfo, plans)
		},
	))
}

var (
	agentsPausedKey  = loadercore.NewKey(newAgentsPausedLoader)
	agentPlanHoldKey = loadercore.NewKey(newAgentPlanHoldLoader)
)

func AgentsPaused(ctx context.Context, orgID string) (bool, error) {
	return loadercore.Load(ctx, agentsPausedKey, orgID)
}

func AgentPlanHold(
	ctx context.Context,
	planID string,
) (*assistantproposalports.ProposalHold, error) {
	return loadercore.Load(ctx, agentPlanHoldKey, planID)
}

func (f *AgentPlanHoldLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func (f *AgentsPausedLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
