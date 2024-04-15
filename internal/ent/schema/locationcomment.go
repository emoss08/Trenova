package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// LocationComment holds the schema definition for the LocationComment entity.
type LocationComment struct {
	ent.Schema
}

// Fields of the LocationComment.
func (LocationComment) Fields() []ent.Field {
	return []ent.Field{
		// ID will be set in the service layer.
		field.UUID("location_id", uuid.UUID{}).
			StructTag(`json:"locationId" validate:"omitempty"`),
		field.UUID("user_id", uuid.UUID{}).
			StructTag(`json:"userId" validate:"required"`),
		field.UUID("comment_type_id", uuid.UUID{}).
			StructTag(`json:"commentTypeId" validate:"omitempty"`),
		field.Text("comment").
			NotEmpty().
			StructTag(`json:"comment" validate:"required"`),
	}
}

// Mixin of the LocationComment.
func (LocationComment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the LocationComment.
func (LocationComment) Edges() []ent.Edge {
	return []ent.Edge{
		// One to many edge of the LocationComment to Location.
		edge.From("location", Location.Type).
			Field("location_id").
			Ref("comments").
			Required().
			Unique(),
		edge.To("user", User.Type).
			Field("user_id").
			Required().
			Unique(),
		edge.To("comment_type", CommentType.Type).
			Field("comment_type_id").
			Required().
			StructTag(`json:"commentType" validate:"omitempty"`).
			Unique(),
	}
}
