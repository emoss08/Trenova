package querybuilder

import (
	"testing"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
)

type builderTestEntity struct {
	ID          string `json:"id"          bun:"id,pk"`
	Name        string `json:"name"        bun:"name"`
	Status      string `json:"status"      bun:"status"`
	Description string `json:"description" bun:"description"`
}

func (b *builderTestEntity) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "bte",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "description", Type: domaintypes.FieldTypeText},
		},
	}
}

func (b *builderTestEntity) GetTableName() string {
	return "builder_test_entities"
}

type relatedEntity struct {
	ID   string `json:"id"   bun:"id,pk"`
	Code string `json:"code" bun:"code"`
	Name string `json:"name" bun:"name"`
}

type entityWithRelationship struct {
	ID        string `json:"id"        bun:"id,pk"`
	Name      string `json:"name"      bun:"name"`
	RelatedID string `json:"relatedId" bun:"related_id"`
}

func (e *entityWithRelationship) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "ewr",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:        "related",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: &relatedEntity{},
				TargetTable:  "related_entities",
				ForeignKey:   "related_id",
				ReferenceKey: "id",
				Alias:        "rel",
				Queryable:    true,
			},
		},
	}
}

func (e *entityWithRelationship) GetTableName() string {
	return "entities_with_relationship"
}

func TestNewFieldConfigBuilder(t *testing.T) {
	entity := &builderTestEntity{}
	builder := NewFieldConfigBuilder(entity)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.config)
	assert.NotNil(t, builder.config.FilterableFields)
	assert.NotNil(t, builder.config.SortableFields)
	assert.NotNil(t, builder.config.FieldMap)
	assert.NotNil(t, builder.config.EnumMap)
	assert.NotNil(t, builder.config.NestedFields)
}

func TestFieldConfigBuilder_WithAutoMapping(t *testing.T) {
	ClearCaches()

	entity := &builderTestEntity{}
	config := NewFieldConfigBuilder(entity).
		WithAutoMapping().
		Build()

	assert.Equal(t, "id", config.FieldMap["id"])
	assert.Equal(t, "name", config.FieldMap["name"])
	assert.Equal(t, "status", config.FieldMap["status"])
	assert.Equal(t, "description", config.FieldMap["description"])
}

func TestFieldConfigBuilder_WithFilterableField(t *testing.T) {
	entity := &builderTestEntity{}
	config := NewFieldConfigBuilder(entity).
		WithFilterableField("name", "status").
		Build()

	assert.True(t, config.FilterableFields["name"])
	assert.True(t, config.FilterableFields["status"])
	assert.False(t, config.FilterableFields["id"])
	assert.False(t, config.FilterableFields["description"])

	assert.Equal(t, "name", config.FieldMap["name"])
	assert.Equal(t, "status", config.FieldMap["status"])
}

func TestFieldConfigBuilder_WithSortableField(t *testing.T) {
	entity := &builderTestEntity{}
	config := NewFieldConfigBuilder(entity).
		WithSortableField("name", "id").
		Build()

	assert.True(t, config.SortableFields["name"])
	assert.True(t, config.SortableFields["id"])
	assert.False(t, config.SortableFields["status"])

	assert.Equal(t, "name", config.FieldMap["name"])
	assert.Equal(t, "id", config.FieldMap["id"])
}

func TestFieldConfigBuilder_WithAllFieldsFilterable(t *testing.T) {
	ClearCaches()

	entity := &builderTestEntity{}
	config := NewFieldConfigBuilder(entity).
		WithAutoMapping().
		WithAllFieldsFilterable().
		Build()

	assert.True(t, config.FilterableFields["id"])
	assert.True(t, config.FilterableFields["name"])
	assert.True(t, config.FilterableFields["status"])
	assert.True(t, config.FilterableFields["description"])
}

func TestFieldConfigBuilder_WithAllFieldsSortable(t *testing.T) {
	ClearCaches()

	entity := &builderTestEntity{}
	config := NewFieldConfigBuilder(entity).
		WithAutoMapping().
		WithAllFieldsSortable().
		Build()

	assert.True(t, config.SortableFields["id"])
	assert.True(t, config.SortableFields["name"])
	assert.True(t, config.SortableFields["status"])
	assert.True(t, config.SortableFields["description"])
}

