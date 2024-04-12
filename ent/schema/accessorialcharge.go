package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
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
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(1)",
				dialect.SQLite:   "VARCHAR(1)",
			}).
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("code").
			NotEmpty().
			MaxLen(4).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(4)",
				dialect.SQLite:   "VARCHAR(4)",
			}).
			StructTag(`json:"code" validate:"required,max=4"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty,max=100"`),
		field.Bool("is_detention").
			Default(false).
			StructTag(`json:"isDetention" validate:"omitempty"`),
		field.Enum("method").
			Values("Distance",
				"Flat",
				"Percentage").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			StructTag(`json:"method" validate:"required,oneof=Distance Flat Percentage"`),
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

// Edges of the AccessorialCharge.
func (AccessorialCharge) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("shipment_charges", ShipmentCharges.Type).
			StructTag(`json:"shipmentCharges,omitempty"`),
	}
}
