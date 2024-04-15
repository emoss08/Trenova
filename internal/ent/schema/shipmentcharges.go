package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ShipmentCharges holds the schema definition for the ShipmentCharges entity.
type ShipmentCharges struct {
	ent.Schema
}

// Fields of the ShipmentCharges.
func (ShipmentCharges) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("shipment_id", uuid.UUID{}).
			Immutable().
			StructTag(`json:"shipmentId" validate:"omitempty"`), // Shipment ID will be set by the system.
		field.UUID("accessorial_charge_id", uuid.UUID{}).
			StructTag(`json:"accessorialChargeId" validate:"required"`),
		field.Text("description").
			StructTag(`json:"description" validate:"omitempty"`),
		field.Float("charge_amount").
			Positive().
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(19,4)",
				dialect.Postgres: "numeric(19,4)",
			}).
			StructTag(`json:"chargeAmount" validate:"required"`),
		field.Int("units").
			Positive().
			StructTag(`json:"units" validate:"required"`),
		field.Float("sub_total").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(19,4)",
				dialect.Postgres: "numeric(19,4)",
			}).
			Positive().
			StructTag(`json:"subTotal" validate:"required"`),
		field.UUID("created_by", uuid.UUID{}).
			StructTag(`json:"createdBy" validate:"omitempty"`),
	}
}

// Mixin of the ShipmentCharges.
func (ShipmentCharges) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the ShipmentCharges.
func (ShipmentCharges) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("shipment", Shipment.Type).
			Ref("shipment_charges").
			Unique().
			Field("shipment_id").
			Immutable().
			Required(),
		edge.From("accessorial_charge", AccessorialCharge.Type).
			Ref("shipment_charges").
			Field("accessorial_charge_id").
			Unique().
			Required(),
		edge.From("user", User.Type).
			Ref("shipment_charges").
			Field("created_by").
			Unique().
			Required().
			StructTag(`json:"createdBy,omitempty"`),
	}
}
