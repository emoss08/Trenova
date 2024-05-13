package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// CustomerDetentionPolicy holds the schema definition for the CustomerDetentionPolicy entity.
type CustomerDetentionPolicy struct {
	ent.Schema
}

// Fields of the CustomerDetentionPolicy.
func (CustomerDetentionPolicy) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(1)",
				dialect.SQLite:   "VARCHAR(1)",
			}).
			Default("A"),
		field.UUID("customer_id", uuid.UUID{}).
			Immutable().
			Unique(),
		field.UUID("commodity_id", uuid.UUID{}).
			Unique().
			Optional().
			Nillable().
			Comment("The type of commodity to which the detention policy applies. This helps in customizing policies for different commodities.").
			StructTag(`json:"commodityId" validate:"omitempty"`),
		field.UUID("revenue_code_id", uuid.UUID{}).
			Unique().
			Optional().
			Nillable().
			Comment("A unique code associated with the revenue generated from detention charges.").
			StructTag(`json:"revenueCodeId" validate:"omitempty"`),
		field.UUID("shipment_type_id", uuid.UUID{}).
			Unique().
			Comment("Type of shipment (e.g., Standard, Expedited) to which the detention policy is applicable.").
			Optional().
			Nillable().
			StructTag(`json:"shipmentTypeId" validate:"omitempty"`),
		field.Enum("application_scope").
			Values("PICKUP", "DELIVERY", "BOTH").
			Default("PICKUP").
			Comment("Specifies whether the policy applies to pickups, deliveries, or both.").
			StructTag(`json:"applicationScope" validate:"required,oneof=PICKUP DELIVERY BOTH"`),
		field.Int("charge_free_time").
			Optional().
			Positive().
			Comment("The threshold time (in minutes) for the start of detention charges. This represents the allowed free time before charges apply.").
			StructTag(`json:"chargeFreeTime" validate:"omitempty,gt=0"`),
		field.Int("payment_free_time").
			Optional().
			Positive().
			Comment("The time (in minutes) considered for calculating detention payments. This can differ from charge_free_time in certain scenarios.").
			StructTag(`json:"paymentFreeTime" validate:"omitempty,gt=0"`),
		field.Bool("late_arrival_policy").
			Optional().
			Default(false).
			Comment("Indicates whether the policy applies to late arrivals. True if detention charges apply to late arrivals.").
			StructTag(`json:"lateArrivalPolicy" validate:"omitempty"`),
		field.Int("grace_period").
			Optional().
			Positive().
			Comment("An additional time buffer (in minutes) provided before detention charges kick in, often used to accommodate slight delays.").
			StructTag(`json:"gracePeriod" validate:"omitempty,gt=0"`),
		field.UUID("accessorial_charge_id", uuid.UUID{}).
			Unique().
			Optional().
			Nillable().
			Comment("The unique identifier for the accessorial charge associated with the detention policy.").
			StructTag(`json:"accessorialChargeId" validate:"omitempty"`),
		field.Int("units").
			Optional().
			Positive().
			Comment("The number of units (e.g., pallets, containers) considered for detention charges.").
			StructTag(`json:"units" validate:"omitempty,gt=0"`),
		field.Float("amount").
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(19,4)",
				dialect.Postgres: "numeric(19,4)",
			}).
			StructTag(`json:"amount" validate:"required,gt=0"`),
		field.Text("notes").
			Optional().
			Comment("Additional notes or comments about the detention policy.").
			StructTag(`json:"notes" validate:"omitempty"`),
		field.Other("effective_date", &pgtype.Date{}).
			Optional().
			Nillable().
			Comment("The date when the detention policy becomes effective.").
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"effectiveDate"`),
		field.Other("expiration_date", &pgtype.Date{}).
			Optional().
			Nillable().
			Comment("The date when the detention policy expires.").
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"expirationDate"`),
	}
}

// Edges of the CustomerDetentionPolicy.
func (CustomerDetentionPolicy) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", Customer.Type).
			Field("customer_id").
			Ref("detention_policies").
			Required().
			Immutable().
			Unique(),
		edge.To("commodity", Commodity.Type).
			Field("commodity_id").
			Unique().
			StructTag(`json:"commodity"`),
		edge.To("revenue_code", RevenueCode.Type).
			Field("revenue_code_id").
			Unique().
			StructTag(`json:"revenueCode"`),
		edge.To("shipment_type", ShipmentType.Type).
			Field("shipment_type_id").
			Unique().
			StructTag(`json:"shipmentType"`),
		edge.To("accessorial_charge", AccessorialCharge.Type).
			Field("accessorial_charge_id").
			Unique().
			StructTag(`json:"accessorialCharge"`),
	}
}

// Mixin of the CustomerDetentionPolicy.
func (CustomerDetentionPolicy) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
