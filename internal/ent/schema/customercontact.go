package schema

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	gen "github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/hook"
	"github.com/emoss08/trenova/internal/util"

	"github.com/google/uuid"
)

// CustomerContact holds the schema definition for the CustomerContact entity.
type CustomerContact struct {
	ent.Schema
}

// Fields of the CustomerContact.
func (CustomerContact) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("customer_id", uuid.UUID{}).
			Immutable().
			Unique().
			StructTag(`json:"customerId" validate:"omitempty"`),
		field.String("name").
			NotEmpty().
			MaxLen(150).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(150)",
				dialect.SQLite:   "VARCHAR(150)",
			}).
			StructTag(`json:"name" validate:"required,max=10"`),
		field.String("email").
			Optional().
			MaxLen(150).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(150)",
				dialect.SQLite:   "VARCHAR(150)",
			}).
			StructTag(`json:"email" validate:"required,email"`),
		field.String("title").
			Optional().
			MaxLen(100).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(100)",
				dialect.SQLite:   "VARCHAR(100)",
			}).
			StructTag(`json:"title" validate:"omitempty,max=100"`),
		field.String("phone_number").
			MaxLen(15).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(15)",
				dialect.SQLite:   "VARCHAR(15)",
			}).
			StructTag(`json:"phoneNumber" validate:"omitempty,phoneNum"`),
		field.Bool("is_payable_contact").
			Default(false).
			StructTag(`json:"isPayableContact" validate:"omitempty"`),
	}
}

// Edges of the CustomerContact.
func (CustomerContact) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", Customer.Type).
			Field("customer_id").
			Ref("contacts").
			Required().
			Immutable().
			Unique(),
	}
}

// Mixin of the CustomerContact.
func (CustomerContact) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Hooks of the CustomerContact.
func (CustomerContact) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.CustomerContactFunc(func(ctx context.Context, m *gen.CustomerContactMutation) (ent.Value, error) {
					payableContact, exists := m.IsPayableContact()
					emailValue, emailExists := m.Email()

					// Check if 'payableContact' is true and email does not exist
					if exists && payableContact && (!emailExists || emailValue == "") {
						return nil, util.NewValidationError("Payable contact must have an email address. Please try again.", "invalid", "email")
					}

					return next.Mutate(ctx, m)
				})
			}, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
	}
}
