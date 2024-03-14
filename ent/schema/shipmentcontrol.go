package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ShipmentControl holds the schema definition for the ShipmentControl entity.
type ShipmentControl struct {
	ent.Schema
}

// Fields of the ShipmentControl.
func (ShipmentControl) Fields() []ent.Field {
	return []ent.Field{
		field.Bool("auto_rate_shipment").
			Default(true).
			StructTag(`json:"autoRateShipment"`),
		field.Bool("calculate_distance").
			Default(true).
			StructTag(`json:"calculateDistance"`),
		field.Bool("enforce_rev_code").
			Default(false).
			StructTag(`json:"enforceRevCode"`),
		field.Bool("enforce_voided_comm").
			Default(false).
			StructTag(`json:"enforceVoidedComm"`),
		field.Bool("generate_routes").
			Default(false).
			StructTag(`json:"generateRoutes"`),
		field.Bool("enforce_commodity").
			Default(false).
			StructTag(`json:"enforceCommodity"`),
		field.Bool("auto_sequence_stops").
			Default(true).
			StructTag(`json:"autoSequenceStops"`),
		field.Bool("auto_shipment_total").
			Default(true).
			StructTag(`json:"autoShipmentTotal"`),
		field.Bool("enforce_origin_destination").
			Default(false).
			StructTag(`json:"enforceOriginDestination"`),
		field.Bool("check_for_duplicate_bol").
			Default(false).
			StructTag(`json:"checkForDuplicateBol"`),
		field.Bool("send_placard_info").
			Default(false).
			StructTag(`json:"sendPlacardInfo"`),
		field.Bool("enforce_hazmat_seg_rules").
			Default(true).
			StructTag(`json:"enforceHazmatSegRules"`),
	}
}

// Mixin of the ShipmentControl.
func (ShipmentControl) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

// Edges of the ShipmentControl.
func (ShipmentControl) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("shipment_control").
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
