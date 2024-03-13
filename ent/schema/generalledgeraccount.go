package schema

import (
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
			Default("A"),
		field.String("account_number").
			MaxLen(7),
		field.Enum("account_type").
			Values("AccountTypeAsset",
				"AccountTypeLiability",
				"AccountTypeEquity",
				"AccountTypeRevenue",
				"AccountTypeExpense"),
		field.Enum("cash_flow_type").
			Optional().
			Values("CashFlowOperating", "CashFlowInvesting", "CashFlowFinancing"),
		field.Enum("account_sub_type").
			Optional().
			Values("AccountSubTypeCurrentAsset",
				"AccountSubTypeFixedAsset",
				"AccountSubTypeOtherAsset",
				"AccountSubTypeCurrentLiability",
				"AccountSubTypeLongTermLiability",
				"AccountSubTypeEquity",
				"AccountSubTypeRevenue",
				"AccountSubTypeCostOfGoodsSold",
				"AccountSubTypeExpense",
				"AccountSubTypeOtherIncome",
				"AccountSubTypeOtherExpense"),
		field.Enum("account_class").
			Optional().
			Values("AccountClassificationBank",
				"AccountClassificationCash",
				"AccountClassificationAR",
				"AccountClassificationAP",
				"AccountClassificationINV",
				"AccountClassificationOCA",
				"AccountClassificationFA"),
		field.Float("balance").
			Optional(),
		field.Float("interest_rate").
			Optional(),
		field.Time("date_opened").
			Immutable().
			Default(time.Now),
		field.Time("date_closed").
			Optional(),
		field.String("notes").
			Optional(),
		field.Bool("is_tax_relevant").
			Default(false),
		field.Bool("is_reconciled").
			Default(false),
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
