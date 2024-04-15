package schema

import (
	"context"
	"log"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	gen "github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/generalledgeraccount"
	"github.com/emoss08/trenova/internal/ent/hook"
	"github.com/emoss08/trenova/internal/util"
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
		hook.On(ensureExpenseAccountType, ent.OpCreate|ent.OpUpdateOne|ent.OpUpdate),
		hook.On(ensureRevenueAccountType, ent.OpCreate|ent.OpUpdateOne|ent.OpUpdate),
	}
}

// ensureExpenseAccountType checks that the expense account has an "Expense" type.
func ensureExpenseAccountType(next ent.Mutator) ent.Mutator {
	return hook.RevenueCodeFunc(func(ctx context.Context, m *gen.RevenueCodeMutation) (ent.Value, error) {
		if !m.Op().Is(ent.OpCreate) && !m.Op().Is(ent.OpUpdate) && !m.Op().Is(ent.OpUpdateOne) {
			log.Println("Operation is not create or update, no check needed")
			return next.Mutate(ctx, m)
		}

		expenseAccountID, exists := m.ExpenseAccountID()
		if exists {
			expenseAccount, err := m.Client().GeneralLedgerAccount.Get(ctx, expenseAccountID)
			if err != nil {
				return nil, err
			}
			if expenseAccount.AccountType != generalledgeraccount.AccountTypeExpense {
				return nil, util.NewValidationError("The expense account must be an expense account",
					"invalidExpenseAccount", "expenseAccountId")
			}
		}
		return next.Mutate(ctx, m)
	})
}

// ensureRevenueAccountType checks that the revenue account has a "Revenue" type.
func ensureRevenueAccountType(next ent.Mutator) ent.Mutator {
	return hook.RevenueCodeFunc(func(ctx context.Context, m *gen.RevenueCodeMutation) (ent.Value, error) {
		if !m.Op().Is(ent.OpCreate) && !m.Op().Is(ent.OpUpdate) && !m.Op().Is(ent.OpUpdateOne) {
			return next.Mutate(ctx, m)
		}

		revenueAccountID, exists := m.RevenueAccountID()
		if exists {
			revenueAccount, err := m.Client().GeneralLedgerAccount.Get(ctx, revenueAccountID)
			if err != nil {
				return nil, err
			}
			if revenueAccount.AccountType != generalledgeraccount.AccountTypeRevenue {
				return nil, util.NewValidationError("The revenue account must be a revenue account",
					"invalidRevenueAccount", "revenueAccountId")
			}
		}
		return next.Mutate(ctx, m)
	})
}
