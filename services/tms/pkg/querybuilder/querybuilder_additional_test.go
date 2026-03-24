package querybuilder

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

type additionalSearchEntity struct {
	ID     string `json:"id"     bun:"id,pk"`
	Name   string `json:"name"   bun:"name"`
	Status string `json:"status" bun:"status"`
	Code   string `json:"code"   bun:"code"`
	Count  int    `json:"count"  bun:"count"`
	Active bool   `json:"active" bun:"active"`
}

func (s *additionalSearchEntity) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:         "sve",
		UseSearchVector:    true,
		SearchVectorColumn: "search_vector",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "code", Type: domaintypes.FieldTypeText},
			{Name: "count", Type: domaintypes.FieldTypeNumber},
			{Name: "active", Type: domaintypes.FieldTypeBoolean},
		},
	}
}

func (s *additionalSearchEntity) GetTableName() string {
	return "extended_search_entities"
}

type customRelEntity struct {
	ID   string `json:"id"   bun:"id,pk"`
	Name string `json:"name" bun:"name"`
}

func (c *customRelEntity) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "cre",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:        "worker",
				Type:         dbtype.RelationshipTypeCustom,
				TargetTable:  "workers",
				Alias:        "w",
				ForeignKey:   "worker_id",
				ReferenceKey: "id",
				Queryable:    true,
				IsEnum:       true,
				TargetField:  "status",
				CustomJoinPath: []domaintypes.JoinStep{
					{
						Table:     "workers",
						Alias:     "w",
						Condition: "main.worker_id = w.id",
						JoinType:  dbtype.JoinTypeLeft,
					},
				},
			},
			{
				Field:        "owner",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetTable:  "owners",
				Alias:        "o",
				ForeignKey:   "owner_id",
				ReferenceKey: "id",
				Queryable:    true,
			},
			{
				Field:              "tags",
				Type:               dbtype.RelationshipTypeManyToMany,
				TargetTable:        "tags",
				Alias:              "tg",
				ForeignKey:         "tag_id",
				ReferenceKey:       "id",
				Queryable:          true,
				JoinTable:          "entity_tags",
				JoinTableAlias:     "et",
				JoinTableSourceKey: "entity_id",
				JoinTableTargetKey: "tag_id",
			},
			{
				Field:        "detail",
				Type:         dbtype.RelationshipTypeHasOne,
				TargetTable:  "details",
				Alias:        "d",
				ForeignKey:   "entity_id",
				ReferenceKey: "id",
				Queryable:    true,
			},
			{
				Field:        "items",
				Type:         dbtype.RelationshipTypeHasMany,
				TargetTable:  "items",
				Alias:        "it",
				ForeignKey:   "entity_id",
				ReferenceKey: "id",
				Queryable:    true,
			},
			{
				Field:       "emptyCustom",
				Type:        dbtype.RelationshipTypeCustom,
				TargetTable: "empty_table",
				Queryable:   true,
				CustomJoinPath: []domaintypes.JoinStep{
					{
						Table:     "empty_table",
						Alias:     "emp",
						Condition: "main.empty_id = emp.id",
					},
				},
			},
		},
	}
}

func (c *customRelEntity) GetTableName() string {
	return "custom_rel_entities"
}

func newAdditionalTestDB() *bun.DB {
	return bun.NewDB(&sql.DB{}, pgdialect.New())
}

