package schema

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	gen "github.com/emoss08/trenova/internal/ent"

	"github.com/emoss08/trenova/internal/ent/hook"

	"github.com/emoss08/trenova/internal/validators"
	"github.com/jackc/pgx/v5/pgtype"
)

// TableChangeAlert holds the schema definition for the TableChangeAlert entity.
type TableChangeAlert struct {
	ent.Schema
}

// Fields of the TableChangeAlert.
func (TableChangeAlert) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A").
			StructTag(`json:"status" validate:"required,oneof=A I"`),
		field.String("name").
			MaxLen(50).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(50)",
				dialect.SQLite:   "VARCHAR(50)",
			}).
			StructTag(`json:"name" validate:"required,max=50"`),
		field.Enum("database_action").
			Values("Insert", "Update", "Delete", "All").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(6)",
				dialect.SQLite:   "VARCHAR(6)",
			}).
			StructTag(`json:"databaseAction" validate:"required,oneof=Insert Update Delete All"`),
		field.String("topic_name").
			Optional().
			StructTag(`json:"topicName" validate:"max=255"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description"`),
		field.String("custom_subject").
			Optional().
			StructTag(`json:"customSubject" validate:"omitempty,max=255"`),
		field.Enum("delivery_method").
			Values("Email", "Local", "Api", "Sms").
			Default("Email").
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(5)",
				dialect.SQLite:   "VARCHAR(5)",
			}).
			StructTag(`json:"deliveryMethod" validate:"required,oneof=Email Local Api Sms"`),
		field.Text("email_recipients").
			Optional().
			StructTag(`json:"emailRecipients" validate:"omitempty,commaSeparatedEmails,required_if=DeliveryMethod Email"`),
		// TODO(Wolfred): Add relationship to the External API entity.
		field.Other("effective_date", &pgtype.Date{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"effectiveDate"`),
		field.Other("expiration_date", &pgtype.Date{}).
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "date",
				dialect.SQLite:   "date",
			}).
			StructTag(`json:"expirationDate"`),
		field.JSON("conditional_logic", map[string]any{}).
			Optional().
			StructTag(`json:"conditionalLogic"`),
	}
}

// Mixin of the TableChangeAlert.
func (TableChangeAlert) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

// Edges of the TableChangeAlert.
func (TableChangeAlert) Edges() []ent.Edge {
	return nil
}

// Hooks for the TableChangeAlert.
func (TableChangeAlert) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(validators.ValidateTableChangeAlerts, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return hook.TableChangeAlertFunc(func(ctx context.Context, m *gen.TableChangeAlertMutation) (ent.Value, error) {
					// conLogic, exists := m.ConditionalLogic()
					// if !exists {
					// 	return next.Mutate(ctx, m)
					// }

					// // if conditional logic is provided, ensure that it is validated.
					// if err := validators.ValidateConditionalLogic(conLogic); err != nil {
					// 	return nil, err
					// }

					validators.ValidateTableChangeAlerts(next)

					return next.Mutate(ctx, m)
				})
			}, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
	}
}
