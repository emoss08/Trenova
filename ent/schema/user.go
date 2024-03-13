package schema

import "entgo.io/ent"

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return nil
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return nil
}

// Mixin of the BusinessUnit.
func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}
