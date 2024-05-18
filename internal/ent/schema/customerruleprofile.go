package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// CustomerRuleProfile holds the schema definition for the CustomerRuleProfile entity.
type CustomerRuleProfile struct {
	ent.Schema
}

// Fields of the CustomerRuleProfile.
func (CustomerRuleProfile) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("customer_id", uuid.UUID{}).
			Immutable().
			Unique(),
		field.Enum("billing_cycle").
			Values("PER_SHIPMENT", "QUARTERLY", "MONTHLY", "ANNUALLY").
			Default("PER_SHIPMENT").
			StructTag(`json:"billingCycle" validate:"required,oneof=PER_JOB QUARTERLY MONTHLY ANNUALLY"`),
	}
}

// Edges of the CustomerRuleProfile.
func (CustomerRuleProfile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("customer", Customer.Type).
			Field("customer_id").
			Ref("rule_profile").
			Unique().
			Required().
			Immutable(),
		edge.To("document_classifications", DocumentClassification.Type),
	}
}

// Mixin of the CustomerRuleProfile.
func (CustomerRuleProfile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}
