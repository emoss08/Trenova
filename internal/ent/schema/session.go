package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Session holds the schema definition for the Session entity.
type Session struct {
	ent.Schema
}

// Fields of the Session.
func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique(),
		field.String("data").
			NotEmpty(),
		field.Time("created_at"),
		field.Time("updated_at"),
		field.Time("expires_at"),
	}
}

// Edges of the Session.
func (Session) Edges() []ent.Edge {
	return nil
}
