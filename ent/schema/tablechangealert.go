package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
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
			StructTag(`json:"status"`),
		field.String("name").
			MaxLen(50).
			StructTag(`json:"name" validate:"required,max=50"`),
		field.Enum("database_action").
			Values("Insert", "Update", "Delete", "All").
			StructTag(`json:"databaseAction" validate:"required,oneof=Insert Update Delete All"`),
		field.Enum("source").
			Values("Kafka", "Database").
			StructTag(`json:"source" validate:"required,oneof=Kafka Database"`),
		field.String("table_name").
			Optional().
			MaxLen(255).
			StructTag(`json:"tableName" validate:"max=255"`),
		field.String("topic").
			Optional().
			MaxLen(255).
			StructTag(`json:"topic"`),
		field.Text("description").
			Optional().
			StructTag(`json:"description"`),
		field.String("custom_subject").
			Optional().
			MaxLen(255).
			StructTag(`json:"customSubject"`),
		field.String("function_name").
			Optional().
			MaxLen(50).
			StructTag(`json:"functionName"`),
		field.String("trigger_name").
			Optional().
			MaxLen(50).
			StructTag(`json:"triggerName"`),
		field.String("listener_name").
			Optional().
			MaxLen(50).
			StructTag(`json:"listenerName"`),
		// TODO(Wolfred): turn `email_receipients` into a relationship with the User entity
		field.Text("email_recipients").
			Optional().
			StructTag(`json:"emailRecipients"`),
		field.Time("effective_date").
			Optional().
			StructTag(`json:"effectiveDate"`),
		field.Time("expiration_date").
			Optional().
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