func TestIsValidSearchTerm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		term     string
		expected bool
	}{
		{"letter only", "hello", true},
		{"digit only", "123", true},
		{"mixed", "abc123", true},
		{"special chars with letter", "!@#a", true},
		{"special chars only", "!@#$%", false},
		{"empty string", "", false},
		{"single letter", "a", true},
		{"single digit", "1", true},
		{"whitespace only", "   ", false},
		{"unicode letter", "\u00e9", true},
		{"symbols", "---", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isValidSearchTerm(tt.term)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryBuilder_HasWebsearchOperators(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{"plain text", "hello world", false},
		{"quoted phrase", `"exact match"`, true},
		{"OR operator uppercase", "cat OR dog", true},
		{"or operator lowercase", "cat or dog", true},
		{"negation", "-excluded word", true},
		{"negation single char", "- word", false},
		{"no operators", "simple search", false},
		{"empty string", "", false},
		{"single word", "test", false},
		{"OR in middle of word", "FOREST", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			query := db.NewSelect().ModelTableExpr("test AS t")
			qb := &QueryBuilder{query: query, tableAlias: "t"}
			result := qb.hasWebsearchOperators(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryBuilder_IsStringOperation(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()

	tests := []struct {
		name     string
		operator dbtype.Operator
		expected bool
	}{
		{"contains", dbtype.OpContains, true},
		{"equal", dbtype.OpEqual, true},
		{"startsWith", dbtype.OpStartsWith, true},
		{"endsWith", dbtype.OpEndsWith, true},
		{"like", dbtype.OpLike, true},
		{"ilike", dbtype.OpILike, true},
		{"greaterThan", dbtype.OpGreaterThan, false},
		{"lessThan", dbtype.OpLessThan, false},
		{"in", dbtype.OpIn, false},
		{"notIn", dbtype.OpNotIn, false},
		{"isNull", dbtype.OpIsNull, false},
		{"isNotNull", dbtype.OpIsNotNull, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			query := db.NewSelect().ModelTableExpr("test AS t")
			qb := &QueryBuilder{query: query, tableAlias: "t"}
			result := qb.isStringOperation(tt.operator)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryBuilder_IsEmptyEnumValue(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()

	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"empty string", "", true},
		{"non-empty string", "active", false},
		{"integer", 42, false},
		{"nil", nil, false},
		{"boolean", true, false},
		{"float", 3.14, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			query := db.NewSelect().ModelTableExpr("test AS t")
			qb := &QueryBuilder{query: query, tableAlias: "t"}
			result := qb.isEmptyEnumValue(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryBuilder_GetFieldReference(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()

	tests := []struct {
		name       string
		tableAlias string
		dbField    string
		expected   string
	}{
		{"with alias no dot", "t", "name", "t.name"},
		{"with alias with dot", "t", "rel.name", "rel.name"},
		{"no alias no dot", "", "name", "name"},
		{"no alias with dot", "", "rel.name", "rel.name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			query := db.NewSelect().ModelTableExpr("test AS t")
			qb := &QueryBuilder{query: query, tableAlias: tt.tableAlias}
			result := qb.getFieldReference(tt.dbField)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryBuilder_GetDBField_FieldMap(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"myField": true},
		SortableFields:   map[string]bool{"myField": true},
		FieldMap:         map[string]string{"myField": "my_field"},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		appliedJoins: make(map[string]bool),
	}

	result := qb.getDBField("myField")
	assert.Equal(t, "my_field", result)
}

func TestQueryBuilder_GetDBField_DottedField(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		appliedJoins: make(map[string]bool),
	}

	result := qb.getDBField("relatedField.name")
	assert.Equal(t, "related_field.name", result)
}

func TestQueryBuilder_GetDBField_PassthroughUnknown(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		appliedJoins: make(map[string]bool),
	}

	result := qb.getDBField("unknown_field")
	assert.Equal(t, "unknown_field", result)
}

func TestQueryBuilder_GetDBField_NestedField(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"rel.name": true},
		SortableFields:   map[string]bool{"rel.name": true},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields: map[string]domaintypes.NestedFieldDefintion{
			"rel.name": {
				DatabaseField: "r.name",
				RequiredJoins: []domaintypes.JoinStep{
					{
						Table:     "related_table",
						Alias:     "r",
						Condition: "t.related_id = r.id",
						JoinType:  dbtype.JoinTypeLeft,
					},
				},
			},
		},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		appliedJoins: make(map[string]bool),
	}

	result := qb.getDBField("rel.name")
	assert.Equal(t, "r.name", result)
	assert.True(t, qb.appliedJoins["related_table_r"])
}

func TestQueryBuilder_IsEnumField(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{"status": true, "type": true},
		NestedFields: map[string]domaintypes.NestedFieldDefintion{
			"rel.status": {
				DatabaseField: "r.status",
				IsEnum:        true,
			},
		},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		appliedJoins: make(map[string]bool),
	}

	assert.True(t, qb.isEnumField("status"))
	assert.True(t, qb.isEnumField("type"))
	assert.False(t, qb.isEnumField("name"))
	assert.True(t, qb.isEnumField("rel.status"))
}

func TestQueryBuilder_ResolveNestedField_NoSearchConfig(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		searchConfig: nil,
		appliedJoins: make(map[string]bool),
	}

	sqlField, joinDefs := qb.resolveNestedField("related.name")
	assert.Equal(t, "related.name", sqlField)
	assert.Nil(t, joinDefs)
}

func TestQueryBuilder_ResolveNestedField_SinglePart(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:      query,
		tableAlias: "t",
		searchConfig: &domaintypes.PostgresSearchConfig{
			TableAlias: "t",
		},
		appliedJoins: make(map[string]bool),
	}

	sqlField, joinDefs := qb.resolveNestedField("singleField")
	assert.Equal(t, "singleField", sqlField)
	assert.Nil(t, joinDefs)
}

func TestQueryBuilder_ResolveNestedField_NoMatchingRelationship(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:      query,
		tableAlias: "t",
		searchConfig: &domaintypes.PostgresSearchConfig{
			TableAlias:    "t",
			Relationships: []*domaintypes.RelationshipDefintion{},
		},
		appliedJoins: make(map[string]bool),
	}

	sqlField, joinDefs := qb.resolveNestedField("unknown.name")
	assert.Empty(t, sqlField)
	assert.Nil(t, joinDefs)
}

func TestQueryBuilder_ResolveCustomRelationship(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &customRelEntity{}
	config := entity.GetPostgresSearchConfig()

	query := db.NewSelect().ModelTableExpr("custom_rel_entities AS cre")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "cre",
		searchConfig: &config,
		entity:       entity,
		appliedJoins: make(map[string]bool),
	}

	sqlField, joinDefs := qb.resolveNestedField("worker.status")
	assert.Equal(t, "w.status", sqlField)
	require.Len(t, joinDefs, 1)
	assert.Equal(t, "workers", joinDefs[0].Table)
	assert.Equal(t, "w", joinDefs[0].Alias)
	assert.Equal(t, dbtype.JoinTypeLeft, joinDefs[0].JoinType)
}

func TestQueryBuilder_ResolveCustomRelationship_EmptyAlias(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	config := &domaintypes.PostgresSearchConfig{
		TableAlias: "t",
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:       "noAlias",
				Type:        dbtype.RelationshipTypeCustom,
				TargetTable: "no_alias_table",
				Queryable:   true,
				CustomJoinPath: []domaintypes.JoinStep{
					{
						Table:     "no_alias_table",
						Alias:     "",
						Condition: "t.no_alias_id = .id",
					},
				},
			},
		},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		searchConfig: config,
		appliedJoins: make(map[string]bool),
	}

	sqlField, _ := qb.resolveNestedField("noAlias.name")
	assert.Empty(t, sqlField)
}

