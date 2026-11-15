package platformcatalog_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func declaredFeatureKeys(t *testing.T) map[string]platformcatalog.FeatureKey {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "constants.go", nil, 0)
	require.NoError(t, err)

	declared := make(map[string]platformcatalog.FeatureKey)
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range value.Names {
				if !strings.HasPrefix(name.Name, "Feature") {
					continue
				}
				require.Less(t, i, len(value.Values), "%s has no value", name.Name)
				call, ok := value.Values[i].(*ast.CallExpr)
				require.True(t, ok, "%s is not a FeatureKey conversion", name.Name)
				require.Len(t, call.Args, 1, name.Name)
				lit, ok := call.Args[0].(*ast.BasicLit)
				require.True(t, ok, "%s is not a string literal", name.Name)
				unquoted, err := strconv.Unquote(lit.Value)
				require.NoError(t, err, name.Name)
				declared[name.Name] = platformcatalog.FeatureKey(unquoted)
			}
		}
	}
	return declared
}

func TestEveryDeclaredFeatureIsRegistered(t *testing.T) {
	t.Parallel()

	declared := declaredFeatureKeys(t)
	require.Len(t, declared, 40)

	for name, key := range declared {
		feature, ok := platformcatalog.Lookup(key)
		assert.True(t, ok, "%s (%s) is not registered", name, key)
		assert.Equal(t, key, feature.Key, name)
	}

	registered := platformcatalog.Features()
	assert.Len(t, registered, len(declared))
}

func TestNoFeatureIsRegisteredTwice(t *testing.T) {
	t.Parallel()

	seen := make(map[platformcatalog.FeatureKey]bool)
	for _, feature := range platformcatalog.Features() {
		assert.False(t, seen[feature.Key], "%s is registered twice", feature.Key)
		seen[feature.Key] = true
	}
}

func TestEveryFeatureHasANameAndAKnownAddOn(t *testing.T) {
	t.Parallel()

	known := map[platformcatalog.AddOnKey]bool{platformcatalog.AddOnBase: true}
	for _, addOn := range platformcatalog.AddOns() {
		known[addOn.Key] = true
	}

	for _, feature := range platformcatalog.Features() {
		assert.NotEmpty(t, strings.TrimSpace(feature.Name), "%s has no name", feature.Key)
		assert.True(t, known[feature.AddOn], "%s has unknown add-on %q", feature.Key, feature.AddOn)
	}
}

func TestAddOnsAreTheThirteenSKUs(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []platformcatalog.AddOn{
		{Key: "ACC", Name: "Accounting & Cash"},
		{Key: "RTE", Name: "Rating & Revenue Recovery"},
		{Key: "FUL", Name: "Fuel & IFTA"},
		{Key: "DSP", Name: "Dispatch Intelligence"},
		{Key: "VIS", Name: "Visibility & Tracking"},
		{Key: "BRK", Name: "Carrier & Brokerage"},
		{Key: "PAY", Name: "Driver Pay"},
		{Key: "HRS", Name: "HR Suite"},
		{Key: "SAF", Name: "Safety & Compliance"},
		{Key: "DOC", Name: "Document Intelligence"},
		{Key: "AI", Name: "Trenova AI"},
		{Key: "EDI", Name: "EDI & Integrations"},
		{Key: "SEC", Name: "Security & Identity"},
	}, platformcatalog.AddOns())
}

func TestEveryAddOnSellsAFeature(t *testing.T) {
	t.Parallel()

	sold := make(map[platformcatalog.AddOnKey]int)
	for _, feature := range platformcatalog.Features() {
		sold[feature.AddOn]++
	}

	for _, addOn := range platformcatalog.AddOns() {
		assert.Positive(t, sold[addOn.Key], "%s sells no feature", addOn.Key)
	}
}

