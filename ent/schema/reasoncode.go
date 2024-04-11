package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
)

// ReasonCode holds the schema definition for the ReasonCode entity.
type ReasonCode struct {
	ent.Schema
}

// Fields of the ReasonCode.
func (ReasonCode) Fields() []ent.Field {
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
			MaxLen(10).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(10)",
				dialect.SQLite:   "VARCHAR(10)",
			}).
			StructTag(`json:"code" validate:"required,max=10"`),
		field.Enum("code_type").
			Values("Voided", "Cancelled").
			StructTag(`json:"codeType" validate:"required,oneof=Voided Cancelled"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description"`),
	}
}

// Mixin of the ReasonCode.
func (ReasonCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the ReasonCode.
func (ReasonCode) Edges() []ent.Edge {
	return nil
}