func TestQueryBuilder_ResolveShipmentDestinationLocationUsesLastMoveAndStop(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &shipment.Shipment{}
	config := entity.GetPostgresSearchConfig()

	query := db.NewSelect().Model((*shipment.Shipment)(nil)).ModelTableExpr("shipments AS sp")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "sp",
		searchConfig: &config,
		entity:       entity,
		appliedJoins: make(map[string]bool),
	}

	sqlField, joinDefs := qb.resolveNestedField("destinationLocation.name")
	assert.Equal(t, "dest_loc.name", sqlField)
	require.Len(t, joinDefs, 3)
	assert.Equal(t, "shipment_moves", joinDefs[0].Table)
	assert.Equal(t, "sm_dest", joinDefs[0].Alias)
	assert.Contains(t, joinDefs[0].Condition, "sp.id = sm_dest.shipment_id")
	assert.Contains(t, joinDefs[0].Condition, "sm_dest.sequence = (SELECT MAX(sm2.sequence) FROM shipment_moves AS sm2 WHERE sm2.shipment_id = sp.id)")
	assert.Equal(t, "stops", joinDefs[1].Table)
	assert.Equal(t, "stop_dest", joinDefs[1].Alias)
	assert.Contains(t, joinDefs[1].Condition, "sm_dest.id = stop_dest.shipment_move_id")
	assert.Contains(t, joinDefs[1].Condition, "stop_dest.sequence = (SELECT MAX(stp2.sequence) FROM stops AS stp2 WHERE stp2.shipment_move_id = sm_dest.id)")
	assert.NotContains(t, joinDefs[1].Condition, "stop_dest.type")
	assert.Equal(t, "locations", joinDefs[2].Table)
	assert.Equal(t, "dest_loc", joinDefs[2].Alias)

	fieldConfig := GetFieldConfiguration(entity)
	filterQuery := db.NewSelect().Model((*shipment.Shipment)(nil)).ModelTableExpr("shipments AS sp")
	filterQB := NewWithPostgresSearch(filterQuery, "sp", fieldConfig, entity).WithTraversalSupport(true)
	filterQB.ApplyFilters([]domaintypes.FieldFilter{
		{Field: "destinationLocation.name", Operator: dbtype.OpContains, Value: "Dallas"},
	})

	sql := filterQB.GetQuery().String()
	assert.Contains(t, sql, `LEFT JOIN shipment_moves AS sm_dest`)
	assert.Contains(t, sql, `sm_dest.sequence = (SELECT MAX(sm2.sequence) FROM shipment_moves AS sm2 WHERE sm2.shipment_id = sp.id)`)
	assert.Contains(t, sql, `LEFT JOIN stops AS stop_dest`)
	assert.Contains(t, sql, `stop_dest.sequence = (SELECT MAX(stp2.sequence) FROM stops AS stp2 WHERE stp2.shipment_move_id = sm_dest.id)`)
	assert.NotContains(t, sql, `stop_dest.type`)
}

