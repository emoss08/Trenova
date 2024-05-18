package schema

import (
	"context"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	gen "github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/hook"
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

// Hooks for the ReasonCode.
func (ReasonCode) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(uppercaseCode, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
	}
}

func uppercaseCode(next ent.Mutator) ent.Mutator {
	return hook.ReasonCodeFunc(func(ctx context.Context, m *gen.ReasonCodeMutation) (ent.Value, error) {
		code, exists := m.Code()
		if exists {
			// Ensure the code is always uppercase.
			m.SetCode(strings.ToUpper(code))
		}

		return next.Mutate(ctx, m)
	})
}
