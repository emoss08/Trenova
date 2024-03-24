package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ChargeType holds the schema definition for the ChargeType entity.
type ChargeType struct {
	ent.Schema
}

// Fields of the ChargeType.
func (ChargeType) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("name").
			MaxLen(50).
			NotEmpty().
			StructTag(`json:"name" validate:"required,max=50"`),
		field.String("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
	}
}

// Edges of the ChargeType.
func (ChargeType) Edges() []ent.Edge {
	return nil
}

// Mixin for the ChargeType.
func (ChargeType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Indexes of the ChargeType.
func (ChargeType) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name", "organization_id").
			Unique(),
	}
}
