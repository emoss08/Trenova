package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Commodity holds the schema definition for the Commodity entity.
type Commodity struct {
	ent.Schema
}

// Fields of the Commodity.
func (Commodity) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("name").
			MaxLen(100).
			NotEmpty().
			StructTag(`json:"name" validate:"required"`),
		field.Bool("is_hazmat").
			Default(false).
			StructTag(`json:"isHazmat" validate:"omitempty"`),
		field.String("unit_of_measure").
			Optional().
			StructTag(`json:"unitOfMeasure" validate:"omitempty,oneof=Pallet Tote Drum Cylinder Case Ampule Bag Bottle Pail Pieces IsoTank"`),
		field.Int("min_temp").
			Optional().
			StructTag(`json:"minTemp" validate:"omitempty,max=150"`),
		field.Int("max_temp").
			Optional().
			StructTag(`json:"maxTemp" validate:"omitempty,max=150"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
		field.UUID("hazardous_material_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"hazardousMaterialId" validate:"omitempty"`),
	}
}

// Edges of the Commodity.
func (Commodity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("hazardous_material", HazardousMaterial.Type).
			Field("hazardous_material_id").
			Annotations(entsql.OnDelete(entsql.Restrict)).
			StructTag(`json:"hazardousMaterial"`).
			Unique(),
	}
}

// Mixin of the Commodity.
func (Commodity) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
