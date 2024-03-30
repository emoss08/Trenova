package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
)

// DocumentClassification holds the schema definition for the DocumentClassification entity.
type DocumentClassification struct {
	ent.Schema
}

// Fields of the DocumentClassification.
func (DocumentClassification) Fields() []ent.Field {
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
			MaxLen(10).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			StructTag(`json:"name" validate:"required,max=10"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
	}
}

// Mixin of the DocumentClassification.
func (DocumentClassification) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the DocumentClassification.
func (DocumentClassification) Edges() []ent.Edge {
	return nil
}
