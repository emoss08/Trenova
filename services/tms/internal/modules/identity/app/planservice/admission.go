package planservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

type AdmissionRequest struct {
	TenantInfo pagination.TenantInfo
	Write      bool
	APIKey     bool
	ExpiredOK  bool
}

func Admit(
	ctx context.Context,
	plans platformplanports.PlanService,
	gate platformcatalogports.FeatureGate,
	req *AdmissionRequest,
) error {
	var resolved *platformplan.ResolvedPlan
	if plans.EnforcesPlans() && !req.TenantInfo.OrgID.IsNil() && !req.TenantInfo.BuID.IsNil() {
		var err error
		resolved, err = plans.Resolve(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID)
		if err != nil {
			return err
		}
	}

	if !resolved.AllowsLogin() && !req.ExpiredOK {
		return errortypes.NewPlanRestrictionError(
			platformplan.ReasonSubscriptionExpired,
			resolved.Key().String(),
		)
	}
	if req.APIKey {
		if err := platformcatalogports.RequireFeature(
			ctx,
			gate,
			platformcatalog.FeatureAPIKeys,
		); err != nil {
			return err
		}
	}
	if req.Write && !resolved.AllowsWrites() {
		reason := platformplan.ReasonSubscriptionReadOnly
		if !resolved.AllowsLogin() {
			reason = platformplan.ReasonSubscriptionExpired
		}

		return errortypes.NewPlanRestrictionError(reason, resolved.Key().String())
	}

	return nil
}
