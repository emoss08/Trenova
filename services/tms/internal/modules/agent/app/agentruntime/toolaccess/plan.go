package toolaccess

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/planwording"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"go.uber.org/zap"
)

// MaxPlanFeatureNames bounds the features one add-on's line names in the
// prompt; the rest are counted. With at most one line per add-on in the
// catalog, what a turn is told about its plan stays a few hundred bytes.
const MaxPlanFeatureNames = 4

// toolPolicy is the policy of the registered tool behind a name.
func (s *Access) toolPolicy(name string) (agenttoolports.ToolPolicy, bool) {
	if tool, ok := s.queryTools.Get(name); ok {
		return tool.Policy(), true
	}
	if tool, ok := s.actionTools.Get(name); ok {
		return tool.Policy(), true
	}

	return agenttoolports.ToolPolicy{}, false
}

// ToolFeatures are the plan features a tool needs: its resource's, and the
// one its policy names when its work is an add-on's beyond its resource.
func (s *Access) ToolFeatures(name string) []platformcatalog.FeatureKey {
	policy, ok := s.toolPolicy(name)
	if !ok {
		return nil
	}
	features := make([]platformcatalog.FeatureKey, 0, 2)
	if feature, known := s.registry.FeatureOf(policy.Resource.String()); known {
		features = append(features, feature)
	}
	if policy.Feature != "" && policy.Feature != firstOr(features) {
		features = append(features, policy.Feature)
	}

	return features
}

func firstOr(features []platformcatalog.FeatureKey) platformcatalog.FeatureKey {
	if len(features) == 0 {
		return ""
	}

	return features[0]
}

// declaredFeatureDenial is the plan's denial of the feature a tool's policy
// names, when the plan leaves it out. The resource's own feature is the
// permission engine's to refuse.
func (s *Access) declaredFeatureDenial(
	ctx context.Context,
	name string,
) (permissionports.PlanDenial, bool, error) {
	policy, ok := s.toolPolicy(name)
	if !ok || policy.Feature == "" {
		return permissionports.PlanDenial{}, false, nil
	}

	return s.plan.Feature(ctx, policy.Feature)
}

// onPlanOnly drops the tools whose declared feature the plan leaves out. A
// plan that cannot be read withholds them.
func (s *Access) onPlanOnly(ctx context.Context, names []string) []string {
	kept := make([]string, 0, len(names))
	for _, name := range names {
		_, off, err := s.declaredFeatureDenial(ctx, name)
		if err != nil {
			s.logger.Warn("could not read whether the plan includes a tool; withholding it",
				zap.String("tool", name), zap.Error(err))
			continue
		}
		if !off {
			kept = append(kept, name)
		}
	}

	return kept
}

// PlanRefusal refuses a call to a tool whose declared feature the plan
// leaves out, naming the add-on. Dispatch asks it before the permission
// check, which refuses a tool whose resource is off the plan the same way.
func (s *Access) PlanRefusal(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	name string,
) (toolcall.Result, bool) {
	denial, off, err := s.declaredFeatureDenial(ctx, name)
	if err != nil {
		return toolcall.Refused(aitrace.OutcomeFailed, "its plan could not be checked",
			"Tool %q could not be authorized. Try again later.", name), true
	}
	if !off {
		return toolcall.Result{}, false
	}

	return toolcall.Refused(
		aitrace.OutcomeDenied,
		planwording.Reason(&denial),
		"%s",
		planwording.Refusal(
			fmt.Sprintf("Tool %q", name),
			&denial,
			req.Actor != nil && req.Actor.IsAgent(),
		),
	), true
}

// readPlan reads, once for the turn, every feature the plan leaves out. A
// plan that cannot be read tells the turn nothing about it; the tools are
// still withheld by the checks that read it per tool.
func (s *Access) readPlan(ctx context.Context, set *ToolSet) {
	unavailable, err := s.plan.Unavailable(ctx)
	if err != nil {
		s.logger.Warn("could not read what the plan leaves out", zap.Error(err))
		return
	}
	set.planOff(unavailable)
}

// planOff records what the plan leaves out on the set: by feature for the
// refusals, and by add-on for the prompt.
func (t *ToolSet) planOff(unavailable []permissionports.PlanDenial) {
	if len(unavailable) == 0 {
		return
	}
	t.offPlan = make(map[platformcatalog.FeatureKey]permissionports.PlanDenial, len(unavailable))
	for _, denial := range unavailable {
		t.offPlan[denial.Feature] = denial
	}
	t.NotOnPlan = PlanAddOns(unavailable)
}

// offPlanDenials is what the plan leaves out, in catalog order, for the
// set's durable state.
func (t *ToolSet) offPlanDenials() []permissionports.PlanDenial {
	if len(t.offPlan) == 0 {
		return nil
	}
	denials := make([]permissionports.PlanDenial, 0, len(t.offPlan))
	for _, feature := range platformcatalog.Features() {
		if denial, off := t.offPlan[feature.Key]; off {
			denials = append(denials, denial)
		}
	}

	return denials
}

// planDenial is the plan's denial of a tool, read from what the turn
// already knows of the plan: its resource's feature first, then the one its
// policy names.
func (s *Access) planDenial(set *ToolSet, name string) (*permissionports.PlanDenial, bool) {
	for _, feature := range s.ToolFeatures(name) {
		if denial, off := set.offPlan[feature]; off {
			return &denial, true
		}
	}

	return nil, false
}

// PlanAddOns groups what the plan leaves out by add-on, in catalog order:
// the add-on's name and the names of its features that are off, at most
// MaxPlanFeatureNames of them with the rest counted. A feature of no add-on
// is left out, since there is nothing to buy.
func PlanAddOns(unavailable []permissionports.PlanDenial) []agentturn.PlanAddOn {
	byAddOn := make(map[platformcatalog.AddOnKey]*agentturn.PlanAddOn, len(unavailable))
	for _, denial := range unavailable {
		if denial.AddOnName == "" {
			continue
		}
		addOn, ok := byAddOn[denial.AddOn]
		if !ok {
			addOn = &agentturn.PlanAddOn{Name: denial.AddOnName}
			byAddOn[denial.AddOn] = addOn
		}
		name := strings.TrimSpace(denial.FeatureName)
		if name == "" {
			name = string(denial.Feature)
		}
		if len(addOn.Features) < MaxPlanFeatureNames {
			addOn.Features = append(addOn.Features, name)
		} else {
			addOn.MoreFeatures++
		}
	}

	out := make([]agentturn.PlanAddOn, 0, len(byAddOn))
	for _, catalogAddOn := range platformcatalog.AddOns() {
		if addOn, ok := byAddOn[catalogAddOn.Key]; ok {
			out = append(out, *addOn)
		}
	}

	return out
}
