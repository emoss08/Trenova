package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Resource holds the schema definition for the Resource entity.
type Resource struct {
	ent.Schema
}

// Fields of the Resource.
func (Resource) Fields() []ent.Field {
	return []ent.Field{
		field.String("type").
			NotEmpty().
			Unique(),
		field.String("description").
			Optional(),
	}
}

// Edges of the Resource.
func (Resource) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("permissions", Permission.Type),
	}
}

// Mixin of the Resource.
func (Resource) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}
