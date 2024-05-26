package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Rate holds the schema definition for the Rate entity.
type Rate struct {
	ent.Schema
}

// Fields of the Rate.
func (Rate) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(1)",
				dialect.SQLite:   "VARCHAR(1)",
			}).
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("rate_number").
			NotEmpty().
			MaxLen(10).
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			StructTag(`json:"rate_number" validate:"omitempty"`),
		field.UUID("customer_id", uuid.UUID{}).
			Unique().
			StructTag(`json:"customerId" validate:"required"`),
		field.Other("effective_date", &pgtype.Date{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"effectiveDate"`),
		field.Other("expiration_date", &pgtype.Date{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"expirationDate"`),
		field.UUID("commodity_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"commodityId" validate:"omitempty"`),
		field.UUID("shipment_type_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"shipmentTypeId" validate:"omitempty"`),
		field.UUID("origin_location_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"originLocationId" validate:"omitempty"`),
		field.UUID("destination_location_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"destinationLocationId" validate:"omitempty"`),
		field.Enum("rating_method").
			Values("FlatRate", "PerMile", "PerHundredWeight", "PerStop", "PerPound", "Other").
			Default("FlatRate").
			StructTag(`json:"ratingMethod" validate:"omitempty"`),
		field.Float("rate_amount").
			Positive().
			SchemaType(map[string]string{
				dialect.MySQL:    "decimal(19,4)",
				dialect.Postgres: "numeric(19,4)",
			}).
			StructTag(`json:"rateAmount" validate:"required"`),
		field.Text("comment").
			Optional().
			StructTag(`json:"comment" validate:"omitempty"`),
	}
}

// Mixin of the Rate.
func (Rate) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the Rate.
func (Rate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", Customer.Type).
			Ref("rates").
			Field("customer_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("commodity", Commodity.Type).
			Ref("rates").
			Field("commodity_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("shipment_type", ShipmentType.Type).
			Ref("rates").
			Field("shipment_type_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("origin_location", Location.Type).
			Ref("rates_origin").
			Field("origin_location_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("destination_location", Location.Type).
			Ref("rates_destination").
			Field("destination_location_id").
			Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
