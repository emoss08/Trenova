package schema

import (
	"regexp"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/jackc/pgx/v5/pgtype"
)

// GeneralLedgerAccount holds the schema definition for the GeneralLedgerAccount entity.
type GeneralLedgerAccount struct {
	ent.Schema
}

// Fields of the GeneralLedgerAccount.
func (GeneralLedgerAccount) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(1)",
				dialect.SQLite:   "VARCHAR(1)",
			}).
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("account_number").
			MaxLen(7).
			Match(regexp.MustCompile("^[0-9]{4}-[0-9]{2}$")).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(7)",
				dialect.SQLite:   "VARCHAR(7)",
			}).
			StructTag(`json:"accountNumber" validate:"required,max=7"`),
		field.Enum("account_type").
			Values("Asset",
				"Liability",
				"Equity",
				"Revenue",
				"Expense").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(9)",
				dialect.SQLite:   "VARCHAR(9)",
			}).
			StructTag(`json:"accountType" validate:"required"`),
		field.String("cash_flow_type").
			Optional().
			StructTag(`json:"cashFlowType" validate:"omitempty"`),
		field.String("account_sub_type").
			Optional().
			StructTag(`json:"accountSubType" validate:"omitempty"`),
		field.String("account_class").
			Optional().
			StructTag(`json:"accountClass" validate:"omitempty"`),
		field.Float("balance").
			Optional().
			StructTag(`json:"balance" validate:"omitempty"`),
		field.Float("interest_rate").
			Optional().
			StructTag(`json:"interestRate" validate:"omitempty"`),
		field.Other("date_closed", &pgtype.Date{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"dateClosed" validate:"omitempty"`),
		field.String("notes").
			Optional(),
		field.Bool("is_tax_relevant").
			Default(false).
			StructTag(`json:"isTaxRelevant" validate:"omitempty"`),
		field.Bool("is_reconciled").
			Default(false).
			StructTag(`json:"isReconciled" validate:"omitempty"`),
	}
}

// Edges of the GeneralLedgerAccount.
func (GeneralLedgerAccount) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("tags", Tag.Type),
	}
}

// Mixin of the GeneralLedgerAccount.
func (GeneralLedgerAccount) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