func TestQueryBuilder_ResolveBelongsToRelationship(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &customRelEntity{}
	config := entity.GetPostgresSearchConfig()

	query := db.NewSelect().ModelTableExpr("custom_rel_entities AS cre")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "cre",
		searchConfig: &config,
		entity:       entity,
		appliedJoins: make(map[string]bool),
	}

	sqlField, joinDefs := qb.resolveNestedField("owner.name")
	assert.Equal(t, "o.name", sqlField)
	require.Len(t, joinDefs, 1)
	assert.Equal(t, "owners", joinDefs[0].Table)
	assert.Equal(t, "o", joinDefs[0].Alias)
	assert.Equal(t, dbtype.JoinTypeLeft, joinDefs[0].JoinType)
	assert.Contains(t, joinDefs[0].Condition, "cre.owner_id = o.id")
}

func TestQueryBuilder_ResolveManyToManyRelationship(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &customRelEntity{}
	config := entity.GetPostgresSearchConfig()

	query := db.NewSelect().ModelTableExpr("custom_rel_entities AS cre")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "cre",
		searchConfig: &config,
		entity:       entity,
		appliedJoins: make(map[string]bool),
	}

	sqlField, joinDefs := qb.resolveNestedField("tags.name")
	assert.Equal(t, "tg.name", sqlField)
	require.Len(t, joinDefs, 2)
	assert.Equal(t, "entity_tags", joinDefs[0].Table)
	assert.Equal(t, "et", joinDefs[0].Alias)
	assert.Equal(t, "tags", joinDefs[1].Table)
	assert.Equal(t, "tg", joinDefs[1].Alias)
}

func TestQueryBuilder_ResolveHasOneRelationship(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &customRelEntity{}
	config := entity.GetPostgresSearchConfig()

	query := db.NewSelect().ModelTableExpr("custom_rel_entities AS cre")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "cre",
		searchConfig: &config,
		entity:       entity,
		appliedJoins: make(map[string]bool),
	}

	sqlField, joinDefs := qb.resolveNestedField("detail.name")
	assert.Equal(t, "d.name", sqlField)
	require.Len(t, joinDefs, 1)
	assert.Equal(t, "details", joinDefs[0].Table)
	assert.Equal(t, "d", joinDefs[0].Alias)
	assert.Contains(t, joinDefs[0].Condition, "d.entity_id = cre.id")
}

func TestQueryBuilder_ResolveHasManyRelationship(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &customRelEntity{}
	config := entity.GetPostgresSearchConfig()

	query := db.NewSelect().ModelTableExpr("custom_rel_entities AS cre")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "cre",
		searchConfig: &config,
		entity:       entity,
		appliedJoins: make(map[string]bool),
	}

	sqlField, joinDefs := qb.resolveNestedField("items.fieldName")
	assert.Equal(t, "it.field_name", sqlField)
	require.Len(t, joinDefs, 1)
	assert.Equal(t, "items", joinDefs[0].Table)
	assert.Equal(t, "it", joinDefs[0].Alias)
	assert.Contains(t, joinDefs[0].Condition, "it.entity_id = cre.id")
}

func TestQueryBuilder_IsCustomRelationshipEnum(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &customRelEntity{}
	config := entity.GetPostgresSearchConfig()

	query := db.NewSelect().ModelTableExpr("custom_rel_entities AS cre")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "cre",
		searchConfig: &config,
		entity:       entity,
		appliedJoins: make(map[string]bool),
	}

	assert.True(t, qb.isCustomRelationshipEnum("worker.status"))
	assert.False(t, qb.isCustomRelationshipEnum("owner.name"))
	assert.False(t, qb.isCustomRelationshipEnum("singleField"))
	assert.False(t, qb.isCustomRelationshipEnum("unknown.field"))
}

