package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Location holds the schema definition for the Location entity.
type Location struct {
	ent.Schema
}

// Fields of the Location.
func (Location) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			Comment("Current status of the location.").
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("code").
			NotEmpty().
			MaxLen(10).
			Comment("Unique code for the location.").
			StructTag(`json:"code" validate:"required,max=10"`),
		field.UUID("location_category_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("Location category ID.").
			StructTag(`json:"locationCategoryId" validate:"omitempty"`),
		field.String("name").
			NotEmpty().
			Comment("Name of the location.").
			StructTag(`json:"name" validate:"required"`),
		field.Text("description").
			Optional().
			Comment("Description of the location.").
			StructTag(`json:"description" validate:"omitempty"`),
		field.String("address_line_1").
			NotEmpty().
			Comment("Adress Line 1 of the location.").
			MaxLen(150).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(150)",
				dialect.SQLite:   "VARCHAR(150)",
			}).
			StructTag(`json:"addressLine1" validate:"required,max=150"`),
		field.String("address_line_2").
			Optional().
			MaxLen(150).
			Comment("Adress Line 2 of the location.").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(150)",
				dialect.SQLite:   "VARCHAR(150)",
			}).
			StructTag(`json:"addressLine2" validate:"omitempty,max=150"`),
		field.String("city").
			NotEmpty().
			MaxLen(150).
			Comment("City of the location.").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(150)",
				dialect.SQLite:   "VARCHAR(150)",
			}).
			StructTag(`json:"city" validate:"required,max=150"`),
		field.UUID("state_id", uuid.UUID{}).
			Comment("State ID.").
			StructTag(`json:"stateId" validate:"omitempty,uuid"`),
		field.String("postal_code").
			NotEmpty().
			MaxLen(10).
			Comment("Postal code of the location.").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			StructTag(`json:"postalCode" validate:"required,max=10"`),
		field.Float("longitude").
			Optional().
			Comment("Longitude of the location.").
			StructTag(`json:"longitude" validate:"omitempty"`),
		field.Float("latitude").
			Optional().
			Comment("Latitude of the location.").
			StructTag(`json:"latitude" validate:"omitempty"`),
		field.String("place_id").
			Optional().
			Comment("Place ID from Google Maps API.").
			MaxLen(255).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(255)",
				dialect.SQLite:   "VARCHAR(255)",
			}).
			Comment("Place ID from Google Maps API.").
			StructTag(`json:"placeId" validate:"omitempty,max=255"`),
		field.Bool("is_geocoded").
			Comment("Is the location geocoded?").
			Default(false).
			StructTag(`json:"isGeocoded"`),
	}
}

// Mixin of the Location.
func (Location) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the Location.
func (Location) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("location_category", LocationCategory.Type).
			Field("location_category_id").
			Unique().
			StructTag(`json:"locationCategory"`),
		edge.To("state", UsState.Type).
			Field("state_id").
			Required().
			StructTag(`json:"state"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("comments", LocationComment.Type).
			StructTag(`json:"comments"`).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("contacts", LocationContact.Type).
			StructTag(`json:"contacts"`).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("origin_route_locations", ShipmentRoute.Type).
			StructTag(`json:"originLocations"`).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("destination_route_locations", ShipmentRoute.Type).
			StructTag(`json:"originLocations"`).
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("rates_origin", Rate.Type),
		edge.To("rates_destination", Rate.Type),
	}
}
