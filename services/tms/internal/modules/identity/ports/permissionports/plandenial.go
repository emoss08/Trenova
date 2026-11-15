package permissionports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const ReasonNotOnPlan = "not_on_plan"

type PlanDenial struct {
	Feature     platformcatalog.FeatureKey `json:"feature"`
	FeatureName string                     `json:"featureName"`
	AddOn       platformcatalog.AddOnKey   `json:"addOn"`
	AddOnName   string                     `json:"addOnName"`
}

func NewPlanDenial(feature platformcatalog.FeatureKey) PlanDenial {
	denial := PlanDenial{Feature: feature}
	if known, ok := platformcatalog.Lookup(feature); ok {
		denial.FeatureName = known.Name
	}
	if addOn, ok := platformcatalog.AddOnFor(feature); ok {
		denial.AddOn = addOn.Key
		denial.AddOnName = addOn.Name
	}

	return denial
}

func (d *PlanDenial) Restriction() *errortypes.PlanRestrictionError {
	return errortypes.NewFeatureRestrictionError(string(d.Feature), d.AddOnName)
}

func DeniedError(result *PermissionCheckResult, denied error) error {
	if result != nil && result.PlanDenial != nil {
		return result.PlanDenial.Restriction()
	}

	return denied
}

// PlanFeatures is how a surface outside the identity module asks what the
// organization's plan leaves out (onplan): one feature, or every feature in
// catalog order. A gate that cannot be read is an error, never included.
type PlanFeatures interface {
	Feature(ctx context.Context, feature platformcatalog.FeatureKey) (PlanDenial, bool, error)
	Unavailable(ctx context.Context) ([]PlanDenial, error)
}
