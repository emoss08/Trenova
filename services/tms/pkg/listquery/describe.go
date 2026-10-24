package listquery

import (
	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

func Describe(
	model domaintypes.PostgresSearchable,
	table *schema.Table,
) *domaintypes.FieldConfiguration {
	if model == nil || table == nil {
		return nil
	}

	c := catalogOf(table, func() *domaintypes.PostgresSearchConfig {
		config := model.GetPostgresSearchConfig()
		return &config
	})

	out := &domaintypes.FieldConfiguration{
		FilterableFields:  make(map[string]bool, len(c.fields)),
		SortableFields:    make(map[string]bool, len(c.fields)),
		NonNullableFields: make(map[string]bool, len(c.fields)),
		IntegerFields:     make(map[string]bool),
		FieldMap:          make(map[string]string, len(c.fields)),
		EnumMap:           make(map[string]bool),
	}
	for name, index := range c.fields {
		describeColumn(out, name, &table.Columns[index], true)
	}
	for name, index := range c.computed {
		describeColumn(out, name, &table.Columns[index], false)
	}
	for relName, rel := range c.relations {
		if !rel.toOne() {
			continue
		}
		for name, index := range rel.fields {
			describeColumn(out, relName+"."+name, &rel.target.Columns[index], false)
		}
	}

	return out
}

func describeColumn(
	out *domaintypes.FieldConfiguration,
	name string,
	col *schema.Column,
	base bool,
) {
	out.FilterableFields[name] = true
	out.SortableFields[name] = true
	if len(col.Enum) > 0 {
		out.EnumMap[name] = true
	}
	if !base {
		return
	}

	out.FieldMap[name] = col.Name
	if !col.MayBeNull() {
		out.NonNullableFields[name] = true
	}
	switch col.Kind {
	case schema.KindInt16, schema.KindInt32, schema.KindInt64:
		out.IntegerFields[name] = true
	default:
	}
}
