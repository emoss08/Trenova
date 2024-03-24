package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ServiceType holds the schema definition for the ServiceType entity.
type ServiceType struct {
	ent.Schema
}

// Fields of the ServiceType.
func (ServiceType) Fields() []ent.Field {
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

// Mixin of the ServiceType.
func (ServiceType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Indexes of the ServiceType.
func (ServiceType) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure the code is unique for the organization.
		index.Fields("code", "organization_id").
			Unique(),
	}
}

// Edges of the ServiceType.
func (ServiceType) Edges() []ent.Edge {
	return nil
}
