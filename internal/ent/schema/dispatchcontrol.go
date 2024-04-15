package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// DispatchControl holds the schema definition for the DispatchControl entity.
type DispatchControl struct {
	ent.Schema
}

// Fields of the DispatchControl.
func (DispatchControl) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("record_service_incident").
			Values("Never",
				"Pickup",
				"Delivery",
				"PickupAndDelivery",
				"AllExceptShipper").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(17)",
				dialect.SQLite:   "VARCHAR(17)",
			}).
			Default("Never").
			StructTag(`json:"recordServiceIncident" validate:"required,oneof=Never Pickup Delivery PickupAndDelivery AllExceptShipper"`),
		field.Float("deadhead_target").
			Default(0).
			StructTag(`json:"deadheadTarget" validate:"omitempty"`),
		field.Int32("max_shipment_weight_limit").
			Default(80000).
			Positive().
			StructTag(`json:"maxShipmentWeightLimit" validate:"required,gt=0,lt=1000000"`),
		field.Uint8("grace_period").
			Default(0).
			StructTag(`json:"gracePeriod" validate:"required,lt=100"`),
		field.Bool("enforce_worker_assign").
			Default(true).
			StructTag(`json:"enforceWorkerAssign" validate:"omitempty"`),
		field.Bool("trailer_continuity").
			Default(false).
			StructTag(`json:"trailerContinuity" validate:"omitempty"`),
		field.Bool("dupe_trailer_check").
			Default(false).
			StructTag(`json:"dupeTrailerCheck" validate:"omitempty"`),
		field.Bool("maintenance_compliance").
			Default(true).
			StructTag(`json:"maintenanceCompliance" validate:"omitempty"`),
		field.Bool("regulatory_check").
			Default(false).
			StructTag(`json:"regulatoryCheck" validate:"omitempty"`),
		field.Bool("prev_shipment_on_hold").
			Default(false).
			StructTag(`json:"prevShipmentOnHold" validate:"omitempty"`),
		field.Bool("worker_time_away_restriction").
			Default(true).
			StructTag(`json:"workerTimeAwayRestriction" validate:"omitempty"`),
		field.Bool("tractor_worker_fleet_constraint").
			Default(false).
			StructTag(`json:"tractorWorkerFleetConstraint" validate:"omitempty"`),
	}
}

// Mixin for the DispatchControl.
func (DispatchControl) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

// Edges of the DispatchControl.
func (DispatchControl) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("dispatch_control").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
		edge.To("business_unit", BusinessUnit.Type).
			StorageKey(edge.Column("business_unit_id")).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
	}
}