func TestQueryBuilder_IsCustomRelationshipEnum_NilConfig(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		searchConfig: nil,
		appliedJoins: make(map[string]bool),
	}

	assert.False(t, qb.isCustomRelationshipEnum("worker.status"))
}

func TestQueryBuilder_ApplyTraversalFilter_ShortPath(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &customRelEntity{}
	fieldConfig := GetFieldConfiguration(entity)

	query := db.NewSelect().ModelTableExpr("custom_rel_entities AS cre")
	qb := NewWithPostgresSearch(query, "cre", fieldConfig, entity)
	qb.WithTraversalSupport(true)

	initialInvalid := len(qb.invalidFields)
	qb.applyTraversalFilter("singleField", domaintypes.FieldFilter{
		Field:    "singleField",
		Operator: dbtype.OpEqual,
		Value:    "test",
	})

	assert.Equal(t, initialInvalid, len(qb.invalidFields))
}

func TestQueryBuilder_FindRelationship(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &customRelEntity{}
	config := entity.GetPostgresSearchConfig()

	query := db.NewSelect().ModelTableExpr("custom_rel_entities AS cre")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "cre",
		searchConfig: &config,
		entity:       entity,
		appliedJoins: make(map[string]bool),
	}

	rel := qb.findRelationship("worker")
	require.NotNil(t, rel)
	assert.Equal(t, "worker", rel.Field)
	assert.Equal(t, dbtype.RelationshipTypeCustom, rel.Type)

	rel = qb.findRelationship("owner")
	require.NotNil(t, rel)
	assert.Equal(t, dbtype.RelationshipTypeBelongsTo, rel.Type)

	rel = qb.findRelationship("nonexistent")
	assert.Nil(t, rel)
}

func TestQueryBuilder_FindRelationship_CaseInsensitive(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	entity := &customRelEntity{}
	config := entity.GetPostgresSearchConfig()

	query := db.NewSelect().ModelTableExpr("custom_rel_entities AS cre")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "cre",
		searchConfig: &config,
		entity:       entity,
		appliedJoins: make(map[string]bool),
	}

	rel := qb.findRelationship("Worker")
	require.NotNil(t, rel)
	assert.Equal(t, "worker", rel.Field)

	rel = qb.findRelationship("OWNER")
	require.NotNil(t, rel)
	assert.Equal(t, "owner", rel.Field)
}

func TestQueryBuilder_ApplyNestedFieldJoins_Deduplication(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := New(query, "t", fieldConfig)

	joins := []domaintypes.JoinStep{
		{
			Table:     "related",
			Alias:     "r",
			Condition: "t.related_id = r.id",
			JoinType:  dbtype.JoinTypeLeft,
		},
	}

	qb.applyNestedFieldJoins(joins)
	assert.True(t, qb.appliedJoins["related_r"])

	qb.applyNestedFieldJoins(joins)
	assert.True(t, qb.appliedJoins["related_r"])
}

func TestQueryBuilder_ApplyNestedFieldJoins_AllJoinTypes(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	tests := []struct {
		name     string
		joinType dbtype.JoinType
	}{
		{"left join", dbtype.JoinTypeLeft},
		{"right join", dbtype.JoinTypeRight},
		{"inner join", dbtype.JoinTypeInner},
		{"unknown join type", dbtype.JoinType("CROSS")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			query := db.NewSelect().ModelTableExpr("test AS t")
			qb := New(query, "t", fieldConfig)

			joins := []domaintypes.JoinStep{
				{
					Table:     "table_" + string(tt.joinType),
					Alias:     "alias_" + string(tt.joinType),
					Condition: "t.id = alias.id",
					JoinType:  tt.joinType,
				},
			}

			qb.applyNestedFieldJoins(joins)
			expectedKey := "table_" + string(tt.joinType) + "_alias_" + string(tt.joinType)
			assert.True(t, qb.appliedJoins[expectedKey])
		})
	}
}

