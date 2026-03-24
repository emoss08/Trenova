package domaintypes

import (
	"github.com/emoss08/trenova/pkg/dbtype"
)

type FieldType string

const (
	FieldTypeText      = FieldType("Text")
	FieldTypeNumber    = FieldType("Number")
	FieldTypeBoolean   = FieldType("Boolean")
	FieldTypeEnum      = FieldType("Enum")
	FieldTypeDate      = FieldType("Date")
	FieldTypeComposite = FieldType("Composite")
)

type SearchWeight string

const (
	SearchWeightA     = SearchWeight("A")
	SearchWeightB     = SearchWeight("B")
	SearchWeightC     = SearchWeight("C")
	SearchWeightD     = SearchWeight("D")
	SearchWeightBlank = SearchWeight("")
)

func (w SearchWeight) GetScore() int {
	switch w {
	case SearchWeightA:
		return 4
	case SearchWeightB:
		return 3
	case SearchWeightC:
		return 2
	case SearchWeightD:
		return 1
	case SearchWeightBlank:
		return 0
	}

	return 1
}

type SearchableField struct {
	Name   string // database field name
	Type   FieldType
	Weight SearchWeight
}

type PostgresSearchConfig struct {
	TableAlias         string
	SearchableFields   []SearchableField
	Relationships      []*RelationshipDefintion
	UseSearchVector    bool   // whether to use "search_vector" column for text search
	SearchVectorColumn string // name of the search vector column
}

type JoinStep struct {
	Table     string
	Alias     string
	Condition string
	JoinType  dbtype.JoinType
}

type RelationshipDefintion struct {
	Field        string
	Type         dbtype.RelationshipType
	TargetEntity any    // target entity type (for field extraction)
	TargetTable  string // target table name
	ForeignKey   string // foreign key field (e.g. "user_id")
	ReferenceKey string // reference key in target (e.g. "id")
	Alias        string // table alias to use for joins
	Queryable    bool   // whether this relationship can be used in queries

	JoinTable          string // join table name (e.g. "user_roles")
	JoinTableAlias     string // join table alias (e.g. "usr")
	JoinTableSourceKey string // source key in join table (e.g. "user_id")
	JoinTableTargetKey string // taget key in join table (e.g. "role_id")

	CustomJoinPath []JoinStep // explicit join sequence for complex relationships
	TargetField    string     // the field to reference on the final table (e.g. "name")
	IsEnum         bool       // whethere the target field is an enum type
}

type PostgresSearchable interface {
	GetPostgresSearchConfig() PostgresSearchConfig
	GetTableName() string
}

type NestedFieldDefintion struct {
	DatabaseField string     `json:"databaseField"`
	RequiredJoins []JoinStep `json:"requiredJoins"`
	IsEnum        bool       `json:"isEnum"`
}

type FieldConfiguration struct {
	FilterableFields    map[string]bool                 `json:"filterableFields"`
	SortableFields      map[string]bool                 `json:"sortableFields"`
	GeoFilterableFields map[string]bool                 `json:"geoFilterableFields"`
	FieldMap            map[string]string               `json:"fieldMap"`
	EnumMap             map[string]bool                 `json:"enumMap"`
	NestedFields        map[string]NestedFieldDefintion `json:"nestedFields"`
}

type SortField struct {
	Field     string               `json:"field"     form:"field"`
	Direction dbtype.SortDirection `json:"direction" form:"direction" binding:"oneof=asc desc"`
}

type FieldFilter struct {
	Field    string          `json:"field"    form:"field"`
	Operator dbtype.Operator `json:"operator" form:"operator" binding:"oneof=eq ne gt gte lt lte contains startswith endswith like ilike in notin isnull isnotnull daterange lastndays nextndays today yesterday tomorrow"`
	Value    any             `json:"value"    form:"value"`
}

type FilterGroup struct {
	Filters []FieldFilter `json:"filters"`
}

type GeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type GeoFilter struct {
	Field    string   `json:"field"`
	Center   GeoPoint `json:"center"`
	RadiusKm float64  `json:"radiusKm"`
}

type AggregateFilter struct {
	Relation string          `json:"relation"`
	Operator dbtype.Operator `json:"operator" binding:"oneof=countgt countlt counteq countgte countlte"`
	Value    int             `json:"value"`
}
