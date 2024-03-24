package schema

import (
	"regexp"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("account_number").
			MaxLen(7).
			Match(regexp.MustCompile("^[0-9]{4}-[0-9]{2}$")).
			StructTag(`json:"accountNumber" validate:"required,max=7"`),
		field.Enum("account_type").
			Values("Asset",
				"Liability",
				"Equity",
				"Revenue",
				"Expense").
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
			Nillable().
			StructTag(`json:"balance" validate:"omitempty"`),
		field.Float("interest_rate").
			Optional().
			Nillable().
			StructTag(`json:"interestRate" validate:"omitempty"`),
		field.Time("date_opened").
			Immutable().
			Default(time.Now).
			StructTag(`json:"dateOpened" validate:"omitempty"`),
		field.Time("date_closed").
			Optional().
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

// Indexes of the GeneralLedgerAccount.
func (GeneralLedgerAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_number", "organization_id").
			Unique(),
	}
}
