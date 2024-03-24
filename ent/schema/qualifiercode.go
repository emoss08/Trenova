package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// QualifierCode holds the schema definition for the QualifierCode entity.
type QualifierCode struct {
	ent.Schema
}

// Fields of the QualifierCode.
func (QualifierCode) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("code").
			NotEmpty().
			MaxLen(10).
			StructTag(`json:"code" validate:"required,max=10"`),
		field.Text("description").
			NotEmpty().
			StructTag(`json:"description" validate:"omitempty"`),
	}
}

// Mixin of the QualifierCode.
func (QualifierCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Indexes of the QualifierCode.
func (QualifierCode) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure the code is unique for the organization.
		index.Fields("code", "organization_id").
			Unique(),
	}
}

// Edges of the QualifierCode.
func (QualifierCode) Edges() []ent.Edge {
	return nil
}
