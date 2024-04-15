package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ShipmentMove holds the schema definition for the ShipmentMove entity.
type ShipmentMove struct {
	ent.Schema
}

// Fields of the ShipmentMove.
func (ShipmentMove) Fields() []ent.Field {
	return []ent.Field{
		field.String("reference_number").
			NotEmpty().
			Unique().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			Immutable().
			StructTag(`json:"reference_number"`),
		field.Enum("status").
			Values("New",
				"InProgress",
				"Completed",
				"Voided").
			Default("New").
			StructTag(`json:"status" validate:"required"`),
		field.Bool("is_loaded").
			Default(false).
			StructTag(`json:"isLoaded" validate:"required"`),
		field.UUID("shipment_id", uuid.UUID{}).
			Unique().
			Immutable().
			StructTag(`json:"shipmentId"`),
		field.UUID("tractor_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"tractorId" validate:"omitempty"`),
		field.UUID("trailer_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"trailerId" validate:"omitempty"`),
		field.UUID("primary_worker_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"primaryWorkerId" validate:"omitempty"`),
		field.UUID("secondary_worker_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"secondaryWorkerId" validate:"omitempty"`),
	}
}

// Mixin of the ShipmentMove.
func (ShipmentMove) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the ShipmentMove.
func (ShipmentMove) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("shipment", Shipment.Type).
			Ref("shipment_moves").
			Unique().
			Field("shipment_id").
			Immutable().
			Required().
			StructTag(`json:"shipment,omitempty"`),
		edge.To("tractor", Tractor.Type).
			Field("tractor_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"tractor,omitempty"`),
		edge.To("trailer", Tractor.Type).
			Field("trailer_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"trailer,omitempty"`),
		edge.To("primary_worker", Worker.Type).
			Field("primary_worker_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"primaryWorker,omitempty"`),
		edge.To("secondary_worker", Worker.Type).
			Field("secondary_worker_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"secondaryWorker,omitempty"`),
		edge.To("move_stops", Stop.Type).
			StructTag(`json:"moveStops,omitempty"`),
	}
}
