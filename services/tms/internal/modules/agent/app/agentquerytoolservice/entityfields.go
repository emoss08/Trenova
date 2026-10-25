package agentquerytoolservice

import (
	"reflect"

	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/keelcatalog"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
)

func entityTable(entity domaintypes.PostgresSearchable) (*schema.Table, bool) {
	if entity == nil {
		return nil, false
	}
	model := reflect.TypeOf(entity)
	for model.Kind() == reflect.Pointer {
		model = model.Elem()
	}
	table, ok := keelcatalog.ForModel(model.String())

	return table, ok
}

func entityFields(entity domaintypes.PostgresSearchable) *domaintypes.FieldConfiguration {
	table, ok := entityTable(entity)
	if !ok {
		return &domaintypes.FieldConfiguration{}
	}

	return listquery.Describe(entity, table)
}
