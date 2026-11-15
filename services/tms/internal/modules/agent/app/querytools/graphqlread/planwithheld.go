package graphqlread

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

// OutcomeWithheldByPlan is the key a read's result names what the
// organization's plan withheld under, apart from what the reader's access
// withheld: one sentence per add-on, naming its fields.
const OutcomeWithheldByPlan = "withheldByPlan"

// PlanWithheld words what the plan withheld from a read, by paths relative to
// prefix: one sentence per feature, in feature order, saying its fields are
// part of an add-on the organization's plan does not include.
func PlanWithheld(withheld []graphqlreadports.WithheldByPlan, prefix string) []string {
	if len(withheld) == 0 {
		return nil
	}
	byFeature := make(map[platformcatalog.FeatureKey][]string, 2)
	features := make([]platformcatalog.FeatureKey, 0, 2)
	for _, entry := range withheld {
		if _, seen := byFeature[entry.Feature]; !seen {
			features = append(features, entry.Feature)
		}
		path := strings.TrimPrefix(entry.Path, prefix)
		if !slices.Contains(byFeature[entry.Feature], path) {
			byFeature[entry.Feature] = append(byFeature[entry.Feature], path)
		}
	}
	slices.Sort(features)

	out := make([]string, 0, len(features))
	for _, feature := range features {
		out = append(out, planSentence(byFeature[feature], permissionports.NewPlanDenial(feature)))
	}

	return out
}

func planSentence(paths []string, denial permissionports.PlanDenial) string {
	slices.Sort(paths)
	verb, pronoun := "is", "it"
	if len(paths) > 1 {
		verb, pronoun = "are", "them"
	}
	product := denial.FeatureName
	if product == "" {
		product = string(denial.Feature)
	}
	part := product
	if denial.AddOnName != "" {
		part = fmt.Sprintf("the %s add-on (%s)", denial.AddOnName, product)
	}

	return fmt.Sprintf(
		"%s %s part of %s, which this organization's plan does not include. Do not look "+
			"%s up another way; say the plan does not include %s.",
		strings.Join(paths, ", "), verb, part, pronoun, pronoun,
	)
}

// PrefixedPlanWithheld is PlanWithheld for a read made beside a tool's own,
// whose fields the result shows under key.
func PrefixedPlanWithheld(
	withheld []graphqlreadports.WithheldByPlan,
	prefix, key string,
) []string {
	keyed := make([]graphqlreadports.WithheldByPlan, 0, len(withheld))
	for _, entry := range withheld {
		keyed = append(keyed, graphqlreadports.WithheldByPlan{
			Path:    key + "." + strings.TrimPrefix(entry.Path, prefix),
			Feature: entry.Feature,
		})
	}

	return PlanWithheld(keyed, "")
}
