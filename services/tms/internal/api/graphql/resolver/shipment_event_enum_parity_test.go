package resolver

import (
	"sort"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentEventTypeParity_DomainMatchesSchema(t *testing.T) {
	t.Parallel()

	domain := make(map[string]struct{}, len(shipmentevent.AllTypes))
	for _, v := range shipmentevent.AllTypes {
		domain[string(v)] = struct{}{}
	}
	require.Len(t, domain, len(shipmentevent.AllTypes), "shipmentevent.AllTypes contains duplicates")

	schema := make(map[string]struct{}, len(gqlmodel.AllShipmentEventType))
	for _, v := range gqlmodel.AllShipmentEventType {
		schema[string(v)] = struct{}{}
	}
	require.Len(
		t,
		schema,
		len(gqlmodel.AllShipmentEventType),
		"gqlmodel.AllShipmentEventType contains duplicates",
	)

	assert.Empty(
		t,
		missingFrom(domain, schema),
		"shipmentevent.Type values missing from enum ShipmentEventType in shipment.graphqls",
	)
	assert.Empty(
		t,
		missingFrom(schema, domain),
		"ShipmentEventType schema values missing from shipmentevent.AllTypes",
	)
}

func TestShipmentEventTypeParity_AllTypesAreValid(t *testing.T) {
	t.Parallel()

	for _, v := range shipmentevent.AllTypes {
		assert.Truef(t, v.IsValid(), "shipmentevent.Type(%q) reported invalid", string(v))
	}
	assert.False(t, shipmentevent.Type("NotARealShipmentEventType").IsValid())
	assert.False(t, shipmentevent.Type("").IsValid())
}

func missingFrom(want, have map[string]struct{}) []string {
	missing := make([]string, 0, len(want))
	for v := range want {
		if _, ok := have[v]; !ok {
			missing = append(missing, v)
		}
	}
	sort.Strings(missing)
	return missing
}
