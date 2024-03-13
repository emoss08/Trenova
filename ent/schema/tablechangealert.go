package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Conditioion struct {
	Id        int
	Column    string
	Operation string
	Value     any
	DataType  string
}

type ConditionalLogic struct {
	Name        string
	Description string
	EntityName  string
	Conditions  []Conditioion
}

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
			StructTag(`json:"name"`),
		field.Enum("database_action").
			Values("Insert", "Update", "Delete", "All").
			StructTag(`json:"databaseAction"`),
		field.Enum("source").
			Values("Kafka", "Db").
			StructTag(`json:"source"`),
		field.String("table_name").
			Optional().
			Nillable().
			MaxLen(255).
			StructTag(`json:"tableName"`),
		field.String("topic").
			Nillable().
			Optional().
			MaxLen(255).
			StructTag(`json:"topic"`),
		field.Text("description").
			Optional().
			Nillable().
			StructTag(`json:"description"`),
		field.String("custom_subject").
			Optional().
			Nillable().
			MaxLen(255).
			StructTag(`json:"customSubject"`),
		field.String("function_name").
			Optional().
			Nillable().
			MaxLen(50).
			StructTag(`json:"functionName"`),
		field.String("trigger_name").
			Optional().
			Nillable().
			MaxLen(50).
			StructTag(`json:"triggerName"`),
		field.String("listener_name").
			Optional().
			Nillable().
			MaxLen(50).
			StructTag(`json:"listenerName"`),
		field.Text("email_recipients").
			Optional().
			Nillable().
			StructTag(`json:"emailRecipients"`),
		field.JSON("conditional_logic", &ConditionalLogic{}).
			Optional().
			StructTag(`json:"conditionalLogic"`),
		field.Time("effective_date").
			Optional().
			Nillable().
			StructTag(`json:"effectiveDate"`),
		field.Time("expiration_date").
			Optional().
			Nillable().
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
