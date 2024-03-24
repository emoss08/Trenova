package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CommentType holds the schema definition for the CommentType entity.
type CommentType struct {
	ent.Schema
}

// Fields of the CommentType.
func (CommentType) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("name").
			NotEmpty().
			MaxLen(10).
			StructTag(`json:"name" validate:"required,max=10"`),
		field.Enum("severity").
			Values("High", "Medium", "Low").
			Default("Low").
			StructTag(`json:"severity" validate:"required,oneof=High Medium Low"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
	}
}

// Edges of the CommentType.
func (CommentType) Edges() []ent.Edge {
	return nil
}

// Indexes of the CommentType.
func (CommentType) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure the code is unique for the organization.
		index.Fields("name", "organization_id").
			Unique(),
	}
}

// Mixin of the CommentType.
func (CommentType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
