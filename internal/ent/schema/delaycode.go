package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
)

// DelayCode holds the schema definition for the DelayCode entity.
type DelayCode struct {
	ent.Schema
}

// Fields of the DelayCode.
func (DelayCode) Fields() []ent.Field {
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
			MaxLen(20).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(20)",
				dialect.SQLite:   "VARCHAR(20)",
			}).
			StructTag(`json:"code" validate:"required,max=4"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description" validate:"omitempty"`),
		field.Bool("f_carrier_or_driver").
			Optional().
			StructTag(`json:"fCarrierOrDriver" validate:"omitempty"`),
		field.String("color").
			Optional().
			StructTag(`json:"color" validate:"omitempty"`),
	}
}

// Edges of the DelayCode.
func (DelayCode) Edges() []ent.Edge {
	return nil
}

// Mixin of the DelayCode.
func (DelayCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
