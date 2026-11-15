package storagequotaservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

type Status struct {
	Managed  bool  `json:"managed"`
	Used     int64 `json:"used"`
	Included int64 `json:"included"`
	Overage  bool  `json:"overage"`
	Warning  bool  `json:"warning"`
	Stopped  bool  `json:"stopped"`
}

type Params struct {
	fx.In

	Quota platformplanports.QuotaGuard
}

type Service struct {
	quota platformplanports.QuotaGuard
}

func New(p Params) *Service {
	return &Service{quota: quotaservice.OrUnlimited(p.Quota)}
}

func (s *Service) Status(ctx context.Context, tenant pagination.TenantInfo) (*Status, error) {
	summary, err := s.quota.Usage(ctx, tenant)
	if err != nil {
		return nil, err
	}
	if summary.Plan != platformplan.PlanKeyManaged {
		return &Status{}, nil
	}
	for i := range summary.Meters {
		m := &summary.Meters[i]
		if m.Meter != platformcatalog.MeterDocumentStorageBytes {
			continue
		}
		return statusOf(m, summary.Overage), nil
	}
	return &Status{}, nil
}

func statusOf(m *platformplanports.QuotaMeterUsage, overage bool) *Status {
	full := (m.HardCap || !overage) && m.Used >= m.Limit
	return &Status{
		Managed:  true,
		Used:     m.Used,
		Included: m.Limit,
		Overage:  overage && !m.HardCap,
		Warning: managedusage.NearStorageAllowance(managedusage.Meter{
			Quantity:  m.Used,
			Allowance: m.Limit,
		}),
		Stopped: m.Stopped || full,
	}
}
