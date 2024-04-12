package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ShipmentCommodity holds the schema definition for the ShipmentCommodity entity.
type ShipmentCommodity struct {
	ent.Schema
}

// Fields of the ShipmentCommodity.
func (ShipmentCommodity) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("shipment_id", uuid.UUID{}).
			Immutable().
			StructTag(`json:"shipmentId" validate:"omitempty"`), // Shipment ID will be set by the system.
		field.UUID("commodity_id", uuid.UUID{}).
			StructTag(`json:"commodityId" validate:"required"`),
		field.UUID("hazardous_material_id", uuid.UUID{}).
			StructTag(`json:"hazardousMaterialId" validate:"omitempty"`),
		field.Float("sub_total").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(10,2)",
				dialect.Postgres: "numeric(10,2)",
			}).
			Positive().
			StructTag(`json:"subTotal" validate:"required"`),
		field.Bool("placard_needed").
			Default(false).
			StructTag(`json:"placardNeeded" validate:"omitempty"`),
	}
}

// Mixin of the ShipmentCommodity.
func (ShipmentCommodity) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the ShipmentCommodity.
func (ShipmentCommodity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("shipment", Shipment.Type).
			Immutable().
			Required().
			Field("shipment_id").
			Ref("shipment_commodities").
			Unique(),
	}
}
