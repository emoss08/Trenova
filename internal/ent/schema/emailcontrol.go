package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// EmailControl holds the schema definition for the EmailControl entity.
type EmailControl struct {
	ent.Schema
}

// Fields of the EmailControl.
func (EmailControl) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("billing_email_profile_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"billingEmailProfileId" validate:"omitempty"`),
		field.UUID("rate_expirtation_email_profile_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"rateExpirtationEmailProfileId" validate:"omitempty"`),
	}
}

// Mixin of the EmailControl.
func (EmailControl) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

// Edges of the EmailControl.
func (EmailControl) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("email_control").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
		edge.To("business_unit", BusinessUnit.Type).
			StorageKey(edge.Column("business_unit_id")).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
		edge.To("billing_email_profile", EmailProfile.Type).
			Field("billing_email_profile_id").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
		edge.To("rate_email_profile", EmailProfile.Type).
			Field("rate_expirtation_email_profile_id").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Unique(),
	}
}
