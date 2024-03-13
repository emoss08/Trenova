package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// DispatchControl holds the schema definition for the DispatchControl entity.
type DispatchControl struct {
	ent.Schema
}

// Fields of the DispatchControl.
func (DispatchControl) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("organization_id", uuid.UUID{}).
			StructTag(`json:"organizationId"`),
		field.UUID("business_unit_id", uuid.UUID{}).
			StructTag(`json:"businessUnitId"`),
		field.Enum("record_service_incident").
			Values("Never", "Pickup", "Delivery", "PickupAndDelivery", "AllExceptShipper").
			Default("Never").
			StructTag(`json:"recordServiceIncident"`),
		field.Float("deadhead_target").
			Default(0).
			StructTag(`json:"deadheadTarget"`),
		field.Int("max_shipment_weight_limit").
			Default(80000).
			Positive().
			StructTag(`json:"maxShipmentWeightLimit"`),
		field.Uint8("grace_period").
			Default(0).
			StructTag(`json:"gracePeriod"`),
		field.Bool("enforce_worker_assign").
			Default(true).
			StructTag(`json:"enforceWorkerAssign"`),
		field.Bool("trailer_continuity").
			Default(false).
			StructTag(`json:"trailerContinuity"`),
		field.Bool("dupe_trailer_check").
			Default(false).
			StructTag(`json:"dupeTrailerCheck"`),
		field.Bool("maintenance_compliance").
			Default(true).
			StructTag(`json:"maintenanceCompliance"`),
		field.Bool("regulatory_check").
			Default(false).
			StructTag(`json:"regulatoryCheck"`),
		field.Bool("prev_shipment_on_hold").
			Default(false).
			StructTag(`json:"prevShipmentOnHold"`),
		field.Bool("worker_time_away_restriction").
			Default(true).
			StructTag(`json:"workerTimeAwayRestriction"`),
		field.Bool("tractor_worker_fleet_constraint").
			Default(false).
			StructTag(`json:"tractorWorkerFleetConstraint"`),
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
		edge.To("organization", Organization.Type).
			Field("organization_id").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
		edge.To("business_unit", BusinessUnit.Type).
			Field("business_unit_id").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
	}
}
