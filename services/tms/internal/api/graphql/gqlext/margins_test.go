package gqlext

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/schemadiff"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

// marginName is the conservative pattern a field that may carry a margin is
// named by. A field that matches is protected by the margins group, sits on a
// type reached only through fields that are, or is listed below with why it
// carries no margin.
var marginName = regexp.MustCompile(`(?i)margin|profit`)

// unprotectedMarginNames are the fields whose names match marginName and that
// carry no margin, each with why.
var unprotectedMarginNames = map[string]string{
	"DocumentTemplateVersion.marginTop":    "a page margin of a printed document",
	"DocumentTemplateVersion.marginBottom": "a page margin of a printed document",
	"DocumentTemplateVersion.marginLeft":   "a page margin of a printed document",
	"DocumentTemplateVersion.marginRight":  "a page margin of a printed document",
	"ShipmentAnalytics.profitability": "the card: its margin figures are protected " +
		"field by field, and its cost per mile and load counts are not margins",
	"ShipmentProfitabilityAnalytics.hasMargin": "whether any load had a margin to " +
		"average, so the card can tell withheld from empty; not the margin",
}

// marginEnumValues are the enum values named for a margin, each with where a
// reader without the margins group is refused it.
var marginEnumValues = map[string]string{
	"ProtectedFieldGroup.Margins": "the group itself, which grants nothing on its own",
	"ShipmentQuickFilter.LowMargin": "shipment.AuthorizeQuickFilters refuses the filter " +
		"and the board does not count it for a reader without the margins group",
}

func TestEveryMarginFieldIsProtectedByTheMarginsGroup(t *testing.T) {
	t.Parallel()

	schema, err := schemadiff.LoadDir("../schema")
	require.NoError(t, err)
	protected, err := ProtectedFields(schema)
	require.NoError(t, err)
	covered := typesReachedOnlyThroughMargins(schema, protected)

	var missing []string
	matched := make(map[string]struct{}, len(unprotectedMarginNames))
	for _, name := range sortedTypeNames(schema) {
		def := schema.Types[name]
		if def.BuiltIn || (def.Kind != ast.Object && def.Kind != ast.Interface) {
			continue
		}
		for _, field := range def.Fields {
			if strings.HasPrefix(field.Name, "__") || !marginName.MatchString(field.Name) {
				continue
			}
			key := name + "." + field.Name
			if _, listed := unprotectedMarginNames[key]; listed {
				matched[key] = struct{}{}
				continue
			}
			if group, ok := protected.Group(name, field.Name); ok &&
				group == permission.FieldGroupMargins {
				continue
			}
			if _, ok := covered[name]; ok {
				continue
			}
			missing = append(missing, key)
		}
	}
	require.Empty(t, missing,
		"\nthese fields are named for a margin but nothing withholds them from a reader "+
			"without the margins group: mark each @protected(group: Margins) (make it "+
			"nullable), protect the field that reaches its type, or list it in "+
			"unprotectedMarginNames with why it carries no margin:\n"+strings.Join(missing, "\n"))

	for key := range unprotectedMarginNames {
		_, ok := matched[key]
		assert.Truef(t, ok, "%s is listed in unprotectedMarginNames but no such field "+
			"matches; remove it", key)
	}
}

func TestEveryMarginEnumValueSaysWhereItIsRefused(t *testing.T) {
	t.Parallel()

	schema, err := schemadiff.LoadDir("../schema")
	require.NoError(t, err)

	var missing []string
	for _, name := range sortedTypeNames(schema) {
		def := schema.Types[name]
		if def.BuiltIn || def.Kind != ast.Enum {
			continue
		}
		for _, value := range def.EnumValues {
			key := name + "." + value.Name
			if !marginName.MatchString(value.Name) {
				continue
			}
			if _, ok := marginEnumValues[key]; !ok {
				missing = append(missing, key)
			}
		}
	}
	require.Empty(t, missing,
		"\nthese enum values select or rank by a margin; refuse them to a reader without "+
			"the margins group and list each in marginEnumValues with where:\n"+
			strings.Join(missing, "\n"))
}

