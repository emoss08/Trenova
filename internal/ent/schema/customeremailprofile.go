package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// CustomerEmailProfile holds the schema definition for the CustomerEmailProfile entity.
type CustomerEmailProfile struct {
	ent.Schema
}

// Fields of the CustomerEmailProfile.
func (CustomerEmailProfile) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("customer_id", uuid.UUID{}).
			Immutable().
			Unique(),
		field.String("subject").
			Optional().
			MaxLen(100).
			SchemaType(map[string]string{
				dialect.Postgres: "VARCHAR(100)",
				dialect.SQLite:   "VARCHAR(100)",
			}).
			StructTag(`json:"code" validate:"omitempty,max=100"`),
		field.UUID("email_profile_id", uuid.UUID{}).
			Nillable().
			Unique().
			Optional(),
		field.Text("email_recipients").
			NotEmpty().
			StructTag(`json:"emailRecipients" validate:"omitempty,commaSeparatedEmails"`),
		field.Text("email_cc_recipients").
			Optional().
			StructTag(`json:"emailCcRecipients" validate:"omitempty,commaSeparatedEmails"`),
		field.Text("attachment_name").
			Optional().
			StructTag(`json:"attachmentName" validate:"omitempty"`),
		field.Enum("email_format").
			Values("PLAIN", "HTML").
			Default("PLAIN").
			StructTag(`json:"emailFormat" validate:"required,oneof=PLAIN HTML"`),
		// field.UUID("template_id", uuid.UUID{}).
		// 	Optional(),
	}
}

// Edges of the CustomerEmailProfile.
func (CustomerEmailProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", Customer.Type).
			Field("customer_id").
			Ref("email_profile").
			Unique().
			Required().
			Immutable(),
		edge.To("email_profile", EmailProfile.Type).
			Field("email_profile_id").
			Unique().
			StructTag(`json:"emailProfile"`),
	}
}

// Mixin of the CustomerEmailProfile.
func (CustomerEmailProfile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
