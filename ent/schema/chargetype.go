package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
)

// ChargeType holds the schema definition for the ChargeType entity.
type ChargeType struct {
	ent.Schema
}

// Fields of the ChargeType.
func (ChargeType) Fields() []ent.Field {
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
			MaxLen(50).
			NotEmpty().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(50)",
				dialect.SQLite:   "VARCHAR(50)",
			}).
			StructTag(`json:"name" validate:"required,max=50"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
	}
}

// Edges of the ChargeType.
func (ChargeType) Edges() []ent.Edge {
	return nil
}

// Mixin for the ChargeType.
func (ChargeType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
