package managedplanresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedplanservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/shared/i18n"
)

func planToModel(ctx context.Context, plan *managedplanservice.Plan) *gqlmodel.ManagedPlan {
	out := &gqlmodel.ManagedPlan{
		Managed:         plan.Managed,
		FinalBackupDays: plan.BackupDays,
		Usage:           usageToModel(ctx, plan.Usage),
		Base:            featuresToModel(ctx, plan.View.Base),
		Included:        addOnFeaturesToModel(ctx, plan.View.Included),
		Available:       addOnFeaturesToModel(ctx, plan.View.Available),
		Changes:         make([]*gqlmodel.ManagedPlanChange, 0, len(plan.Changes)),
	}
	if plan.ManageURL != "" {
		manageURL := plan.ManageURL
		out.ManageURL = &manageURL
	}
	if plan.EndsAt > 0 {
		endsAt := int(plan.EndsAt)
		out.EndsAt = &endsAt
	}
	byKey := addOnFeaturesByKey()
	for _, entry := range plan.Changes {
		out.Changes = append(out.Changes, &gqlmodel.ManagedPlanChange{
			ChangedAt: int(entry.ChangedAt),
			Starting:  entry.Starting,
			Left:      addOnsToModel(ctx, entry.Left, byKey),
			Joined:    addOnsToModel(ctx, entry.Joined, byKey),
			Ending:    endingsToModel(ctx, entry.Ending, byKey),
			Kept:      addOnsToModel(ctx, entry.Kept, byKey),
		})
	}
	return out
}

func usageToModel(ctx context.Context, usage *managedusage.Usage) *gqlmodel.ManagedPlanUsage {
	if usage == nil {
		return nil
	}
	out := &gqlmodel.ManagedPlanUsage{
		PeriodStart: int(usage.PeriodStart),
		PeriodEnd:   int(usage.PeriodEnd),
		Meters:      make([]*gqlmodel.ManagedPlanMeter, 0, len(usage.Meters)),
	}
	for _, m := range usage.Meters {
		out.Meters = append(out.Meters, &gqlmodel.ManagedPlanMeter{
			Key:      string(m.Meter.Key),
			Name:     i18n.T(ctx, m.Meter.Label),
			Unit:     unitToModel(m.Unit),
			Used:     int(m.Used),
			Included: int(m.Included),
			HardCap:  m.HardCap,
			Stopped:  m.Stopped,
		})
	}
	return out
}

func unitToModel(unit managedusage.Unit) gqlmodel.ManagedPlanMeterUnit {
	if unit == managedusage.UnitBytes {
		return gqlmodel.ManagedPlanMeterUnitBytes
	}
	return gqlmodel.ManagedPlanMeterUnitCount
}

func featuresToModel(
	ctx context.Context,
	features []platformcatalog.Feature,
) []*gqlmodel.ManagedPlanFeature {
	out := make([]*gqlmodel.ManagedPlanFeature, 0, len(features))
	for _, feature := range features {
		out = append(out, &gqlmodel.ManagedPlanFeature{
			Key:  string(feature.Key),
			Name: i18n.T(ctx, feature.Name),
		})
	}
	return out
}

func addOnFeaturesToModel(
	ctx context.Context,
	entries []managedfeatures.AddOnFeatures,
) []*gqlmodel.ManagedPlanAddOn {
	out := make([]*gqlmodel.ManagedPlanAddOn, 0, len(entries))
	for _, entry := range entries {
		addOn := &gqlmodel.ManagedPlanAddOn{
			Key:      string(entry.AddOn.Key),
			Name:     i18n.T(ctx, entry.AddOn.Name),
			Features: featuresToModel(ctx, entry.Features),
		}
		if entry.EndsAt > 0 {
			endsAt := int(entry.EndsAt)
			addOn.EndsAt = &endsAt
		}
		out = append(out, addOn)
	}
	return out
}

func addOnsToModel(
	ctx context.Context,
	addOns []platformcatalog.AddOn,
	byKey map[platformcatalog.AddOnKey][]platformcatalog.Feature,
) []*gqlmodel.ManagedPlanAddOn {
	out := make([]*gqlmodel.ManagedPlanAddOn, 0, len(addOns))
	for _, addOn := range addOns {
		out = append(out, addOnToModel(ctx, addOn, byKey))
	}
	return out
}

func endingsToModel(
	ctx context.Context,
	endings []managedfeatures.AddOnEndingEntry,
	byKey map[platformcatalog.AddOnKey][]platformcatalog.Feature,
) []*gqlmodel.ManagedPlanAddOnEnding {
	out := make([]*gqlmodel.ManagedPlanAddOnEnding, 0, len(endings))
	for _, ending := range endings {
		out = append(out, &gqlmodel.ManagedPlanAddOnEnding{
			AddOn:  addOnToModel(ctx, ending.AddOn, byKey),
			EndsAt: int(ending.EndsAt),
		})
	}
	return out
}

func addOnToModel(
	ctx context.Context,
	addOn platformcatalog.AddOn,
	byKey map[platformcatalog.AddOnKey][]platformcatalog.Feature,
) *gqlmodel.ManagedPlanAddOn {
	return &gqlmodel.ManagedPlanAddOn{
		Key:      string(addOn.Key),
		Name:     i18n.T(ctx, addOn.Name),
		Features: featuresToModel(ctx, byKey[addOn.Key]),
	}
}

func addOnFeaturesByKey() map[platformcatalog.AddOnKey][]platformcatalog.Feature {
	out := make(map[platformcatalog.AddOnKey][]platformcatalog.Feature)
	for _, feature := range platformcatalog.Features() {
		out[feature.AddOn] = append(out[feature.AddOn], feature)
	}
	return out
}
