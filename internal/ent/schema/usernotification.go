package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// UserNotification holds the schema definition for the UserNotification entity.
type UserNotification struct {
	ent.Schema
}

// Fields of the UserNotification.
func (UserNotification) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("user_id", uuid.UUID{}).
			Immutable().
			StructTag(`json:"userId"`),
		field.Bool("is_read").
			Default(false).
			StructTag(`json:"isRead"`),
		field.String("title").
			NotEmpty().
			StructTag(`json:"title"`),
		field.Text("description").
			NotEmpty().
			StructTag(`json:"description"`),
		field.String("action_url").
			Optional().
			Comment("URL to redirect the user to when the notification is clicked.").
			StructTag(`json:"actionUrl"`),
	}
}

// Mixin for the UserNotification.
func (UserNotification) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the UserNotification.
func (UserNotification) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("user_notifications").
			Field("user_id").
			Unique().
			Required().
			Immutable(),
	}
}
