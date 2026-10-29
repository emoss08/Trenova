package listcatalog

import (
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

const defaultSearchVectorColumn = "search_vector"

type Relation struct {
	Index  int32
	Name   string
	Kind   schema.RelKind
	Target *schema.Table
	Fields map[string]int32
}

type SearchColumn struct {
	Index int32
	Kind  domaintypes.FieldType
}

type Catalog struct {
	Table     *schema.Table
	Fields    map[string]int32
	Computed  map[string]int32
	Relations map[string]Relation
	Vector    int32
	Search    []SearchColumn
	Fuzzy     [][]int32
	Strings   []int32
}

var catalogs = struct {
	sync.RWMutex
	m map[*schema.Table]*Catalog
}{m: make(map[*schema.Table]*Catalog)}

func For[T any](table *schema.Table) *Catalog {
	return Of(table, searchConfigOf[T])
}

func Of(
	table *schema.Table,
	config func() *domaintypes.PostgresSearchConfig,
) *Catalog {
	catalogs.RLock()
	c, ok := catalogs.m[table]
	catalogs.RUnlock()
	if ok {
		return c
	}

	c = Build(table, config())

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

func Build(
	table *schema.Table,
	config *domaintypes.PostgresSearchConfig,
) *Catalog {
	c := &Catalog{
		Table:     table,
		Fields:    columnsByJSON(table),
		Computed:  make(map[string]int32),
		Relations: make(map[string]Relation),
		Vector:    -1,
	}

	for _, index := range c.Fields {
		col := &table.Columns[index]
		if col.Kind == schema.KindString && len(col.Enum) == 0 {
			c.Strings = append(c.Strings, index)
		}
	}
	slices.Sort(c.Strings)

	if config == nil {
		return c
	}

	c.Relations = queryableRelations(table, config.Relationships)
	c.Computed = computedFields(table, config.ComputedFilters)
	if config.UseSearchVector {
		c.Vector = vectorColumn(table, config.SearchVectorColumn)
	}
	for _, field := range config.SearchableFields {
		if strings.Contains(field.Name, ".") {
			continue
		}
		if index := table.Column(field.Name); index >= 0 && table.Columns[index].Stored() {
			c.Search = append(c.Search, SearchColumn{Index: index, Kind: field.Type})
		}
	}
	c.Fuzzy = fuzzyColumns(table, config.FuzzyFields)

	return c
}

func fuzzyColumns(table *schema.Table, fields []domaintypes.FuzzyField) [][]int32 {
	out := make([][]int32, 0, len(fields))
	for _, field := range fields {
		if columns, ok := textColumns(table, field); ok {
			out = append(out, columns)
		}
	}

	return out
}

func textColumns(table *schema.Table, names []string) ([]int32, bool) {
	if len(names) == 0 {
		return nil, false
	}

	columns := make([]int32, 0, len(names))
	for _, name := range names {
		index := table.Column(name)
		if index < 0 {
			return nil, false
		}
		col := &table.Columns[index]
		if !col.Stored() || col.Kind != schema.KindString || len(col.Enum) > 0 {
			return nil, false
		}
		columns = append(columns, index)
	}

	return columns, true
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
	table *schema.Table,
	defs []*domaintypes.RelationshipDefintion,
) map[string]Relation {
	allowed := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		if def != nil && def.Queryable {
			allowed[strings.ToLower(def.Field)] = struct{}{}
		}
	}

	out := make(map[string]Relation, len(allowed))
	if len(allowed) == 0 {
		return out
	}

	for i := range table.Relations {
		rel := &table.Relations[i]
		if rel.JSON == "" || rel.Target == nil {
			continue
		}
		_, byJSON := allowed[strings.ToLower(rel.JSON)]
		_, byName := allowed[strings.ToLower(rel.Name)]
		if !byJSON && !byName {
			continue
		}

		out[rel.JSON] = Relation{
			Index:  int32(i),
			Name:   rel.Name,
			Kind:   rel.Kind,
			Target: rel.Target,
			Fields: columnsByJSON(rel.Target),
		}
	}

	return out
}

func columnsByJSON(table *schema.Table) map[string]int32 {
	out := make(map[string]int32, len(table.Columns))
	for i := range table.Columns {
		col := &table.Columns[i]
		if col.JSON == "" || !filterable(col) {
			continue
		}
		out[col.JSON] = int32(i)
	}

	return out
}

func computedFields(table *schema.Table, names []string) map[string]int32 {
	out := make(map[string]int32, len(names))
	for i := range table.Columns {
		col := &table.Columns[i]
		if col.JSON == "" || col.Stored() || !slices.Contains(names, col.JSON) {
			continue
		}
		if filterableKind(col.Kind) {
			out[col.JSON] = int32(i)
		}
	}

	return out
}

func filterable(col *schema.Column) bool {
	if !col.Stored() || col.Flags.Has(schema.FlagNoSelect) {
		return false
	}

	return filterableKind(col.Kind)
}

func filterableKind(kind schema.Kind) bool {
	switch kind {
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

func (c *Catalog) Relation(name string) (Relation, bool) {
	rel, ok := c.Relations[name]
	return rel, ok
}

func (r Relation) ToOne() bool {
	switch r.Kind {
	case schema.BelongsTo, schema.HasOne, schema.Pick:
		return true
	case schema.HasMany, schema.ManyToMany:
		return false
	default:
		return false
	}
}

func (r Relation) Countable() bool {
	return r.Kind == schema.HasMany || r.Kind == schema.HasOne
}
