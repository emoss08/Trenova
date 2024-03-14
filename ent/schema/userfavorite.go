package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// UserFavorite holds the schema definition for the UserFavorite entity.
type UserFavorite struct {
	ent.Schema
}

// Fields of the UserFavorite.
func (UserFavorite) Fields() []ent.Field {
	return []ent.Field{
		field.String("page_link").
			MaxLen(255).
			Unique().
			StructTag(`json:"pageLink"`),
		field.UUID("user_id", uuid.UUID{}).
			Immutable().
			StructTag(`json:"userId"`),
	}
}

// Mixin for the UserFavorite.
func (UserFavorite) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the UserFavorite.
func (UserFavorite) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("user_favorites").
			Field("user_id").
			Unique().
			Required().
			Immutable(),
	}
}
