package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// HazardousMaterial holds the schema definition for the HazardousMaterial entity.
type HazardousMaterial struct {
	ent.Schema
}

// Fields of the HazardousMaterial.
func (HazardousMaterial) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			MaxLen(100).
			NotEmpty().
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
			StructTag(`json:"hazardClass"`),
		field.String("erg_number").
			Optional().
			Nillable().
			MaxLen(255).
			StructTag(`json:"ergNumber"`),
		field.Text("description").
			Optional().
			Nillable().
			StructTag(`json:"description"`),
		field.Enum("packing_group").
			Optional().
			Nillable().
			Values("PackingGroupI",
				"PackingGroupII",
				"PackingGroupIII").
			StructTag(`json:"packingGroup"`),
		field.Text("proper_shipping_name").
			Optional().
			Nillable().
			StructTag(`json:"properShippingName"`),
	}
}

// Edges of the HazardousMaterial.
func (HazardousMaterial) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("commodities", Commodity.Type).
			StorageKey(edge.Column("hazardous_material_id")),
	}
}

// Mixin of the HazardousMaterial.
func (HazardousMaterial) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
