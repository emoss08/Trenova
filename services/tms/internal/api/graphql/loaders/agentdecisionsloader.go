package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type decisionsByProposalsLister interface {
	ListByProposals(
		ctx context.Context,
		req agentports.ListAgentDecisionsByProposalsRequest,
	) ([]*agent.AgentDecision, error)
}

type AgentDecisionsByProposalIDLoaderFactoryParams struct {
	fx.In

	Decisions agentports.AgentDecisionRepository
}

// AgentDecisionsByProposalIDLoaderFactory reads the decisions behind a page
// of proposals in one query, so a table showing what each approver changed
// does not ask once per row.
type AgentDecisionsByProposalIDLoaderFactory struct {
	decisions decisionsByProposalsLister
}

func NewAgentDecisionsByProposalIDLoaderFactory(
	p AgentDecisionsByProposalIDLoaderFactoryParams,
) *AgentDecisionsByProposalIDLoaderFactory {
	return &AgentDecisionsByProposalIDLoaderFactory{decisions: p.Decisions}
}

func (f *AgentDecisionsByProposalIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*agent.AgentDecision] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *AgentDecisionsByProposalIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*agent.AgentDecision] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*agent.AgentDecision, error) {
			decisions, err := f.decisions.ListByProposals(
				ctx,
				agentports.ListAgentDecisionsByProposalsRequest{
					ProposalIDs: ids,
					TenantInfo:  tenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}

			groups := make(map[pulid.ID][]*agent.AgentDecision, len(ids))
			for _, decision := range decisions {
				if decision == nil || decision.ProposalID == nil {
					continue
				}
				groups[*decision.ProposalID] = append(groups[*decision.ProposalID], decision)
			}

			return groups, nil
		},
	)
}