func TestFieldConfigBuilder_WithEnumField(t *testing.T) {
	entity := &builderTestEntity{}
	config := NewFieldConfigBuilder(entity).
		WithEnumField("status", "type").
		Build()

	assert.True(t, config.EnumMap["status"])
	assert.True(t, config.EnumMap["type"])
	assert.False(t, config.EnumMap["name"])
}

func TestFieldConfigBuilder_WithAutoEnumDetection(t *testing.T) {
	ClearCaches()

	entity := &builderTestEntity{}
	config := NewFieldConfigBuilder(entity).
		WithAutoEnumDetection().
		Build()

	assert.True(t, config.EnumMap["status"])
	assert.False(t, config.EnumMap["name"])
	assert.False(t, config.EnumMap["description"])
}

func TestFieldConfigBuilder_WithNestedField(t *testing.T) {
	entity := &builderTestEntity{}

	joinStep := domaintypes.JoinStep{
		Table:     "related_table",
		Alias:     "rt",
		Condition: "bte.related_id = rt.id",
		JoinType:  dbtype.JoinTypeLeft,
	}

	config := NewFieldConfigBuilder(entity).
		WithNestedField("related.name", "rt.name", joinStep).
		Build()

	nestedDef, exists := config.NestedFields["related.name"]
	assert.True(t, exists)
	assert.Equal(t, "rt.name", nestedDef.DatabaseField)
	assert.Len(t, nestedDef.RequiredJoins, 1)
	assert.Equal(t, "related_table", nestedDef.RequiredJoins[0].Table)

	assert.True(t, config.FilterableFields["related.name"])
	assert.True(t, config.SortableFields["related.name"])
}

func TestFieldConfigBuilder_WithRelationshipFields(t *testing.T) {
	ClearCaches()

	entity := &entityWithRelationship{}
	config := NewFieldConfigBuilder(entity).
		WithRelationshipFields().
		Build()

	assert.True(t, config.FilterableFields["related.id"])
	assert.True(t, config.FilterableFields["related.code"])
	assert.True(t, config.FilterableFields["related.name"])

	assert.True(t, config.SortableFields["related.id"])
	assert.True(t, config.SortableFields["related.code"])
	assert.True(t, config.SortableFields["related.name"])
}

func TestFieldConfigBuilder_Chaining(t *testing.T) {
	ClearCaches()

	entity := &builderTestEntity{}
	config := NewFieldConfigBuilder(entity).
		WithAutoMapping().
		WithAllFieldsFilterable().
		WithAllFieldsSortable().
		WithAutoEnumDetection().
		Build()

	assert.True(t, config.FilterableFields["name"])
	assert.True(t, config.SortableFields["name"])
	assert.True(t, config.EnumMap["status"])
	assert.Equal(t, "name", config.FieldMap["name"])
}

func TestFieldConfigBuilder_Build(t *testing.T) {
	entity := &builderTestEntity{}
	builder := NewFieldConfigBuilder(entity)

	config1 := builder.Build()
	config2 := builder.Build()

	assert.Same(t, config1, config2, "Build should return same config instance")
}

func TestFieldConfigBuilder_EmptyEntity(t *testing.T) {
	type emptyEntity struct{}

	entity := &emptyEntity{}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Should not panic with empty entity: %v", r)
		}
	}()

	_ = ExtractFieldsFromStruct(entity)
}

func TestFieldConfigBuilder_WithFilterableField_AutoConvertsToSnakeCase(t *testing.T) {
	entity := &builderTestEntity{}
	config := NewFieldConfigBuilder(entity).
		WithFilterableField("firstName", "lastName").
		Build()

	assert.Equal(t, "first_name", config.FieldMap["firstName"])
	assert.Equal(t, "last_name", config.FieldMap["lastName"])
}

func TestFieldConfigBuilder_WithSortableField_AutoConvertsToSnakeCase(t *testing.T) {
	entity := &builderTestEntity{}
	config := NewFieldConfigBuilder(entity).
		WithSortableField("createdAt", "updatedAt").
		Build()

	assert.Equal(t, "created_at", config.FieldMap["createdAt"])
	assert.Equal(t, "updated_at", config.FieldMap["updatedAt"])
}

func TestFieldConfigBuilder_WithAutoMapping_DoesNotOverwrite(t *testing.T) {
	ClearCaches()

	entity := &builderTestEntity{}
	config := NewFieldConfigBuilder(entity).
		WithFilterableField("customField").
		WithAutoMapping().
		Build()

	assert.Equal(t, "custom_field", config.FieldMap["customField"])
	assert.Equal(t, "name", config.FieldMap["name"])
}
