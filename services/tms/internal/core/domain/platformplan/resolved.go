package platformplan

import (
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/shared/pulid"
)

type ResolvedPlan struct {
	Plan           *Plan                      `json:"plan"`
	Origin         Origin                     `json:"origin"`
	OrganizationID pulid.ID                   `json:"organizationId"`
	BusinessUnitID pulid.ID                   `json:"businessUnitId"`
	Subscription   *subscription.Subscription `json:"subscription,omitempty"`
	Status         subscription.Status        `json:"status,omitempty"`
	ResolvedAt     int64                      `json:"resolvedAt"`
}

func NewUnmanaged(
	plan *Plan,
	origin Origin,
	organizationID, businessUnitID pulid.ID,
	resolvedAt int64,
) *ResolvedPlan {
	return &ResolvedPlan{
		Plan:           plan,
		Origin:         origin,
		OrganizationID: organizationID,
		BusinessUnitID: businessUnitID,
		ResolvedAt:     resolvedAt,
	}
}

func NewManaged(plan *Plan, sub *subscription.Subscription, resolvedAt int64) *ResolvedPlan {
	return &ResolvedPlan{
		Plan:           plan,
		Origin:         OriginSubscription,
		OrganizationID: sub.OrganizationID,
		BusinessUnitID: sub.BusinessUnitID,
		Subscription:   sub,
		Status:         sub.EffectiveStatus(resolvedAt),
		ResolvedAt:     resolvedAt,
	}
}

func (r *ResolvedPlan) IsManaged() bool {
	return r != nil && r.Origin == OriginSubscription && r.Subscription != nil
}

func (r *ResolvedPlan) Key() PlanKey {
	if r == nil || r.Plan == nil {
		return PlanKeyUnlimited
	}

	return r.Plan.Key
}

func (r *ResolvedPlan) Limit(meter platformcatalog.MeterKey) (Limit, bool) {
	if r == nil {
		return Limit{}, false
	}

	return r.Plan.Limit(meter)
}

func (r *ResolvedPlan) Allows(capability Capability) bool {
	if r == nil {
		return true
	}

	return r.Plan.Allows(capability)
}

func (r *ResolvedPlan) AllowsWrites() bool {
	if !r.IsManaged() {
		return true
	}

	return r.Status.AllowsWrites()
}

func (r *ResolvedPlan) AllowsLogin() bool {
	if !r.IsManaged() {
		return true
	}

	return r.Status.AllowsLogin()
}
