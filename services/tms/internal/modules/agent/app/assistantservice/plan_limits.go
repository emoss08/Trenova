package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) assertWithinPlan(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	if err := quotaservice.Preflight(ctx, s.quota, &platformplanports.QuotaRequest{
		TenantInfo: tenantInfo,
		Meter:      platformcatalog.MeterAIAssistantMessages,
		Quantity:   0,
	}); err != nil {
		return err
	}

	return quotaservice.Preflight(ctx, s.quota, &platformplanports.QuotaRequest{
		TenantInfo: tenantInfo,
		Meter:      platformcatalog.MeterAISpendCents,
		Quantity:   1,
	})
}
