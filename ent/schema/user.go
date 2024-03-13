package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A"),
		field.String("name").
			MaxLen(255),
		field.String("username").
			MaxLen(30),
		field.String("password").
			MaxLen(100).
			Sensitive(),
		field.String("email").
			MaxLen(255),
		field.String("date_joined").
			Optional(),
		field.Enum("timezone").
			Values(
				"TimezoneAmericaLosAngeles",
				"TimezoneAmericaDenver",
				"TimezoneAmericaChicago",
				"TimezoneAmericaNewYork").
			Default("TimezoneAmericaLosAngeles"),
		field.String("profile_pic_url").
			Optional(),
		field.String("thumbnail_url").
			Optional(),
		field.String("phone_number").
			Optional(),
		field.Bool("is_admin").
			Default(false),
		field.Bool("is_super_admin").
			Default(false),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return nil
}

// Mixin of the BusinessUnit.
func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