func sortedTypeNames(schema *ast.Schema) []string {
	names := make([]string, 0, len(schema.Types))
	for name := range schema.Types {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// typesReachedOnlyThroughMargins is every object type that some field
// returns and every field returning it is protected by the margins group or
// sits on such a type, so no reader without the group reaches it.
func typesReachedOnlyThroughMargins(
	schema *ast.Schema,
	protected ProtectedFieldMap,
) map[string]struct{} {
	type reach struct {
		object, field string
	}
	reachedBy := make(map[string][]reach)
	for name, def := range schema.Types {
		if def.BuiltIn || (def.Kind != ast.Object && def.Kind != ast.Interface) {
			continue
		}
		for _, field := range def.Fields {
			target := field.Type.Name()
			reachedBy[target] = append(reachedBy[target], reach{object: name, field: field.Name})
		}
	}

	covered := make(map[string]struct{})
	for changed := true; changed; {
		changed = false
		for target, reaches := range reachedBy {
			if _, done := covered[target]; done {
				continue
			}
			all := len(reaches) > 0
			for _, r := range reaches {
				group, ok := protected.Group(r.object, r.field)
				_, carrierCovered := covered[r.object]
				if (!ok || group != permission.FieldGroupMargins) && !carrierCovered {
					all = false
					break
				}
			}
			if all {
				covered[target] = struct{}{}
				changed = true
			}
		}
	}

	return covered
}

func TestMarginPayloadsAreReachedOnlyThroughProtectedFields(t *testing.T) {
	t.Parallel()

	schema, err := schemadiff.LoadDir("../schema")
	require.NoError(t, err)
	protected, err := ProtectedFields(schema)
	require.NoError(t, err)
	covered := typesReachedOnlyThroughMargins(schema, protected)

	for _, name := range []string{"ShipmentProfitability", "ShipmentProfitabilityEstimate"} {
		_, ok := covered[name]
		assert.Truef(t, ok, "%s carries a shipment's margin and must be reached only "+
			"through fields protected by the margins group", name)
	}
}

func TestFieldGuardWithholdsEveryMarginFigureWithoutTheMarginsGroup(t *testing.T) {
	t.Parallel()

	cases := []struct{ object, field string }{
		{"Query", "shipmentProfitability"},
		{"Shipment", "profitabilityEstimate"},
		{"CostingControl", "targetMarginPercent"},
		{"FleetCostSummary", "avgMarginPercent"},
		{"FleetCostSummary", "totalRevenue"},
		{"CapacityMatch", "marginPercent"},
		{"CapacityMatch", "quote"},
		{"TenderOffer", "counterMarginPercent"},
		{"DetentionOccurrence", "netMargin"},
		{"DetentionOccurrence", "driverPayAmount"},
		{"ShipmentProfitabilityAnalytics", "unprofitableCount"},
	}

	withoutMargins := newFieldGuard(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceProtectedMargins.String()},
	})
	withMargins := newFieldGuard(t, &mocks.DenyResourcesPermissionEngine{})
	for _, tc := range cases {
		name := fmt.Sprintf("%s.%s", tc.object, tc.field)

		result, resolved := resolveThrough(
			t, withoutMargins, operationContext(t, withoutMargins), tc.object, tc.field,
		)
		assert.Nilf(t, result, "%s reached a reader without the margins group", name)
		assert.Falsef(t, resolved, "%s was resolved for a reader without the group", name)

		result, resolved = resolveThrough(
			t, withMargins, operationContext(t, withMargins), tc.object, tc.field,
		)
		assert.Equalf(t, "stored value", result, "%s was withheld from a margins reader", name)
		assert.Truef(t, resolved, "%s was not resolved for a margins reader", name)
	}
}

func TestFieldGuardHoldsAnAgentReadToItsDataAccessForAMarginFigure(t *testing.T) {
	t.Parallel()

	guard := newFieldGuard(t, &mocks.DenyResourcesPermissionEngine{})

	ctx, _ := agentRead(t, guard, permission.SensitivityInternal)
	result, resolved := resolveThrough(t, guard, ctx, "CapacityMatch", "marginPercent")
	assert.Nil(t, result, "an agent whose data access stops at Internal reads no margin")
	assert.False(t, resolved)

	ctx, _ = agentRead(t, guard, permission.SensitivityRestricted)
	result, resolved = resolveThrough(t, guard, ctx, "CapacityMatch", "marginPercent")
	assert.Equal(t, "stored value", result,
		"an agent reading for a person with the group, at Restricted, reads the margin")
	assert.True(t, resolved)

	denied := newFieldGuard(t, &mocks.DenyResourcesPermissionEngine{
		Denied: []string{permission.ResourceProtectedMargins.String()},
	})
	ctx, _ = agentRead(t, denied, permission.SensitivityRestricted)
	result, resolved = resolveThrough(t, denied, ctx, "CapacityMatch", "marginPercent")
	assert.Nil(t, result, "an agent never reads a margin its person may not see")
	assert.False(t, resolved)
}
