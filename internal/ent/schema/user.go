package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(1)",
				dialect.SQLite:   "VARCHAR(1)",
			}).
			Default("A"),
		field.String("name").
			NotEmpty(),
		field.String("username").
			NotEmpty().
			MaxLen(30),
		field.String("password").
			MaxLen(100).
			Sensitive(),
		field.String("email").
			NotEmpty(),
		field.String("timezone").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(20)",
				dialect.SQLite:   "VARCHAR(20)",
			}).
			StructTag(`json:"timezone" validate:"required,timezone"`),
		field.String("profile_pic_url").
			Optional().
			StructTag(`json:"profilePicUrl"`),
		field.String("thumbnail_url").
			Optional().
			StructTag(`json:"thumbnailUrl"`),
		field.String("phone_number").
			Optional().
			StructTag(`json:"phoneNumber" validate:"omitempty,phoneNum"`),
		field.Bool("is_admin").
			Default(false).
			StructTag(`json:"isAdmin"`),
		field.Bool("is_super_admin").
			Default(false).
			StructTag(`json:"isSuperAdmin"`),
		field.Time("last_login").
			Optional().
			Nillable().
			StructTag(`json:"lastLogin"`),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user_favorites", UserFavorite.Type),
		edge.To("user_notifications", UserNotification.Type),
		edge.To("shipments", Shipment.Type),
		edge.To("shipment_comments", ShipmentComment.Type),
		edge.To("shipment_charges", ShipmentCharges.Type),
		edge.To("reports", UserReport.Type),
		edge.From("roles", Role.Type).
			Ref("users"),
	}
}

// Indexes of the User.
func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("username", "email").
			Unique(),
	}
}

// Mixin of the User.
func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
