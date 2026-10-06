package planservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type UnlimitedPlanService struct {
	plan *platformplan.Plan
}

var _ services.PlanService = (*UnlimitedPlanService)(nil)

func NewUnlimited() *UnlimitedPlanService {
	return &UnlimitedPlanService{plan: platformplan.Unlimited()}
}

func (s *UnlimitedPlanService) EnforcesPlans() bool {
	return false
}

func (s *UnlimitedPlanService) Resolve(
	_ context.Context,
	orgID, buID pulid.ID,
) (*platformplan.ResolvedPlan, error) {
	return platformplan.NewUnmanaged(
		s.plan,
		platformplan.OriginSelfHosted,
		orgID,
		buID,
		time.Now().Unix(),
	), nil
}

func (s *UnlimitedPlanService) RequireCapability(
	context.Context,
	pagination.TenantInfo,
	platformplan.Capability,
) error {
	return nil
}

func (s *UnlimitedPlanService) RequireWritable(context.Context, pagination.TenantInfo) error {
	return nil
}

func (s *UnlimitedPlanService) Invalidate(pulid.ID) {}
