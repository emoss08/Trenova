package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
)

// LocationCategory holds the schema definition for the LocationCategory entity.
type LocationCategory struct {
	ent.Schema
}

// Fields of the LocationCategory.
func (LocationCategory) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(100)",
				dialect.SQLite:   "VARCHAR(100)",
			}).
			StructTag(`json:"name" validate:"required"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
		field.String("color").
			Optional().
			StructTag(`json:"color" validate:"omitempty"`),
	}
}

// Edges of the LocationCategory.
func (LocationCategory) Edges() []ent.Edge {
	return nil
}

// Mixin of the LocationCategory.
func (LocationCategory) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
