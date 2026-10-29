package listquery

import (
	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listcatalog"
)

func Describe(
	model domaintypes.PostgresSearchable,
	table *schema.Table,
) *domaintypes.FieldConfiguration {
	if model == nil || table == nil {
		return nil
	}

	c := listcatalog.Of(table, func() *domaintypes.PostgresSearchConfig {
		config := model.GetPostgresSearchConfig()
		return &config
	})

	out := &domaintypes.FieldConfiguration{
		FilterableFields:  make(map[string]bool, len(c.Fields)),
		SortableFields:    make(map[string]bool, len(c.Fields)),
		NonNullableFields: make(map[string]bool, len(c.Fields)),
		IntegerFields:     make(map[string]bool),
		FieldMap:          make(map[string]string, len(c.Fields)),
		EnumMap:           make(map[string]bool),
	}
	for name, index := range c.Fields {
		describeColumn(out, name, &table.Columns[index], true)
	}
	for name, index := range c.Computed {
		describeColumn(out, name, &table.Columns[index], false)
	}
	for relName, rel := range c.Relations {
		if !rel.ToOne() {
			continue
		}
		for name, index := range rel.Fields {
			describeColumn(out, relName+"."+name, &rel.Target.Columns[index], false)
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
	if col.Kind == schema.KindInt16 || col.Kind == schema.KindInt32 ||
		col.Kind == schema.KindInt64 {
		out.IntegerFields[name] = true
	}
}
