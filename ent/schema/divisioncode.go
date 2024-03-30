package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DivisionCode holds the schema definition for the DivisionCode entity.
type DivisionCode struct {
	ent.Schema
}

// Fields of the DivisionCode.
func (DivisionCode) Fields() []ent.Field {
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
			MaxLen(4).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(4)",
				dialect.SQLite:   "VARCHAR(4)",
			}).
			StructTag(`json:"code" validate:"required,max=4"`),
		field.Text("description").
			NotEmpty().
			StructTag(`json:"description" validate:"required,max=100"`),
		field.UUID("cash_account_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"cashAccountId" validate:"omitempty"`),
		field.UUID("ap_account_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"apAccountId" validate:"omitempty"`),
		field.UUID("expense_account_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"expenseAccountId" validate:"omitempty"`),
	}
}

// Mixin of the DivisionCode.
func (DivisionCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Indexes of the DivisionCode.
func (DivisionCode) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure the code is unique for the organization.
		index.Fields("code", "organization_id").
			Unique(),
	}
}

// Edges of the DivisionCode.
func (DivisionCode) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("cash_account", GeneralLedgerAccount.Type).
			Field("cash_account_id").
			StructTag(`json:"cashAccount"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("ap_account", GeneralLedgerAccount.Type).
			Field("ap_account_id").
			StructTag(`json:"apAccount"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("expense_account", GeneralLedgerAccount.Type).
			Field("expense_account_id").
			StructTag(`json:"expenseAccount"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
	}
}

// Hooks for the DivisionCode.
func (DivisionCode) Hooks() []ent.Hook {
	// TODO(Wolfred): Implement validation to check the general ledger account classifications.
	return nil
}
