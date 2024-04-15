package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
)

// QualifierCode holds the schema definition for the QualifierCode entity.
type QualifierCode struct {
	ent.Schema
}

// Fields of the QualifierCode.
func (QualifierCode) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(1)",
				dialect.SQLite:   "VARCHAR(1)",
			}).
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("code").
			NotEmpty().
			MaxLen(10).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			StructTag(`json:"code" validate:"required,max=10"`),
		field.Text("description").
			NotEmpty().
			StructTag(`json:"description" validate:"omitempty"`),
	}
}

// Mixin of the QualifierCode.
func (QualifierCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the QualifierCode.
func (QualifierCode) Edges() []ent.Edge {
	return nil
}
