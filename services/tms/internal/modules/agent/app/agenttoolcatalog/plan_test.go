package agenttoolcatalog_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type policedTool interface {
	Name() string
	Policy() agenttoolports.ToolPolicy
}

func toolPolicies(t *testing.T) map[string]agenttoolports.ToolPolicy {
	t.Helper()

	policies := make(map[string]agenttoolports.ToolPolicy)
	for _, tool := range buildTools(t) {
		policed, ok := tool.(policedTool)
		require.Truef(t, ok, "%s declares no policy", tool.Name())
		policies[policed.Name()] = policed.Policy()
	}

	return policies
}

// A tool is offered only when the plan includes what it does, and the plan
// is read from the tool itself: its resource's feature, or a feature its
// policy names. A tool that declares neither could be offered on any plan,
// so it fails here rather than leaking an add-on (agent-tool-conventions.md,
// "A tool names its resource or its feature").
func TestEveryToolNamesItsResourceOrItsFeature(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.Default()
	var problems []string
	for name, policy := range toolPolicies(t) {
		if policy.Feature != "" {
			if _, known := platformcatalog.Lookup(policy.Feature); !known {
				problems = append(problems, name+": names an unknown feature "+
					string(policy.Feature))
			}
			continue
		}
		if feature, known := registry.FeatureOf(policy.Resource.String()); !known ||
			feature == "" {
			problems = append(problems, name+": its resource "+policy.Resource.String()+
				" belongs to no plan feature and it names none")
		}
	}

	slices.Sort(problems)
	require.Emptyf(t, problems, "%d tools declare neither a resource with a feature nor a "+
		"feature:\n%s", len(problems), strings.Join(problems, "\n"))
}

// The forecast is Dispatch Intelligence read under shipment-move access, and
// the shipment drafts and import edits read what Document Intelligence
// extracted: each needs its add-on as well as its resource.
func TestToolsWhoseWorkIsAnAddOnsNameIt(t *testing.T) {
	t.Parallel()

	policies := toolPolicies(t)
	want := map[string]platformcatalog.FeatureKey{
		"dispatch_forecast_demand":  platformcatalog.FeatureDispatchIntelligence,
		"intake_get_draft":          platformcatalog.FeatureDocumentIntelligence,
		"intake_accept_field":       platformcatalog.FeatureDocumentIntelligence,
		"intake_accept_confident":   platformcatalog.FeatureDocumentIntelligence,
		"intake_set_field":          platformcatalog.FeatureDocumentIntelligence,
		"intake_set_required_field": platformcatalog.FeatureDocumentIntelligence,
		"intake_set_stop_location":  platformcatalog.FeatureDocumentIntelligence,
		"intake_set_stop_schedule":  platformcatalog.FeatureDocumentIntelligence,
	}
	for name, feature := range want {
		policy, ok := policies[name]
		require.Truef(t, ok, "%s is not registered", name)
		assert.Equalf(t, feature, policy.Feature, "%s", name)
	}
}
