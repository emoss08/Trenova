package recordquerytools_test

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/recordquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type reader struct {
	name   string
	schema map[string]any
	result any
	config *domaintypes.FieldConfiguration
	extras []querytool.Extra
	calls  []*agenttoolports.QueryToolParams
}

func (r *reader) Name() string                      { return r.name }
func (r *reader) Description() string               { return r.name }
func (r *reader) ParamSchema() map[string]any       { return r.schema }
func (r *reader) Policy() agenttoolports.ToolPolicy { return agenttoolports.ToolPolicy{Name: r.name} }
func (r *reader) WhereFields() *domaintypes.FieldConfiguration {
	return r.config
}
func (r *reader) WhereExtras() []querytool.Extra { return r.extras }

func (r *reader) Query(_ context.Context, params *agenttoolports.QueryToolParams) (any, error) {
	r.calls = append(r.calls, params)

	return r.result, nil
}

type readers map[string]agenttoolports.AgentQueryTool

func (r readers) Get(name string) (agenttoolports.AgentQueryTool, bool) {
	tool, ok := r[name]
	return tool, ok
}

func shipmentReaders() (readers, *reader, *reader, *reader) {
	total := 3
	list := &reader{
		name: "shipment_list",
		config: &domaintypes.FieldConfiguration{
			FilterableFields: map[string]bool{
				"status":                                 true,
				"customer.name":                          true,
				"pickupAppointment.scheduledWindowStart": true,
				"temperatureMin":                         true,
			},
			SortableFields: map[string]bool{"status": true},
		},
		extras: []querytool.Extra{{Name: "stops.city"}, {Name: "stops.type"}, {Name: "unassigned"}},
		result: map[string]any{
			"items": []any{
				map[string]any{"id": "shp_1", "proNumber": "PRO-1"},
				map[string]any{"id": "shp_2", "proNumber": "PRO-2"},
			},
			"hasMore": false,
			"total":   total,
		},
	}
	get := &reader{
		name:   "shipment_get",
		schema: map[string]any{"properties": map[string]any{"shipmentIds": map[string]any{}, "proNumber": map[string]any{}}},
		result: map[string]any{"items": []any{map[string]any{"id": "shp_1", "comments": []any{"called"}}}},
	}
	tracking := &reader{
		name:   "shipment_get_tracking",
		schema: map[string]any{"properties": map[string]any{"shipmentIds": map[string]any{}}},
		result: map[string]any{"items": []any{map[string]any{"id": "shp_1", "eta": "14:00"}}},
	}

	return readers{
		"shipment_list":           list,
		"shipment_get":            get,
		"shipment_get_tracking":   tracking,
		"shipment_list_documents": &reader{name: "shipment_list_documents"},
		"shipment_list_permits":   &reader{name: "shipment_list_permits"},
	}, list, get, tracking
}

func params(values map[string]any) *agenttoolports.QueryToolParams {
	org, bu := pulid.MustNew("org_"), pulid.MustNew("bu_")

	return &agenttoolports.QueryToolParams{
		OrganizationID: org,
		BusinessUnitID: bu,
		Timezone:       "America/Los_Angeles",
		Actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: org,
			BusinessUnitID: bu,
		},
		Params: values,
	}
}

func TestRecordsQuery_AnswersTheSunbeltReeferQuestionInOneCall(t *testing.T) {
	t.Parallel()

	all, list, get, tracking := shipmentReaders()
	tool := recordquerytools.NewQueryTool()
	tool.(recordquerytools.Linker).Link(all)

	result, err := tool.Query(t.Context(), params(map[string]any{
		"entity": "shipment",
		"where": []any{
			map[string]any{"field": "status", "op": "eq", "value": "New"},
			map[string]any{"field": "customer", "op": "contains", "value": "Sunbelt"},
			map[string]any{"field": "temperature.min", "op": "isnotnull"},
			map[string]any{"field": "stops.city", "op": "eq", "value": "Los Angeles"},
			map[string]any{"field": "stops.type", "op": "eq", "value": "Pickup"},
			map[string]any{"field": "pickup.scheduled", "op": "between",
				"values": []any{"2026-11-23", "2026-11-29"}},
		},
		"include": []any{"comments", "tracking"},
	}))
	require.NoError(t, err)

	envelope, ok := result.(*recordquerytools.Envelope)
	require.True(t, ok)
	require.Len(t, list.calls, 1)
	where := querytool.WhereOf(list.calls[0].Params)
	require.NotNil(t, where)
	assert.Equal(t, "Los Angeles", where.Extras["stops.city"])
	assert.Equal(t, "Pickup", where.Extras["stops.type"])
	fields := make([]string, 0, len(where.Filters))
	for _, filter := range where.Filters {
		fields = append(fields, filter.Field)
	}
	assert.Contains(t, fields, "customer.name")
	assert.Contains(t, fields, "temperatureMin")
	assert.Contains(t, fields, "pickupAppointment.scheduledWindowStart")
	for _, filter := range where.Filters {
		if filter.Field == "pickupAppointment.scheduledWindowStart" && filter.Operator == dbtype.OpGreaterThanOrEqual {
			assert.IsType(t, int64(0), filter.Value, "a date is compared as the start of its day")
		}
	}

	assert.Len(t, envelope.AppliedFilters, 6)
	assert.Contains(t, envelope.Includes, "tracking")
	assert.Contains(t, envelope.Includes, "record")
	require.Len(t, tracking.calls, 1)
	assert.Equal(t, []any{"shp_1", "shp_2"}, tracking.calls[0].Params["shipmentIds"])
	require.Len(t, get.calls, 1)
	assert.Equal(t, []any{"shp_1", "shp_2"}, get.calls[0].Params["shipmentIds"])
}

