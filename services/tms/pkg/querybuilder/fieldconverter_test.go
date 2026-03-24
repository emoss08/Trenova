package querybuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractDBFieldName_EmptyBunTag(t *testing.T) {
	result := extractDBFieldName("", "firstName")
	assert.Equal(t, "first_name", result)
}

func TestExtractDBFieldName_DashBunTag(t *testing.T) {
	result := extractDBFieldName("-", "firstName")
	assert.Equal(t, "first_name", result)
}

func TestExtractDBFieldName_SimpleColumnName(t *testing.T) {
	result := extractDBFieldName("first_name", "firstName")
	assert.Equal(t, "first_name", result)
}

func TestExtractDBFieldName_ColumnNameWithOptions(t *testing.T) {
	result := extractDBFieldName("first_name,notnull", "firstName")
	assert.Equal(t, "first_name", result)
}

func TestExtractDBFieldName_PrimaryKey(t *testing.T) {
	result := extractDBFieldName("id,pk", "id")
	assert.Equal(t, "id", result)
}

func TestExtractDBFieldName_TypePrefix(t *testing.T) {
	result := extractDBFieldName("type:uuid,pk", "id")
	assert.Equal(t, "id", result)
}

func TestExtractDBFieldName_ComplexTag(t *testing.T) {
	result := extractDBFieldName("column_name,type:varchar(255),notnull,unique", "fieldName")
	assert.Equal(t, "column_name", result)
}

func TestExtractDBFieldName_OnlyModifiers(t *testing.T) {
	result := extractDBFieldName("id,pk,notnull,unique", "id")
	assert.Equal(t, "id", result)
}

func TestExtractDBFieldName_DefaultModifier(t *testing.T) {
	result := extractDBFieldName("status,default:'active'", "status")
	assert.Equal(t, "status", result)
}

func TestExtractFieldsFromStruct_EmbeddedFields(t *testing.T) {
	ClearCaches()

	type BaseModel struct {
		ID        string `json:"id"        bun:"id,pk"`
		CreatedAt int64  `json:"createdAt" bun:"created_at"`
	}

	type ExtendedModel struct {
		BaseModel
		Name string `json:"name" bun:"name"`
	}

	fields := ExtractFieldsFromStruct(&ExtendedModel{})

	assert.Equal(t, "name", fields["name"])
	assert.NotContains(t, fields, "id")
	assert.NotContains(t, fields, "createdAt")
}

func TestExtractFieldsFromStruct_PointerFields(t *testing.T) {
	ClearCaches()

	type ModelWithPointers struct {
		ID   string  `json:"id"   bun:"id"`
		Name *string `json:"name" bun:"name"`
	}

	fields := ExtractFieldsFromStruct(&ModelWithPointers{})

	assert.Equal(t, "id", fields["id"])
	assert.Equal(t, "name", fields["name"])
}

func TestExtractFieldsFromStruct_OmitemptyTag(t *testing.T) {
	ClearCaches()

	type ModelWithOmitempty struct {
		ID   string `json:"id"             bun:"id"`
		Name string `json:"name,omitempty" bun:"name"`
	}

	fields := ExtractFieldsFromStruct(&ModelWithOmitempty{})

	assert.Equal(t, "id", fields["id"])
	assert.Equal(t, "name", fields["name"])
}

func TestExtractFieldsFromStruct_UnexportedFields(t *testing.T) {
	ClearCaches()

	type ModelWithUnexported struct {
		ID       string `bun:"id"       json:"id"`
		internal string `bun:"internal"`
	}

	fields := ExtractFieldsFromStruct(&ModelWithUnexported{})

	assert.Equal(t, "id", fields["id"])
	assert.NotContains(t, fields, "internal")
}

func TestExtractFieldsFromStruct_NoJsonTag(t *testing.T) {
	ClearCaches()

	type ModelWithoutJSON struct {
		ID   string `json:"id" bun:"id"`
		Name string `          bun:"name"`
	}

	fields := ExtractFieldsFromStruct(&ModelWithoutJSON{})

	assert.Equal(t, "id", fields["id"])
	assert.NotContains(t, fields, "Name")
	assert.NotContains(t, fields, "name")
}

func TestExtractFieldsFromStruct_Slice(t *testing.T) {
	ClearCaches()

	type Model struct {
		ID string `json:"id" bun:"id"`
	}

	fields := ExtractFieldsFromStruct([]Model{})
	assert.Empty(t, fields)
}

func TestExtractFieldsFromStruct_Map(t *testing.T) {
	fields := ExtractFieldsFromStruct(map[string]string{})
	assert.Empty(t, fields)
}

func TestExtractFieldsFromStruct_Nil(t *testing.T) {
	fields := ExtractFieldsFromStruct(nil)
	assert.Empty(t, fields)
}

func TestWarmFieldCache_MultipleEntities(t *testing.T) {
	ClearCaches()

	type Entity1 struct {
		ID string `json:"id" bun:"id"`
	}
	type Entity2 struct {
		Name string `json:"name" bun:"name"`
	}

	WarmFieldCache(&Entity1{}, &Entity2{})

	fields1 := ExtractFieldsFromStruct(&Entity1{})
	fields2 := ExtractFieldsFromStruct(&Entity2{})

	assert.Equal(t, "id", fields1["id"])
	assert.Equal(t, "name", fields2["name"])
}

func TestExtractFieldsFromStruct_ComplexBunTags(t *testing.T) {
	ClearCaches()

	type ComplexModel struct {
		ID          string `json:"id"          bun:"id,pk,type:uuid,default:gen_random_uuid()"`
		Name        string `json:"name"        bun:"name,notnull,unique"`
		Status      string `json:"status"      bun:"status,type:status_enum,default:'pending'"`
		Description string `json:"description" bun:"description,type:text,nullzero"`
	}

	fields := ExtractFieldsFromStruct(&ComplexModel{})

	assert.Equal(t, "id", fields["id"])
	assert.Equal(t, "name", fields["name"])
	assert.Equal(t, "status", fields["status"])
	assert.Equal(t, "description", fields["description"])
}
