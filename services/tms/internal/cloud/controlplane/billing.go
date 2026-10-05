package controlplane

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

type CloudBillingProviderParams struct {
	fx.In

	Client Client
}

type CloudBillingProvider struct {
	client Client
}

func NewCloudBillingProvider(p CloudBillingProviderParams) *CloudBillingProvider {
	return &CloudBillingProvider{
		client: p.Client,
	}
}

func (p *CloudBillingProvider) GetBillingSummary(
	ctx context.Context,
	req *services.BillingSummaryRequest,
) (*services.BillingSummaryResult, error) {
	return p.client.GetBillingSummary(ctx, req)
}
