package listquery

import (
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const defaultSearchVectorColumn = "search_vector"

type relation struct {
	index  int32
	name   string
	kind   schema.RelKind
	target *schema.Table
	fields map[string]int32
}

type searchColumn struct {
	index int32
	kind  domaintypes.FieldType
}

type catalog struct {
	table     *schema.Table
	fields    map[string]int32
	relations map[string]relation
	vector    int32
	search    []searchColumn
	strings   []int32
}

var catalogs = struct {
	sync.RWMutex
	m map[*schema.Table]*catalog
}{m: make(map[*schema.Table]*catalog)}

func catalogFor[T any](table *schema.Table) *catalog {
	return catalogOf(reflect.TypeFor[T](), table, searchConfigOf[T])
}

func catalogOf(
	model reflect.Type,
	table *schema.Table,
	config func() *domaintypes.PostgresSearchConfig,
) *catalog {
	catalogs.RLock()
	c, ok := catalogs.m[table]
	catalogs.RUnlock()
	if ok {
		return c
	}

	c = buildCatalog(model, table, config())

	catalogs.Lock()
	defer catalogs.Unlock()
	if existing, found := catalogs.m[table]; found {
		return existing
	}
	catalogs.m[table] = c

	return c
}

func searchConfigOf[T any]() *domaintypes.PostgresSearchConfig {
	searchable, ok := any(new(T)).(domaintypes.PostgresSearchable)
	if !ok {
		return nil
	}

	config := searchable.GetPostgresSearchConfig()
	return &config
}

func buildCatalog(
	model reflect.Type,
	table *schema.Table,
	config *domaintypes.PostgresSearchConfig,
) *catalog {
	c := &catalog{
		table:     table,
		fields:    columnsByJSON(model, table),
		relations: make(map[string]relation),
		vector:    -1,
	}

	for _, index := range c.fields {
		col := &table.Columns[index]
		if col.Kind == schema.KindString && len(col.Enum) == 0 {
			c.strings = append(c.strings, index)
		}
	}
	slices.Sort(c.strings)

	if config == nil {
		return c
	}

	c.relations = queryableRelations(model, table, config.Relationships)
	if config.UseSearchVector {
		c.vector = vectorColumn(table, config.SearchVectorColumn)
	}
	for _, field := range config.SearchableFields {
		if strings.Contains(field.Name, ".") {
			continue
		}
		if index := table.Column(field.Name); index >= 0 && table.Columns[index].Stored() {
			c.search = append(c.search, searchColumn{index: index, kind: field.Type})
		}
	}

	return c
}

func vectorColumn(table *schema.Table, name string) int32 {
	if name == "" {
		name = defaultSearchVectorColumn
	}

	index := table.Column(name)
	if index < 0 || table.Columns[index].Kind != schema.KindTSVector {
		return -1
	}

	return index
}

func queryableRelations(
	model reflect.Type,
	table *schema.Table,
	defs []*domaintypes.RelationshipDefintion,
) map[string]relation {
	allowed := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		if def != nil && def.Queryable {
			allowed[strings.ToLower(def.Field)] = struct{}{}
		}
	}

	out := make(map[string]relation, len(allowed))
	if len(allowed) == 0 {
		return out
	}

	for jsonName, field := range jsonFields(model) {
		_, byJSON := allowed[strings.ToLower(jsonName)]
		_, byField := allowed[strings.ToLower(field.Name)]
		if !byJSON && !byField {
			continue
		}

		index := table.Relation(field.Name)
		if index < 0 || table.Relations[index].Target == nil {
			continue
		}

		rel := &table.Relations[index]
		out[jsonName] = relation{
			index:  index,
			name:   rel.Name,
			kind:   rel.Kind,
			target: rel.Target,
			fields: columnsByJSON(structOf(field.Type), rel.Target),
		}
	}

	return out
}

func columnsByJSON(model reflect.Type, table *schema.Table) map[string]int32 {
	fields := jsonFields(model)
	out := make(map[string]int32, len(fields))
	for jsonName, field := range fields {
		index := columnByField(table, field.Name)
		if index < 0 || !filterable(&table.Columns[index]) {
			continue
		}
		out[jsonName] = index
	}

	return out
}

func columnByField(table *schema.Table, field string) int32 {
	for i := range table.Columns {
		if table.Columns[i].Field == field {
			return int32(i)
		}
	}

	return -1
}

func filterable(col *schema.Column) bool {
	if !col.Stored() || col.Flags.Has(schema.FlagNoSelect) {
		return false
	}

	switch col.Kind {
	case schema.KindTSVector,
		schema.KindJSON,
		schema.KindBytes,
		schema.KindStringArray,
		schema.KindInt64Array:
		return false
	case schema.KindOther,
		schema.KindString,
		schema.KindInt16,
		schema.KindInt32,
		schema.KindInt64,
		schema.KindFloat32,
		schema.KindFloat64,
		schema.KindBool,
		schema.KindTime:
		return true
	default:
		return false
	}
}

func jsonFields(model reflect.Type) map[string]reflect.StructField {
	model = structOf(model)
	if model == nil {
		return nil
	}

	out := make(map[string]reflect.StructField, model.NumField())
	for field := range model.Fields() {
		if !field.IsExported() || field.Anonymous {
			continue
		}

		name := jsonName(field.Tag.Get("json"))
		if name == "" {
			continue
		}
		out[name] = field
	}

	return out
}

func jsonName(tag string) string {
	if tag == "" || tag == "-" {
		return ""
	}

	name, _, _ := strings.Cut(tag, ",")
	return name
}

func structOf(t reflect.Type) reflect.Type {
	for t != nil && (t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice) {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}

	return t
}

func (c *catalog) relation(name string) (relation, bool) {
	rel, ok := c.relations[name]
	return rel, ok
}

func (r relation) toOne() bool {
	switch r.kind {
	case schema.BelongsTo, schema.HasOne, schema.Pick:
		return true
	case schema.HasMany, schema.ManyToMany:
		return false
	default:
		return false
	}
}

func (r relation) countable() bool {
	return r.kind == schema.HasMany || r.kind == schema.HasOne
}