func TestFeaturesBelongToTheirCatalogAddOn(t *testing.T) {
	t.Parallel()

	expected := map[platformcatalog.AddOnKey][]platformcatalog.FeatureKey{
		platformcatalog.AddOnBase: {
			"tms.core", "tms.dispatch", "tms.fleet", "tms.billing", "workforce.core",
			"document_management", "analytics.workspace", "platform.administration",
			"platform.api_keys", "platform.global_search", "platform.realtime_notifications",
			"integrations.exchange_rates",
		},
		platformcatalog.AddOnAccounting: {"tms.accounting", "tms.factoring"},
		platformcatalog.AddOnRating:     {"tms.rating", "tms.detention"},
		platformcatalog.AddOnFuel:       {"tms.fuel_tax"},
		platformcatalog.AddOnDriverPay:  {"tms.settlement", "tms.driver_portal"},
		platformcatalog.AddOnHR: {
			"workforce.time_off", "workforce.time_tracking", "workforce.talent",
			"workforce.benefits", "workforce.self_service",
		},
		platformcatalog.AddOnSafety:    {"workforce.compliance", "workforce.safety"},
		platformcatalog.AddOnBrokerage: {"tms.carrier_brokerage"},
		platformcatalog.AddOnEDI:       {"integrations.edi"},
		platformcatalog.AddOnVisibility: {
			"platform.table_change_alerts", "integrations.samsara", "integrations.google_maps",
			"tms.customer_tracking",
		},
		platformcatalog.AddOnDispatchIntelligence: {"tms.dispatch_intelligence"},
		platformcatalog.AddOnDocumentIntelligence: {"document_intelligence.ai"},
		platformcatalog.AddOnAI: {
			"ai.desk", "ai.watchtower", "ai.triage", "platform.agent_automation",
			"ai.audit_trail",
		},
		platformcatalog.AddOnSecurity: {"platform.identity"},
	}

	actual := make(map[platformcatalog.AddOnKey][]platformcatalog.FeatureKey)
	for _, feature := range platformcatalog.Features() {
		actual[feature.AddOn] = append(actual[feature.AddOn], feature.Key)
	}

	assert.Equal(t, expected, actual)
}

func TestIsBase(t *testing.T) {
	t.Parallel()

	core, ok := platformcatalog.Lookup(platformcatalog.FeatureCoreTMS)
	require.True(t, ok)
	assert.True(t, core.IsBase())

	accounting, ok := platformcatalog.Lookup(platformcatalog.FeatureAccounting)
	require.True(t, ok)
	assert.False(t, accounting.IsBase())
}

func TestLookupUnknownFeature(t *testing.T) {
	t.Parallel()

	feature, ok := platformcatalog.Lookup("tms.unknown")
	assert.False(t, ok)
	assert.Equal(t, platformcatalog.Feature{}, feature)
}

func TestAddOnForBaseFeatureIsFalse(t *testing.T) {
	t.Parallel()

	addOn, ok := platformcatalog.AddOnFor(platformcatalog.FeatureBilling)
	assert.False(t, ok)
	assert.Equal(t, platformcatalog.AddOn{}, addOn)
}

func TestAddOnForUnknownFeatureIsFalse(t *testing.T) {
	t.Parallel()

	addOn, ok := platformcatalog.AddOnFor("tms.unknown")
	assert.False(t, ok)
	assert.Equal(t, platformcatalog.AddOn{}, addOn)
}

func TestAddOnForAddOnFeature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		feature platformcatalog.FeatureKey
		want    platformcatalog.AddOn
	}{
		{
			feature: platformcatalog.FeatureFactoring,
			want:    platformcatalog.AddOn{Key: "ACC", Name: "Accounting & Cash"},
		},
		{
			feature: platformcatalog.FeatureAgentAutomation,
			want:    platformcatalog.AddOn{Key: "AI", Name: "Trenova AI"},
		},
		{
			feature: platformcatalog.FeatureIdentity,
			want:    platformcatalog.AddOn{Key: "SEC", Name: "Security & Identity"},
		},
		{
			feature: platformcatalog.FeatureTableChangeAlerts,
			want:    platformcatalog.AddOn{Key: "VIS", Name: "Visibility & Tracking"},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.feature), func(t *testing.T) {
			t.Parallel()

			got, ok := platformcatalog.AddOnFor(tt.feature)
			require.True(t, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFeaturesReturnsACopy(t *testing.T) {
	t.Parallel()

	first := platformcatalog.Features()
	original := first[0]
	first[0] = platformcatalog.Feature{Key: "mutated", Name: "mutated", AddOn: "mutated"}

	assert.Equal(t, original, platformcatalog.Features()[0])
}

func TestAddOnsReturnsACopy(t *testing.T) {
	t.Parallel()

	first := platformcatalog.AddOns()
	original := first[0]
	first[0] = platformcatalog.AddOn{Key: "mutated", Name: "mutated"}

	assert.Equal(t, original, platformcatalog.AddOns()[0])
}
