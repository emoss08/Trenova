package managedplanresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedplanservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

func planToModel(plan *managedplanservice.Plan) *gqlmodel.ManagedPlan {
	out := &gqlmodel.ManagedPlan{
		Managed:   plan.Managed,
		Base:      featuresToModel(plan.View.Base),
		Included:  addOnFeaturesToModel(plan.View.Included),
		Available: addOnFeaturesToModel(plan.View.Available),
		Changes:   make([]*gqlmodel.ManagedPlanChange, 0, len(plan.Changes)),
	}
	byKey := addOnFeaturesByKey()
	for _, entry := range plan.Changes {
		out.Changes = append(out.Changes, &gqlmodel.ManagedPlanChange{
			ChangedAt: int(entry.ChangedAt),
			Starting:  entry.Starting,
			Left:      addOnsToModel(entry.Left, byKey),
			Joined:    addOnsToModel(entry.Joined, byKey),
		})
	}
	return out
}

func featuresToModel(features []platformcatalog.Feature) []*gqlmodel.ManagedPlanFeature {
	out := make([]*gqlmodel.ManagedPlanFeature, 0, len(features))
	for _, feature := range features {
		out = append(out, &gqlmodel.ManagedPlanFeature{
			Key:  string(feature.Key),
			Name: feature.Name,
		})
	}
	return out
}

func addOnFeaturesToModel(entries []managedfeatures.AddOnFeatures) []*gqlmodel.ManagedPlanAddOn {
	out := make([]*gqlmodel.ManagedPlanAddOn, 0, len(entries))
	for _, entry := range entries {
		out = append(out, &gqlmodel.ManagedPlanAddOn{
			Key:      string(entry.AddOn.Key),
			Name:     entry.AddOn.Name,
			Features: featuresToModel(entry.Features),
		})
	}
	return out
}

func addOnsToModel(
	addOns []platformcatalog.AddOn,
	byKey map[platformcatalog.AddOnKey][]platformcatalog.Feature,
) []*gqlmodel.ManagedPlanAddOn {
	out := make([]*gqlmodel.ManagedPlanAddOn, 0, len(addOns))
	for _, addOn := range addOns {
		out = append(out, &gqlmodel.ManagedPlanAddOn{
			Key:      string(addOn.Key),
			Name:     addOn.Name,
			Features: featuresToModel(byKey[addOn.Key]),
		})
	}
	return out
}

func addOnFeaturesByKey() map[platformcatalog.AddOnKey][]platformcatalog.Feature {
	out := make(map[platformcatalog.AddOnKey][]platformcatalog.Feature)
	for _, feature := range platformcatalog.Features() {
		out[feature.AddOn] = append(out[feature.AddOn], feature)
	}
	return out
}
