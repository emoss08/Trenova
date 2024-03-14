package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Commodity holds the schema definition for the Commodity entity.
type Commodity struct {
	ent.Schema
}

// Fields of the Commodity.
func (Commodity) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A"),
		field.String("name").
			MaxLen(100).
			NotEmpty(),
		field.Bool("is_hazmat").
			Default(false),
		field.Enum("unit_of_measure").
			Values("P", "T", "D", "C", "A", "B", "O", "L", "I", "S").
			Nillable().
			Optional(),
		field.Float("min_temp").
			Nillable().
			Optional(),
		field.Float("max_temp").
			Nillable().
			Optional(),
		field.Float("set_point_temp").
			Nillable().
			Optional(),
		field.Text("description").
			Nillable().
			Optional(),
	}
}

// Edges of the Commodity.
func (Commodity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("hazardous_material", HazardousMaterial.Type).
			Ref("commodities").
			Unique().
			Required(),
	}
}

func (Commodity) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.QueryField(),
		entgql.Mutations(entgql.MutationCreate()),
	}
}

// Mixin of the Commodity.
func (Commodity) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
