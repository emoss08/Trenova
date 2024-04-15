package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
)

// HazardousMaterial holds the schema definition for the HazardousMaterial entity.
type HazardousMaterial struct {
	ent.Schema
}

// Fields of the HazardousMaterial.
func (HazardousMaterial) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("name").
			MaxLen(100).
			NotEmpty().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(100)",
				dialect.SQLite:   "VARCHAR(100)",
			}).
			StructTag(`json:"name"`),
		field.Enum("hazard_class").
			Values("HazardClass1And1",
				"HazardClass1And2",
				"HazardClass1And3",
				"HazardClass1And4",
				"HazardClass1And5",
				"HazardClass1And6",
				"HazardClass2And1",
				"HazardClass2And2",
				"HazardClass2And3",
				"HazardClass3",
				"HazardClass4And1",
				"HazardClass4And2",
				"HazardClass4And3",
				"HazardClass5And1",
				"HazardClass5And2",
				"HazardClass6And1",
				"HazardClass6And2",
				"HazardClass7",
				"HazardClass8",
				"HazardClass9").
			Default("HazardClass1And1").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(16)",
				dialect.SQLite:   "VARCHAR(16)",
			}).
			StructTag(`json:"hazardClass" validate:"required"`),
		field.String("erg_number").
			Optional().
			StructTag(`json:"ergNumber" validate:"omitempty"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
		field.String("packing_group").
			Optional().
			StructTag(`json:"packingGroup" validate:"omitempty"`),
		field.Text("proper_shipping_name").
			Optional().
			StructTag(`json:"properShippingName"`),
	}
}

// Edges of the HazardousMaterial.
func (HazardousMaterial) Edges() []ent.Edge {
	return nil
}

// Mixin of the HazardousMaterial.
func (HazardousMaterial) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
