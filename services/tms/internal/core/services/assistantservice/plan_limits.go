package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/quotaservice"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) assertWithinPlan(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	if err := quotaservice.Preflight(ctx, s.quota, &serviceports.QuotaRequest{
		TenantInfo: tenantInfo,
		Meter:      platformcatalog.MeterAIAssistantMessages,
		Quantity:   0,
	}); err != nil {
		return err
	}

	return quotaservice.Preflight(ctx, s.quota, &serviceports.QuotaRequest{
		TenantInfo: tenantInfo,
		Meter:      platformcatalog.MeterAISpendCents,
		Quantity:   1,
	})
}