func TestQueryBuilder_ApplyTraversalSort_NoJoins(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &customRelEntity{}
	fieldConfig := GetFieldConfiguration(entity)
	config := entity.GetPostgresSearchConfig()

	query := db.NewSelect().ModelTableExpr("custom_rel_entities AS cre")
	qb := &QueryBuilder{
		query:            query,
		tableAlias:       "cre",
		fieldConfig:      fieldConfig,
		searchConfig:     &config,
		entity:           entity,
		appliedJoins:     make(map[string]bool),
		traversalEnabled: true,
		invalidFields:    make([]string, 0),
	}

	qb.applyTraversalSort(domaintypes.SortField{
		Field:     "unknown.field",
		Direction: dbtype.SortDirectionAsc,
	})

	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_ConfigBasedSearch_NumberField(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &additionalSearchEntity{}
	config := entity.GetPostgresSearchConfig()
	config.UseSearchVector = false

	fieldConfig := GetFieldConfiguration(entity)

	query := db.NewSelect().ModelTableExpr("search_vector_entities AS sve")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "sve",
		fieldConfig:  fieldConfig,
		searchConfig: &config,
		entity:       entity,
		appliedJoins: make(map[string]bool),
	}

	qb.applyConfigBasedSearch("42")
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_ConfigBasedSearch_NonNumericForNumberField(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &additionalSearchEntity{}
	config := entity.GetPostgresSearchConfig()
	config.UseSearchVector = false

	fieldConfig := GetFieldConfiguration(entity)

	query := db.NewSelect().ModelTableExpr("search_vector_entities AS sve")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "sve",
		fieldConfig:  fieldConfig,
		searchConfig: &config,
		entity:       entity,
		appliedJoins: make(map[string]bool),
	}

	qb.applyConfigBasedSearch("not-a-number")
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_ConfigBasedSearch_DottedFieldSkipped(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	config := &domaintypes.PostgresSearchConfig{
		TableAlias: "t",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "rel.name", Type: domaintypes.FieldTypeText},
			{Name: "direct_field", Type: domaintypes.FieldTypeText},
		},
	}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		searchConfig: config,
		appliedJoins: make(map[string]bool),
	}

	qb.applyConfigBasedSearch("test")
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_TextSearch_SearchVectorCustomColumn(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	config := &domaintypes.PostgresSearchConfig{
		TableAlias:         "t",
		UseSearchVector:    true,
		SearchVectorColumn: "custom_sv",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
		},
	}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"name": true},
		SortableFields:   map[string]bool{"name": true},
		FieldMap:         map[string]string{"name": "name"},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		searchConfig: config,
		appliedJoins: make(map[string]bool),
	}

	qb.ApplyTextSearch("hello", nil)
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_TextSearch_EmptySearchVectorColumn(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	config := &domaintypes.PostgresSearchConfig{
		TableAlias:         "t",
		UseSearchVector:    true,
		SearchVectorColumn: "",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
		},
	}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"name": true},
		SortableFields:   map[string]bool{"name": true},
		FieldMap:         map[string]string{"name": "name"},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		searchConfig: config,
		appliedJoins: make(map[string]bool),
	}

	qb.ApplyTextSearch("hello", nil)
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_TextSearch_NoTableAlias(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	config := &domaintypes.PostgresSearchConfig{
		TableAlias:      "",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
		},
	}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"name": true},
		SortableFields:   map[string]bool{"name": true},
		FieldMap:         map[string]string{"name": "name"},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "",
		fieldConfig:  fieldConfig,
		searchConfig: config,
		appliedJoins: make(map[string]bool),
	}

	qb.ApplyTextSearch("hello", nil)
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_WebsearchWithPrefix_EmptyWords(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	config := &domaintypes.PostgresSearchConfig{
		TableAlias:      "t",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
		},
	}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"name": true},
		SortableFields:   map[string]bool{"name": true},
		FieldMap:         map[string]string{"name": "name"},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		searchConfig: config,
		appliedJoins: make(map[string]bool),
	}

	qb.applyWebsearchWithPrefix("! @", "t.", "search_vector")
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_WebsearchWithPrefix_SpecialCharsStripped(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	config := &domaintypes.PostgresSearchConfig{
		TableAlias:      "t",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
		},
	}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"name": true},
		SortableFields:   map[string]bool{"name": true},
		FieldMap:         map[string]string{"name": "name"},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		searchConfig: config,
		appliedJoins: make(map[string]bool),
	}

	qb.applyWebsearchWithPrefix("he'll'o world", "t.", "search_vector")
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_ApplySortField_AllCases(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"name": true, "rel.name": true},
		SortableFields:   map[string]bool{"name": true, "rel.name": true},
		FieldMap:         map[string]string{"name": "name"},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	tests := []struct {
		name       string
		tableAlias string
		field      string
	}{
		{"with alias simple field", "t", "name"},
		{"no alias simple field", "", "name"},
		{"dotted field", "t", "rel.name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			query := db.NewSelect().ModelTableExpr("test AS t")
			qb := &QueryBuilder{
				query:        query,
				tableAlias:   tt.tableAlias,
				fieldConfig:  fieldConfig,
				appliedJoins: make(map[string]bool),
			}

			qb.applySortField(domaintypes.SortField{
				Field:     tt.field,
				Direction: dbtype.SortDirectionDesc,
			})
			assert.NotNil(t, qb.GetQuery())
		})
	}
}

