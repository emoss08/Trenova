package querybuilder

import (
	"fmt"
	"maps"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

type FieldConfigBuilder[T domaintypes.PostgresSearchable] struct {
	config *domaintypes.FieldConfiguration
	entity T
}

func NewFieldConfigBuilder[T domaintypes.PostgresSearchable](entity T) *FieldConfigBuilder[T] {
	return &FieldConfigBuilder[T]{
		config: &domaintypes.FieldConfiguration{
			FilterableFields:    make(map[string]bool),
			SortableFields:      make(map[string]bool),
			GeoFilterableFields: make(map[string]bool),
			FieldMap:            make(map[string]string),
			EnumMap:             make(map[string]bool),
			NestedFields:        make(map[string]domaintypes.NestedFieldDefintion),
		},
		entity: entity,
	}
}

func (b *FieldConfigBuilder[T]) WithAutoMapping() *FieldConfigBuilder[T] {
	fieldMap := ExtractFieldsFromStruct(b.entity)
	maps.Copy(b.config.FieldMap, fieldMap)
	return b
}

func (b *FieldConfigBuilder[T]) WithFilterableField(fields ...string) *FieldConfigBuilder[T] {
	for _, field := range fields {
		b.config.FilterableFields[field] = true
		if _, exists := b.config.FieldMap[field]; !exists {
			b.config.FieldMap[field] = stringutils.ConvertCamelToSnake(field)
		}
	}

	return b
}

func (b *FieldConfigBuilder[T]) WithSortableField(fields ...string) *FieldConfigBuilder[T] {
	for _, field := range fields {
		b.config.SortableFields[field] = true
		if _, exists := b.config.FieldMap[field]; !exists {
			b.config.FieldMap[field] = stringutils.ConvertCamelToSnake(field)
		}
	}

	return b
}

func (b *FieldConfigBuilder[T]) WithAllFieldsFilterable() *FieldConfigBuilder[T] {
	for field := range b.config.FieldMap {
		b.config.FilterableFields[field] = true
	}

	return b
}

func (b *FieldConfigBuilder[T]) WithAllFieldsSortable() *FieldConfigBuilder[T] {
	for field := range b.config.FieldMap {
		b.config.SortableFields[field] = true
	}

	return b
}

func (b *FieldConfigBuilder[T]) WithEnumField(fields ...string) *FieldConfigBuilder[T] {
	for _, field := range fields {
		b.config.EnumMap[field] = true
	}

	return b
}

func (b *FieldConfigBuilder[T]) WithAutoEnumDetection() *FieldConfigBuilder[T] {
	config := b.entity.GetPostgresSearchConfig()

	for _, field := range config.SearchableFields {
		if field.Type == domaintypes.FieldTypeEnum {
			jsonName := stringutils.ConvertCamelToSnake(field.Name)
			b.config.EnumMap[jsonName] = true
			if _, exists := b.config.FieldMap[jsonName]; !exists {
				b.config.FieldMap[jsonName] = field.Name
			}
		}
	}

	return b
}

func (b *FieldConfigBuilder[T]) WithNestedField(
	apiField string,
	dbField string,
	joins ...domaintypes.JoinStep,
) *FieldConfigBuilder[T] {
	b.config.NestedFields[apiField] = domaintypes.NestedFieldDefintion{
		DatabaseField: dbField,
		RequiredJoins: joins,
		IsEnum:        false,
	}
	b.config.FilterableFields[apiField] = true
	b.config.SortableFields[apiField] = true

	return b
}

func (b *FieldConfigBuilder[T]) WithRelationshipFields() *FieldConfigBuilder[T] {
	config := b.entity.GetPostgresSearchConfig()

	for _, rel := range config.Relationships {
		if rel.Queryable && rel.TargetEntity != nil {
			relationshipField := stringutils.ConvertCamelToSnake(rel.Field)

			targetFields := ExtractFieldsFromStruct(rel.TargetEntity)

			for jsonField := range targetFields {
				path := fmt.Sprintf("%s.%s", relationshipField, jsonField)
				b.config.FilterableFields[path] = true
				b.config.SortableFields[path] = true
			}
		}
	}

	return b
}

func (b *FieldConfigBuilder[T]) WithGeoFilterableField(fields ...string) *FieldConfigBuilder[T] {
	for _, field := range fields {
		b.config.GeoFilterableFields[field] = true
		if _, exists := b.config.FieldMap[field]; !exists {
			b.config.FieldMap[field] = stringutils.ConvertCamelToSnake(field)
		}
	}

	return b
}

func (b *FieldConfigBuilder[T]) Build() *domaintypes.FieldConfiguration {
	return b.config
}
