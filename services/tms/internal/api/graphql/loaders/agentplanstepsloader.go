package loaders

import (
	"context"

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
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *AgentPlanStepsByPlanIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*agent.AgentProposal] {
	return batchGroupFunc(
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