func TestQueryBuilder_ResolveCustomRelationship_DefaultJoinType(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	entity := &customRelEntity{}
	config := entity.GetPostgresSearchConfig()

	query := db.NewSelect().ModelTableExpr("custom_rel_entities AS cre")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "cre",
		searchConfig: &config,
		entity:       entity,
		appliedJoins: make(map[string]bool),
	}

	sqlField, joinDefs := qb.resolveNestedField("emptyCustom.name")
	require.NotEmpty(t, sqlField)
	require.NotEmpty(t, joinDefs)
	assert.Equal(t, dbtype.JoinTypeLeft, joinDefs[0].JoinType)
}

func TestQueryBuilder_ResolveStandardRelationship_UnknownType(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	config := &domaintypes.PostgresSearchConfig{
		TableAlias: "t",
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:        "weird",
				Type:         dbtype.RelationshipType("Unknown"),
				TargetTable:  "weird_table",
				Alias:        "wt",
				ForeignKey:   "weird_id",
				ReferenceKey: "id",
				Queryable:    true,
			},
		},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		searchConfig: config,
		appliedJoins: make(map[string]bool),
	}

	sqlField, joinDefs := qb.resolveNestedField("weird.name")
	assert.Empty(t, sqlField)
	assert.Nil(t, joinDefs)
}

func TestQueryBuilder_ApplyDateRangeFilter_NonMapValue(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := New(query, "t", fieldConfig)

	qb.applyDateRangeFilter("t.created_at", "not-a-map")
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_ApplyFromDateFilter_NonFloatFrom(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := New(query, "t", fieldConfig)

	dateRange := map[string]any{"from": "not-a-float"}
	qb.applyFromDateFilter("t.created_at", dateRange)
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_ApplyToDateFilter_NonFloatTo(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := New(query, "t", fieldConfig)

	dateRange := map[string]any{"to": "not-a-float"}
	qb.applyToDateFilter("t.created_at", dateRange)
	assert.NotNil(t, qb.GetQuery())
}

func TestExtractDays_NestedMap(t *testing.T) {
	t.Parallel()

	result := extractDays(map[string]any{"days": float64(15)})
	assert.Equal(t, 15, result)
}

func TestExtractDays_MapWithoutDaysKey(t *testing.T) {
	t.Parallel()

	result := extractDays(map[string]any{"count": 10})
	assert.Equal(t, 0, result)
}

func TestExtractDays_String(t *testing.T) {
	t.Parallel()

	result := extractDays("7")
	assert.Equal(t, 0, result)
}

func TestQueryBuilder_ApplySort_InvalidFieldTracked(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &mockEntity{}
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"name": true},
		SortableFields:   map[string]bool{"name": true},
		FieldMap:         map[string]string{"name": "name"},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().Model((*mockEntity)(nil)).ModelTableExpr("mock_entities AS m")
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)

	qb.ApplySort([]domaintypes.SortField{
		{Field: "invalid_field", Direction: dbtype.SortDirectionAsc},
	})

	assert.True(t, qb.HasInvalidFields())
	assert.Contains(t, qb.GetInvalidFields(), "invalid_field")
}

func TestQueryBuilder_ApplySort_DottedFieldConversion(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &mockEntity{}
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"related_name.field": true},
		SortableFields:   map[string]bool{"related_name.field": true},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().Model((*mockEntity)(nil)).ModelTableExpr("mock_entities AS m")
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)

	qb.ApplySort([]domaintypes.SortField{
		{Field: "relatedName.field", Direction: dbtype.SortDirectionAsc},
	})

	assert.False(t, qb.HasInvalidFields())
}

