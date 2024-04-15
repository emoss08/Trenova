package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// CommentType holds the schema definition for the CommentType entity.
type CommentType struct {
	ent.Schema
}

// Fields of the CommentType.
func (CommentType) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(1)",
				dialect.SQLite:   "VARCHAR(1)",
			}).
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("name").
			NotEmpty().
			MaxLen(10).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			StructTag(`json:"name" validate:"required,max=10"`),
		field.Enum("severity").
			Values("High", "Medium", "Low").
			Default("Low").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(6)",
				dialect.SQLite:   "VARCHAR(6)",
			}).
			StructTag(`json:"severity" validate:"required,oneof=High Medium Low"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
	}
}

// Edges of the CommentType.
func (CommentType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("shipment_comments", ShipmentComment.Type).
			StructTag(`json:"shipmentComments"`),
	}
}

// Mixin of the CommentType.
func (CommentType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
