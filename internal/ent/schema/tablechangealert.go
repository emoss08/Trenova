package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"github.com/emoss08/trenova/internal/ent/hook"
	"github.com/emoss08/trenova/internal/util/mutators"
	"github.com/emoss08/trenova/internal/util/validators"
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
		field.Enum("source").
			Values("Kafka", "Database").
			StructTag(`json:"source" validate:"required,oneof=Kafka Database"`),
		field.String("table_name").
			Optional().
			StructTag(`json:"tableName" validate:"max=255,required_if=source Database"`),
		field.String("topic_name").
			Optional().
			StructTag(`json:"topicName" validate:"max=255,required_if=source Kafka"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description"`),
		field.String("custom_subject").
			Optional().
			StructTag(`json:"customSubject"`),
		field.String("function_name").
			Optional().
			MaxLen(50).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(50)",
				dialect.SQLite:   "VARCHAR(50)",
			}).
			StructTag(`json:"functionName"`),
		field.String("trigger_name").
			Optional().
			MaxLen(50).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(50)",
				dialect.SQLite:   "VARCHAR(50)",
			}).
			StructTag(`json:"triggerName"`),
		field.String("listener_name").
			Optional().
			MaxLen(50).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(50)",
				dialect.SQLite:   "VARCHAR(50)",
			}).
			StructTag(`json:"listenerName"`),
		field.Text("email_recipients").
			Optional().
			StructTag(`json:"emailRecipients" validate:"omitempty,commaSeparatedEmails"`),
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
		hook.On(mutators.MutateTableChangeAlerts, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne),
	}
}