func TestQueryBuilder_ApplyFilters_DottedFieldConversion(t *testing.T) {
	t.Parallel()

	ClearCaches()

	db := newAdditionalTestDB()
	entity := &mockEntity{}
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"related_name.field": true},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().Model((*mockEntity)(nil)).ModelTableExpr("mock_entities AS m")
	qb := NewWithPostgresSearch(query, "m", fieldConfig, entity)

	qb.ApplyFilters([]domaintypes.FieldFilter{
		{Field: "relatedName.field", Operator: dbtype.OpEqual, Value: "test"},
	})

	assert.False(t, qb.HasInvalidFields())
}

func TestQueryBuilder_BuildConditionStatement_DefaultCase(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := New(query, "t", fieldConfig)

	condition, args := qb.buildConditionStatement(dbtype.Operator("unknown"), "t.field", "value")
	assert.Empty(t, condition)
	assert.Nil(t, args)
}

func TestQueryBuilder_ApplySimpleSearch_EnumField(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{"status": true, "name": true},
		SortableFields:   map[string]bool{"status": true, "name": true},
		FieldMap:         map[string]string{"status": "status", "name": "name"},
		EnumMap:          map[string]bool{"status": true},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		searchConfig: nil,
		appliedJoins: make(map[string]bool),
	}

	qb.applySimpleSearch("test", []string{"status", "name"})
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_ConfigBasedSearch_CompositeField(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	config := &domaintypes.PostgresSearchConfig{
		TableAlias: "t",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "full_name", Type: domaintypes.FieldTypeComposite},
		},
	}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		searchConfig: config,
		appliedJoins: make(map[string]bool),
	}

	qb.applyConfigBasedSearch("john")
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_ConfigBasedSearch_DateField(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	config := &domaintypes.PostgresSearchConfig{
		TableAlias: "t",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "created_at", Type: domaintypes.FieldTypeDate},
		},
	}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		searchConfig: config,
		appliedJoins: make(map[string]bool),
	}

	qb.applyConfigBasedSearch("2024")
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_ConfigBasedSearch_BooleanField(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	config := &domaintypes.PostgresSearchConfig{
		TableAlias: "t",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "active", Type: domaintypes.FieldTypeBoolean},
		},
	}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		searchConfig: config,
		appliedJoins: make(map[string]bool),
	}

	qb.applyConfigBasedSearch("true")
	assert.NotNil(t, qb.GetQuery())
}

func TestQueryBuilder_ApplyPostgresTextSearch_NilConfig(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		searchConfig: nil,
		appliedJoins: make(map[string]bool),
	}

	result := qb.applyPostgresTextSearch("test")
	assert.NotNil(t, result)
}

func TestQueryBuilder_ApplyPostgresTextSearch_EmptyQuery(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	config := &domaintypes.PostgresSearchConfig{
		TableAlias:      "t",
		UseSearchVector: true,
	}

	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields: map[string]bool{},
		SortableFields:   map[string]bool{},
		FieldMap:         map[string]string{},
		EnumMap:          map[string]bool{},
		NestedFields:     map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := &QueryBuilder{
		query:        query,
		tableAlias:   "t",
		fieldConfig:  fieldConfig,
		searchConfig: config,
		appliedJoins: make(map[string]bool),
	}

	result := qb.applyPostgresTextSearch("")
	assert.NotNil(t, result)
}

func TestQueryBuilder_ApplyGeoFilters_RadiusConversion(t *testing.T) {
	t.Parallel()

	db := newAdditionalTestDB()
	fieldConfig := &domaintypes.FieldConfiguration{
		FilterableFields:    map[string]bool{"location": true},
		SortableFields:      map[string]bool{},
		GeoFilterableFields: map[string]bool{"location": true},
		FieldMap:            map[string]string{"location": "location"},
		EnumMap:             map[string]bool{},
		NestedFields:        map[string]domaintypes.NestedFieldDefintion{},
	}

	query := db.NewSelect().ModelTableExpr("test AS t")
	qb := New(query, "t", fieldConfig)

	qb.ApplyGeoFilters([]domaintypes.GeoFilter{
		{
			Field: "location",
			Center: domaintypes.GeoPoint{
				Latitude:  40.7128,
				Longitude: -74.0060,
			},
			RadiusKm: 10.5,
		},
	})

	assert.NotNil(t, qb.GetQuery())
}
