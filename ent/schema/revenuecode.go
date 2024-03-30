package schema

import (
	"context"
	"log"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	gen "github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/generalledgeraccount"
	"github.com/emoss08/trenova/ent/hook"
	"github.com/emoss08/trenova/tools"
	"github.com/google/uuid"
)

// RevenueCode holds the schema definition for the RevenueCode entity.
type RevenueCode struct {
	ent.Schema
}

// Fields of the RevenueCode.
func (RevenueCode) Fields() []ent.Field {
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
		field.UUID("expense_account_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"expenseAccountId" validate:"omitempty"`),
		field.UUID("revenue_account_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"revenueAccountId" validate:"omitempty"`),
	}
}

// Mixin of the RevenueCode.
func (RevenueCode) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Indexes of the RevenueCode.
func (RevenueCode) Indexes() []ent.Index {
	return []ent.Index{
		// Ensure the code is unique for the organization.
		index.Fields("code", "organization_id").
			Unique(),
	}
}

// Edges of the RevenueCode.
func (RevenueCode) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("expense_account", GeneralLedgerAccount.Type).
			Field("expense_account_id").
			StructTag(`json:"expenseAccount"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("revenue_account", GeneralLedgerAccount.Type).
			Field("revenue_account_id").
			StructTag(`json:"revenueAccount"`).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
	}
}

// Hooks for the RevenueCode.
func (RevenueCode) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				// Hook to ensure that the expense account has an account type of "Expense".
				return hook.RevenueCodeFunc(func(ctx context.Context, m *gen.RevenueCodeMutation) (ent.Value, error) {
					// If the expense account is not being set, no need to check.
					if !m.Op().Is(ent.OpCreate) && !m.Op().Is(ent.OpUpdate) && !m.Op().Is(ent.OpUpdateOne) {
						log.Println("Not a create or update operation")
						return next.Mutate(ctx, m)
					}

					// If the expense account is being set, ensure it is an expense account.
					expenseAccountID, expenseAccountIDExists := m.ExpenseAccountID()
					if expenseAccountIDExists {
						// Get the expense account.
						expenseAccount, err := m.Client().GeneralLedgerAccount.Get(ctx, expenseAccountID)
						if err != nil {
							return nil, err
						}

						// Ensure the expense account is an expense account.
						if expenseAccount.AccountType != generalledgeraccount.AccountTypeExpense {
							return nil, tools.NewValidationError("The expense account must be an expense account",
								"invalidExpenseAccount",
								"expenseAccountId")
						}
					}

					return next.Mutate(ctx, m)
				})
			}, ent.OpCreate|ent.OpUpdateOne|ent.OpUpdate),

		// The same hook ,but for the revenue account.
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.RevenueCodeFunc(func(ctx context.Context, m *gen.RevenueCodeMutation) (ent.Value, error) {
					// If the revenue account is not being set, no need to check.
					if !m.Op().Is(ent.OpCreate) && !m.Op().Is(ent.OpUpdate) && !m.Op().Is(ent.OpUpdateOne) {
						return next.Mutate(ctx, m)
					}

					// If the revenue account is being set, ensure it is a revenue account.
					revenueAccountID, revenueAccountIDExists := m.RevenueAccountID()
					if revenueAccountIDExists {
						// Get the revenue account.
						revenueAccount, err := m.Client().GeneralLedgerAccount.Get(ctx, revenueAccountID)
						if err != nil {
							return nil, err
						}

						// Ensure the revenue account is a revenue account.
						if revenueAccount.AccountType != generalledgeraccount.AccountTypeRevenue {
							return nil, tools.NewValidationError("The revenue account must be a revenue account",
								"invalidRevenueAccount",
								"revenueAccountId")
						}
					}

					return next.Mutate(ctx, m)
				})
			}, ent.OpCreate|ent.OpUpdateOne|ent.OpUpdate),
	}
}
