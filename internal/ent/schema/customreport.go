package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// CustomReport holds the schema definition for the CustomReport entity.
type CustomReport struct {
	ent.Schema
}

// Fields of the CustomReport.
func (CustomReport) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			StructTag(`json:"name" validate:"required"`),
		field.String("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
		field.String("table").
			Optional().
			StructTag(`json:"table" validate:"omitempty"`),
	}
}

// Mixin of the CustomReport.
func (CustomReport) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the CustomReport.
func (CustomReport) Edges() []ent.Edge {
	return nil
}
