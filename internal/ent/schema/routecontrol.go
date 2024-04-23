package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// RouteControl holds the schema definition for the RouteControl entity.
type RouteControl struct {
	ent.Schema
}

// Fields of the RouteControl.
func (RouteControl) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("distance_method").
			Values("Trenova", "Google").
			Default("Trenova").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(8)",
				dialect.SQLite:   "VARCHAR(8)",
			}).
			StructTag(`json:"distanceMethod" validate:"required,oneof=Trenova Google"`),
		field.Enum("mileage_unit").
			Values("UnitsMetric", "UnitsImperial").
			Default("UnitsMetric").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(14)",
				dialect.SQLite:   "VARCHAR(14)",
			}).
			StructTag(`json:"mileageUnit" validate:"required,oneof=UnitsMetric UnitsImperial"`),
		field.Bool("generate_routes").
			Default(false).
			StructTag(`json:"generateRoutes" validate:"omitempty"`),
	}
}

// Mixin of the RouteControl.
func (RouteControl) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

// Edges of the RouteControl.
func (RouteControl) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("route_control").
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
