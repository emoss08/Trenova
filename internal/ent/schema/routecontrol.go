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
			Values("T", "G").
			Default("T").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(1)",
				dialect.SQLite:   "VARCHAR(1)",
			}).
			StructTag(`json:"distanceMethod"`),
		field.Enum("mileage_unit").
			Values("M", "I").
			Default("M").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(1)",
				dialect.SQLite:   "VARCHAR(1)",
			}).
			StructTag(`json:"mileageUnit"`),
		field.Bool("generate_routes").
			Default(false).
			StructTag(`json:"generateRoutes"`),
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