func TestRecordsQuery_CountOnlyReturnsATotalAndNoRows(t *testing.T) {
	t.Parallel()

	all, list, _, _ := shipmentReaders()
	tool := recordquerytools.NewQueryTool()
	tool.(recordquerytools.Linker).Link(all)

	result, err := tool.Query(t.Context(), params(map[string]any{
		"entity":    "shipment",
		"where":     []any{map[string]any{"field": "unassigned", "op": "eq", "value": true}},
		"countOnly": true,
	}))
	require.NoError(t, err)

	envelope := result.(*recordquerytools.Envelope)
	require.NotNil(t, envelope.Total)
	assert.Equal(t, 3, *envelope.Total)
	assert.Empty(t, envelope.Items)
	require.Len(t, list.calls, 1)
	assert.True(t, list.calls[0].IncludeTotal)
	assert.Equal(t, 1, list.calls[0].Params[querytool.ParamLimit])
}

/*
A count of the New shipments came back with the fifteen column names of the
page read to count them and an empty items list beside the total, which read
as a result with no rows. A count carries the filters and the total alone.
*/
func TestRecordsQuery_CountOnlyCarriesNoColumnsOrRows(t *testing.T) {
	t.Parallel()

	all, list, _, _ := shipmentReaders()
	list.result = map[string]any{
		"items":   []any{map[string]any{"id": "shp_1", "proNumber": "PRO-1"}},
		"columns": []any{"proNumber", "status", "customer"},
		"hasMore": true,
		"total":   114,
	}
	tool := recordquerytools.NewQueryTool()
	tool.(recordquerytools.Linker).Link(all)

	result, err := tool.Query(t.Context(), params(map[string]any{
		"entity":    "shipment",
		"where":     []any{map[string]any{"field": "status", "op": "eq", "value": "New"}},
		"countOnly": true,
	}))
	require.NoError(t, err)

	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, sonic.Unmarshal(encoded, &decoded))
	assert.Equal(t, float64(114), decoded["total"])
	assert.Equal(t, false, decoded["hasMore"])
	assert.NotEmpty(t, decoded["appliedFilters"])
	assert.NotContains(t, decoded, "columns")
	assert.NotContains(t, decoded, "items")
	assert.NotContains(t, decoded, "nextCursor")
}

func TestRecordsQuery_AnUnknownFieldNamesTheValidOnes(t *testing.T) {
	t.Parallel()

	all, list, _, _ := shipmentReaders()
	tool := recordquerytools.NewQueryTool()
	tool.(recordquerytools.Linker).Link(all)

	_, err := tool.Query(t.Context(), params(map[string]any{
		"entity": "shipment",
		"where":  []any{map[string]any{"field": "colour", "op": "eq", "value": "red"}},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"colour" is not a field`)
	assert.Contains(t, err.Error(), "stops.city")
	assert.Empty(t, list.calls)
}

func TestRecordsQuery_AnUnknownEntityListsTheEntities(t *testing.T) {
	t.Parallel()

	_, err := recordquerytools.Backing("records_query", map[string]any{"entity": "spaceship"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invoice")
	assert.Contains(t, err.Error(), "shipment")
}

func TestRecordGet_ReadsByNumberAndIncludesSections(t *testing.T) {
	t.Parallel()

	all, _, get, tracking := shipmentReaders()
	tool := recordquerytools.NewGetTool()
	tool.(recordquerytools.Linker).Link(all)

	result, err := tool.Query(t.Context(), params(map[string]any{
		"entity":  "shipment",
		"number":  "PRO-1",
		"include": []any{"all"},
	}))
	require.NoError(t, err)

	envelope := result.(*recordquerytools.Envelope)
	require.Len(t, get.calls, 1)
	assert.Equal(t, "PRO-1", get.calls[0].Params["proNumber"])
	require.Len(t, tracking.calls, 1)
	assert.Contains(t, envelope.Includes, "tracking")
}
