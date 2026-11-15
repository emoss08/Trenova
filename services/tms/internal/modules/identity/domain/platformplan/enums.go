package platformplan

import "github.com/emoss08/trenova/pkg/errortypes"

type PlanKey string

const (
	PlanKeyFreeDemo  = PlanKey("free_demo")
	PlanKeyUnlimited = PlanKey("unlimited")
	PlanKeyManaged   = PlanKey("managed")
)

func AllPlanKeys() []PlanKey {
	return []PlanKey{PlanKeyFreeDemo, PlanKeyUnlimited}
}

func (k PlanKey) String() string {
	return string(k)
}

func (k PlanKey) IsValid() bool {
	switch k {
	case PlanKeyFreeDemo, PlanKeyUnlimited:
		return true
	default:
		return false
	}
}

type Window string

const (
	WindowLifetime = Window("lifetime")
	WindowMonthly  = Window("monthly")
	WindowPerItem  = Window("per_item")
)

func AllWindows() []Window {
	return []Window{WindowLifetime, WindowMonthly, WindowPerItem}
}

func (w Window) String() string {
	return string(w)
}

func (w Window) IsValid() bool {
	switch w {
	case WindowLifetime, WindowMonthly, WindowPerItem:
		return true
	default:
		return false
	}
}

type Origin string

const (
	OriginSelfHosted   = Origin("self_hosted")
	OriginInternal     = Origin("internal")
	OriginSubscription = Origin("subscription")
	OriginManaged      = Origin("managed")
)

const (
	ReasonPlanRestricted       = errortypes.PlanRestrictionReasonPlan
	ReasonSubscriptionReadOnly = errortypes.PlanRestrictionReasonReadOnly
	ReasonSubscriptionExpired  = errortypes.PlanRestrictionReasonExpired
	ReasonSignupsPaused        = errortypes.PlanRestrictionReasonSignupsPaused
)
