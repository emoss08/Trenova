package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ShipmentType holds the schema definition for the ShipmentType entity.
type ShipmentType struct {
	ent.Schema
}

// Fields of the ShipmentType.
func (ShipmentType) Fields() []ent.Field {
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
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
	}
}

// Edges of the ShipmentType.
func (ShipmentType) Edges() []ent.Edge {
	return nil
}

// Mixin of the ShipmentType.
func (ShipmentType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Indexes of the ShipmentType.
func (ShipmentType) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure the code is unique for the organization.
		index.Fields("code", "organization_id").
			Unique(),
	}
}
