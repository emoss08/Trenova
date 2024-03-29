package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DelayCode holds the schema definition for the DelayCode entity.
type DelayCode struct {
	ent.Schema
}

// Fields of the DelayCode.
func (DelayCode) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("code").
			NotEmpty().
			MaxLen(4).
			StructTag(`json:"code" validate:"required,max=4"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
		field.Bool("f_carrier_or_driver").
			Optional().
			StructTag(`json:"fCarrierOrDriver" validate:"omitempty"`),
	}
}

// Edges of the DelayCode.
func (DelayCode) Edges() []ent.Edge {
	return nil
}

// Indexes of the DelayCode.
func (DelayCode) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure the code is unique for the organization.
		index.Fields("code", "organization_id").
			Unique(),
	}
}

// Mixin of the DelayCode.
func (DelayCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
