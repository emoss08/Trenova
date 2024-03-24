package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AccessorialCharge holds the schema definition for the AccessorialCharge entity.
type AccessorialCharge struct {
	ent.Schema
}

// Fields of the AccessorialCharge.
func (AccessorialCharge) Fields() []ent.Field {
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
			MaxLen(100).
			StructTag(`json:"description" validate:"omitempty,max=100"`),
		field.Bool("is_detention").
			Default(false).
			StructTag(`json:"isDetention" validate:"omitempty"`),
		field.Enum("method").
			Values("Distance",
				"Flat",
				"Percentage"),
		field.Float("amount").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(19,4)",
				dialect.Postgres: "numeric(19,4)",
			}).
			Default(0.0).
			StructTag(`json:"amount" validate:"required,gt=0"`),
	}
}

// Mixin of the AccessorialCharge.
func (AccessorialCharge) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Indexes of the AccessorialCharge.
func (AccessorialCharge) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure the code is unique for the organization.
		index.Fields("code", "organization_id").
			Unique(),
	}
}

// Edges of the AccessorialCharge.
func (AccessorialCharge) Edges() []ent.Edge {
	return nil
}
