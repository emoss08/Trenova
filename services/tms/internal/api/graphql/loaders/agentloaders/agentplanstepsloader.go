package agentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type proposalsByPlansLister interface {
	ListByPlans(
		ctx context.Context,
		req agentports.ListAgentProposalsByPlansRequest,
	) ([]*agent.AgentProposal, error)
}

type AgentPlanStepsByPlanIDLoaderFactoryParams struct {
	fx.In

	Proposals agentports.AgentProposalRepository
}

type AgentPlanStepsByPlanIDLoaderFactory struct {
	proposals proposalsByPlansLister
}

func NewAgentPlanStepsByPlanIDLoaderFactory(
	p AgentPlanStepsByPlanIDLoaderFactoryParams,
) *AgentPlanStepsByPlanIDLoaderFactory {
	return &AgentPlanStepsByPlanIDLoaderFactory{proposals: p.Proposals}
}

func newAgentPlanStepsByPlanIDLoader(f *AgentPlanStepsByPlanIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*agent.AgentProposal] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *AgentPlanStepsByPlanIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*agent.AgentProposal] {
	return loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*agent.AgentProposal, error) {
			steps, err := f.proposals.ListByPlans(ctx, agentports.ListAgentProposalsByPlansRequest{
				PlanIDs:    ids,
				TenantInfo: tenantInfo,
			})
			if err != nil {
				return nil, err
			}

			groups := make(map[pulid.ID][]*agent.AgentProposal, len(ids))
			for _, step := range steps {
				if step == nil || step.PlanID == nil {
					continue
				}
				groups[*step.PlanID] = append(groups[*step.PlanID], step)
			}

			return groups, nil
		},
	)
}

var agentPlanStepsByPlanIDKey = loadercore.NewKey(newAgentPlanStepsByPlanIDLoader)

func AgentPlanStepsByPlanID(ctx context.Context, key string) ([]*agent.AgentProposal, error) {
	return loadercore.Load(ctx, agentPlanStepsByPlanIDKey, key)
}

func (f *AgentPlanStepsByPlanIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
