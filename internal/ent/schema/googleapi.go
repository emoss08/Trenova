package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// GoogleApi holds the schema definition for the GoogleApi entity.
type GoogleApi struct { //nolint:stylecheck,revive // This is the name used in the ent generation command.
	ent.Schema
}

// Fields of the GoogleApi.
func (GoogleApi) Fields() []ent.Field {
	return []ent.Field{
		field.String("api_key").
			NotEmpty().
			StructTag(`json:"apiKey" validate:"required"`),
		field.Enum("mileage_unit").
			Values("Imperial", "Metric").
			Default("Imperial").
			StructTag(`json:"mileageUnit" validate:"required,oneof=Imperial Metric"`),
		field.Bool("add_customer_location").
			Default(false).
			StructTag(`json:"addCustomerLocation" validate:"omitempty"`),
		field.Bool("auto_geocode").
			Default(false).
			StructTag(`json:"autoGeocode" validate:"omitempty"`),
		field.Bool("add_location").
			Default(false).
			StructTag(`json:"addLocation" validate:"omitempty"`),
		field.Enum("traffic_model").
			Values("BestGuess", "Optimistic", "Pessimistic").
			Default("BestGuess").
			StructTag(`json:"trafficModel" validate:"required,oneof=BestGuess Optimistic Pessimistic"`),
	}
}

// Mixin for the GoogleApi.
func (GoogleApi) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

// Edges of the GoogleApi.
func (GoogleApi) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("google_api").
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
