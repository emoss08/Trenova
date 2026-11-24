package boardregistry_test

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingboard"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type reach struct {
	level  permission.FieldSensitivity
	groups []protectedfield.Group
}

func reachOf(registry *permissioncatalog.Registry, fields []boardcatalog.Field) reach {
	out := reach{level: permission.SensitivityPublic}
	for _, field := range fields {
		level := registry.GetFieldSensitivity(field.Resource.String(), field.Name)
		if level.Level() > out.level.Level() {
			out.level = level
		}
		if group, ok := protectedfield.GroupAt(field.Resource, []string{field.Name}, nil); ok {
			out.groups = append(out.groups, group)
		}
	}

	return out
}

func (r reach) covers(other reach) bool {
	if r.level.Level() < other.level.Level() {
		return false
	}
	for _, group := range other.groups {
		if !slices.Contains(r.groups, group) {
			return false
		}
	}

	return true
}

func namedReach(
	registry *permissioncatalog.Registry,
	resources []permission.Resource,
	names []string,
) reach {
	fields := make([]boardcatalog.Field, 0, len(resources)*len(names))
	for _, resource := range resources {
		ranked := map[string]permission.FieldSensitivity{}
		if definition, ok := registry.Get(resource.String()); ok {
			ranked = definition.FieldSensitivities
		}
		for _, name := range names {
			_, isRanked := ranked[name]
			_, grouped := protectedfield.GroupAt(resource, []string{name}, nil)
			if isRanked || grouped {
				fields = append(fields, boardcatalog.Field{Resource: resource, Name: name})
			}
		}
	}

	return reachOf(registry, fields)
}

func assertKnown(
	t *testing.T,
	registry *permissioncatalog.Registry,
	key string,
	fields []boardcatalog.Field,
) {
	t.Helper()
	for _, field := range fields {
		_, ok := registry.Get(field.Resource.String())
		assert.True(t, ok, "%s reads %s, which the permission registry does not know", key,
			field.Resource)
		assert.NotEmpty(t, field.Name, key)
	}
}

func TestEveryMeasureDeclaresTheFieldsItReads(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	for _, measure := range boardregistry.Catalog().Measures() {
		key := string(measure.Kind) + ":" + measure.Name
		require.NotEmpty(t, measure.Reads,
			"%s must list the fields it reads, so the registry decides who sees it", key)
		assertKnown(t, registry, key, measure.Reads)
		if measure.Place != nil {
			require.NotEmpty(t, measure.Place.Reads,
				"%s's place reads beyond the measure, so it lists what it reads", key)
			assertKnown(t, registry, key+" place", measure.Place.Reads)
		}
		named := namedReach(registry, []permission.Resource{measure.Kind.Resource()},
			append([]string{measure.Name}, measure.Aliases...))
		assert.True(t, reachOf(registry, measure.Reads).covers(named),
			"%s is named for a field the registry ranks %s or protects as %v, so it must "+
				"declare that field", key, named.level, named.groups)
	}
}

func TestEveryMetricAndMoneyFigureDeclaresTheFieldsItReads(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	for _, metric := range boardregistry.Catalog().Metrics() {
		require.NotEmpty(t, metric.Reads,
			"%s must list the fields it reads, so the registry decides who sees it", metric.Name)
		assertKnown(t, registry, metric.Name, metric.Reads)
		resources := slices.Clone(metric.Requires)
		if metric.For != "" {
			resources = append(resources, metric.For.Resource())
		}
		for _, figure := range metric.Figures {
			key := metric.Name + "." + figure.Name
			assertKnown(t, registry, key, figure.Reads)
			if figure.Value == boardcatalog.ValueMoney {
				assert.NotEmpty(t, figure.Reads, "%s is money, so it lists the field it reads", key)
			}
			read := reachOf(registry, slices.Concat(metric.Reads, figure.Reads))
			named := namedReach(registry, resources, []string{figure.Name})
			assert.True(t, read.covers(named),
				"%s is named for a field the registry ranks %s or protects as %v, so it must "+
					"declare that field", key, named.level, named.groups)
		}
	}
}

func TestAJournalEntrysAmountIsReadAsTheRegistryRanksItsTotals(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	amount, ok := boardregistry.Catalog().MeasureOf(recordkind.Kind(permission.ResourceJournalEntry), "amount")
	require.True(t, ok)
	assert.Equal(t, permission.SensitivityRestricted, reachOf(registry, amount.Reads).level)

	closing, ok := boardregistry.Catalog().MetricNamed(accountingboard.MetricPeriodClose)
	require.True(t, ok)
	waiting, ok := closing.Figure("pendingAmount")
	require.True(t, ok)
	assert.Equal(t, permission.SensitivityRestricted, reachOf(registry, waiting.Reads).level)
	blocking, ok := closing.Figure("blocking")
	require.True(t, ok)
	assert.Equal(t, permission.SensitivityInternal,
		reachOf(registry, slices.Concat(closing.Reads, blocking.Reads)).level,
		"a count of checks is not the money waiting")

	pay, ok := boardregistry.Catalog().MeasureOf(recordkind.DetentionOccurrence, "driverPay")
	require.True(t, ok)
	assert.Equal(
		t,
		[]protectedfield.Group{protectedfield.Margins},
		reachOf(registry, pay.Reads).groups,
	)
}
