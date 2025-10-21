package querybuilder

import (
	"fmt"
	"maps"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/utils"
	"go.uber.org/zap"
)

type FieldConfigBuilder[T domaintypes.PostgresSearchable] struct {
	config *pagination.FieldConfiguration
	entity T
	debug  bool
	logger *zap.Logger
}

func NewFieldConfigBuilder[T domaintypes.PostgresSearchable](entity T) *FieldConfigBuilder[T] {
	return &FieldConfigBuilder[T]{
		config: &pagination.FieldConfiguration{
			FilterableFields: make(map[string]bool),
			SortableFields:   make(map[string]bool),
			FieldMap:         make(map[string]string),
			EnumMap:          make(map[string]bool),
			NestedFields:     make(map[string]pagination.NestedFieldDefinition),
		},
		entity: entity,
		logger: zap.NewNop(),
	}
}

func (b *FieldConfigBuilder[T]) WithAutoMapping() *FieldConfigBuilder[T] {
	fieldMap := ExtractFieldsFromStruct(b.entity)
	maps.Copy(b.config.FieldMap, fieldMap)
	return b
}

func (b *FieldConfigBuilder[T]) WithFilterableFields(fields ...string) *FieldConfigBuilder[T] {
	for _, field := range fields {
		b.config.FilterableFields[field] = true
		if _, exists := b.config.FieldMap[field]; !exists {
			b.config.FieldMap[field] = utils.ConvertCamelToSnake(field)
		}
	}
	return b
}

func (b *FieldConfigBuilder[T]) WithSortableFields(fields ...string) *FieldConfigBuilder[T] {
	for _, field := range fields {
		b.config.SortableFields[field] = true
		if _, exists := b.config.FieldMap[field]; !exists {
			b.config.FieldMap[field] = utils.ConvertCamelToSnake(field)
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

func (b *FieldConfigBuilder[T]) WithEnumFields(fields ...string) *FieldConfigBuilder[T] {
	for _, field := range fields {
		b.config.EnumMap[field] = true
	}
	return b
}

func (b *FieldConfigBuilder[T]) WithAutoEnumDetection() *FieldConfigBuilder[T] {
	config := b.entity.GetPostgresSearchConfig()

	for _, field := range config.SearchableFields {
		if field.Type == domaintypes.FieldTypeEnum {
			jsonName := utils.ConvertSnakeToCamel(field.Name)
			b.config.EnumMap[jsonName] = true
			if b.debug {
				b.logger.Debug("Detected enum field",
					zap.String("field", jsonName),
					zap.String("db_field", field.Name),
				)
			}
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
	joins ...pagination.JoinDefinition,
) *FieldConfigBuilder[T] {
	b.config.NestedFields[apiField] = pagination.NestedFieldDefinition{
		DatabaseField: dbField,
		RequiredJoins: joins,
		IsEnum:        false,
	}
	b.config.FilterableFields[apiField] = true
	b.config.SortableFields[apiField] = true
	return b
}

func (b *FieldConfigBuilder[T]) WithLogger(logger *zap.Logger) *FieldConfigBuilder[T] {
	b.logger = logger
	return b
}

func (b *FieldConfigBuilder[T]) WithDebug() *FieldConfigBuilder[T] {
	b.debug = true
	return b
}

func (b *FieldConfigBuilder[T]) WithRelationshipFields() *FieldConfigBuilder[T] {
	config := b.entity.GetPostgresSearchConfig()

	for _, rel := range config.Relationships {
		if rel.Queryable && rel.TargetEntity != nil {
			relationshipField := utils.ConvertSnakeToCamel(rel.Field)

			targetFields := ExtractFieldsFromStruct(rel.TargetEntity)

			for jsonField := range targetFields {
				path := fmt.Sprintf("%s.%s", relationshipField, jsonField)
				b.config.FilterableFields[path] = true
				b.config.SortableFields[path] = true

				if b.debug {
					b.logger.Debug("Added relationship field",
						zap.String("path", path),
						zap.String("relationship", relationshipField),
						zap.String("field", jsonField),
					)
				}
			}
		}
	}
	return b
}

func (b *FieldConfigBuilder[T]) Build() *pagination.FieldConfiguration {
	if b.debug {
		b.logger.Debug("Building field configuration",
			zap.Any("filterable_fields", b.config.FilterableFields),
			zap.Any("sortable_fields", b.config.SortableFields),
			zap.Any("field_map", b.config.FieldMap),
			zap.Any("enum_map", b.config.EnumMap),
			zap.Int("nested_fields_count", len(b.config.NestedFields)),
		)
	}
	return b.config
}
