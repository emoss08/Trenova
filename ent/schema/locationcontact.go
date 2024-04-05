package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// LocationContact holds the schema definition for the LocationContact entity.
type LocationContact struct {
	ent.Schema
}

// Fields of the LocationContact.
func (LocationContact) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("location_id", uuid.UUID{}).
			StructTag(`json:"locationId" validate:"required"`),
		field.String("name").
			NotEmpty().
			StructTag(`json:"name" validate:"required"`),
		field.String("email_address").
			Optional().
			StructTag(`json:"emailAddress" validate:"omitempty,email"`),
		field.String("phone_number").
			MaxLen(15).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(15)",
				dialect.SQLite:   "VARCHAR(15)",
			}).
			StructTag(`json:"phoneNumber" validate:"omitempty,phone"`),
	}
}

// Mixin of the LocationContact.
func (LocationContact) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the LocationContact.
func (LocationContact) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("location", Location.Type).
			Field("location_id").
			Ref("contacts").
			Required().
			Unique(),
	}
}
