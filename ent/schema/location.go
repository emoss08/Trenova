package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("code").
			NotEmpty().
			MaxLen(10).
			StructTag(`json:"code" validate:"required,max=10"`),
		field.UUID("location_category_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"locationCategoryId" validate:"omitempty"`),
		field.String("name").
			NotEmpty().
			StructTag(`json:"name" validate:"required"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
		field.String("address_line_1").
			NotEmpty().
			MaxLen(150).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(150)",
				dialect.SQLite:   "VARCHAR(150)",
			}).
			StructTag(`json:"addressLine1" validate:"required,max=150"`),
		field.String("address_line_2").
			Optional().
			MaxLen(150).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(150)",
				dialect.SQLite:   "VARCHAR(150)",
			}).
			StructTag(`json:"addressLine2" validate:"omitempty,max=150"`),
		field.String("city").
			NotEmpty().
			MaxLen(150).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(150)",
				dialect.SQLite:   "VARCHAR(150)",
			}).
			StructTag(`json:"city" validate:"required,max=150"`),
		field.UUID("state_id", uuid.UUID{}).
			StructTag(`json:"stateId" validate:"omitempty,uuid"`),
		field.String("postal_code").
			NotEmpty().
			MaxLen(10).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			StructTag(`json:"postalCode" validate:"required,max=10"`),
		field.Float("longitude").
			Optional().
			StructTag(`json:"longitude" validate:"omitempty"`),
		field.Float("latitude").
			Optional().
			StructTag(`json:"latitude" validate:"omitempty"`),
		field.String("place_id").
			Optional().
			MaxLen(255).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(255)",
				dialect.SQLite:   "VARCHAR(255)",
			}).
			Comment("Place ID from Google Maps API.").
			StructTag(`json:"placeId" validate:"omitempty,max=255"`),
		field.Bool("is_geocoded").
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

// Indexes of the Location.
func (Location) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure the code is unique for the organization.
		index.Fields("code", "organization_id").
			Unique(),
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
	}
}
