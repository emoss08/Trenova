package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Permission holds the schema definition for the Permission entity.
type Permission struct {
	ent.Schema
}

// Fields of the Permission.
func (Permission) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty(),
		field.String("description").
			Optional(),
		field.String("action").
			Optional(),
		field.String("name_humanized").
			Comment("Name of the permission in human readable format.").
			Optional(),
		field.UUID("resource_id", uuid.UUID{}).
			Unique().
			Immutable(),
	}
}

// Edges of the Permission.
func (Permission) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("resource", Resource.Type).
			Ref("permissions").
			Field("resource_id").
			Immutable().
			Required().
			Unique(),
		edge.From("roles", Role.Type).
			Ref("permissions"),
	}
}

// Mixin of the Permission.
func (Permission) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
