package planservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

type AdmissionRequest struct {
	TenantInfo pagination.TenantInfo
	Write      bool
	APIKey     bool
	ExpiredOK  bool
}

func Admit(ctx context.Context, plans services.PlanService, req *AdmissionRequest) error {
	if plans == nil || !plans.IsCloud() {
		return nil
	}
	if req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		return nil
	}

	resolved, err := plans.Resolve(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID)
	if err != nil {
		return err
	}
	if !resolved.IsManaged() {
		return nil
	}

	planKey := resolved.Key().String()
	if !resolved.AllowsLogin() && !req.ExpiredOK {
		return errortypes.NewPlanRestrictionError("", platformplan.ReasonSubscriptionExpired, planKey)
	}
	if req.APIKey && !resolved.Allows(platformplan.CapabilityAPIKeys) {
		return errortypes.NewPlanRestrictionError(
			platformplan.CapabilityAPIKeys.String(),
			platformplan.ReasonPlanRestricted,
			planKey,
		)
	}
	if req.Write && !resolved.AllowsWrites() {
		reason := platformplan.ReasonSubscriptionReadOnly
		if !resolved.AllowsLogin() {
			reason = platformplan.ReasonSubscriptionExpired
		}

		return errortypes.NewPlanRestrictionError("", reason, planKey)
	}

	return nil
}
