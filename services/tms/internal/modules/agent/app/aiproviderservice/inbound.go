package aiproviderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

type InboundReadinessParams struct {
	fx.In

	Nova llmports.TaskRunner
}

// InboundReadiness reports whether inbound mail is read: Nova reads it, so it
// is read when Nova runs for the tenant.
type InboundReadiness struct {
	nova llmports.TaskRunner
}

func NewInboundReadiness(p InboundReadinessParams) triageports.InboundModelAvailability {
	return &InboundReadiness{nova: p.Nova}
}

func (r *InboundReadiness) ReadsInboundMail(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	return r.nova.Available(ctx, tenantInfo)
}
