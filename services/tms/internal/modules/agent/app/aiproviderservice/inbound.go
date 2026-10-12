package aiproviderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

type InboundReadinessParams struct {
	fx.In

	Repo agentports.AIProviderRepository
}

type InboundReadiness struct {
	repo agentports.AIProviderRepository
}

func NewInboundReadiness(p InboundReadinessParams) triageports.InboundModelAvailability {
	return &InboundReadiness{repo: p.Repo}
}

func (r *InboundReadiness) ReadsInboundMail(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	providers, err := r.repo.ListForTask(ctx, agentports.ListAIProvidersForTaskRequest{
		TenantInfo: tenantInfo,
		Task:       aiprovider.TaskInboundClassification,
	})
	if err != nil {
		return false, err
	}

	return aiprovider.RouteFor(providers, aiprovider.TaskInboundClassification) != nil, nil
}
