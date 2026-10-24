package agentquerytoolservice

import (
	"reflect"
	"sync"

	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/keeltables"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
)

var entityTables = sync.OnceValue(func() map[string]*schema.Table {
	tables := keeltables.All()
	out := make(map[string]*schema.Table, len(tables))
	ambiguous := make(map[string]bool)
	for _, table := range tables {
		if _, seen := out[table.Model]; seen {
			ambiguous[table.Model] = true
		}
		out[table.Model] = table
	}
	for model := range ambiguous {
		delete(out, model)
	}

	return out
})

func entityTable(entity domaintypes.PostgresSearchable) (*schema.Table, bool) {
	if entity == nil {
		return nil, false
	}
	model := reflect.TypeOf(entity)
	for model.Kind() == reflect.Pointer {
		model = model.Elem()
	}
	table, ok := entityTables()[model.String()]

	return table, ok
}

func entityFields(entity domaintypes.PostgresSearchable) *domaintypes.FieldConfiguration {
	table, ok := entityTable(entity)
	if !ok {
		return &domaintypes.FieldConfiguration{}
	}

	return listquery.Describe(entity, table)
}
