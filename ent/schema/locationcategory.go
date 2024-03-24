package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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

// Indexes of the LocationCategory.
func (LocationCategory) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure the code is unique for the organization.
		index.Fields("name", "organization_id").
			Unique(),
	}
}

// Mixin of the LocationCategory.
func (LocationCategory) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
