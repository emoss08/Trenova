package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// BusinessUnit holds the schema definition for the BusinessUnit entity.
type BusinessUnit struct {
	ent.Schema
}

// Mixin of the BusinessUnit.
func (BusinessUnit) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

// Fields of the BusinessUnit.
func (BusinessUnit) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").
			Values("A", "I").
			Default("A"),
		field.String("name").
			MaxLen(255).
			NotEmpty(),
		field.String("entity_key").
			MaxLen(10).
			NotEmpty(),
		field.String("phone_number").
			MaxLen(15).
			StructTag(`json:"phoneNumber"`),
		field.String("address").
			Optional(),
		field.String("city").
			MaxLen(255).
			Nillable().
			Optional(),
		field.String("state").
			MaxLen(2).
			Nillable().
			Optional(),
		field.String("country").
			MaxLen(2).
			Nillable().
			Optional(),
		field.String("postal_code").
			MaxLen(10).
			Nillable().
			Optional().
			StructTag(`json:"postalCode"`),
		field.String("tax_id").
			MaxLen(20).
			Nillable().
			Optional().
			StructTag(`json:"taxId"`),
		field.String("subscription_plan").
			Optional().
			Nillable().
			StructTag(`json:"subscriptionPlan"`),
		field.String("description").
			Nillable().
			Optional(),
		field.String("legal_name").
			Optional().
			Nillable().
			StructTag(`json:"legalName"`),
		field.String("contact_name").
			Optional().
			Nillable().
			StructTag(`json:"contactName"`),
		field.String("contact_email").
			Optional().
			Nillable().
			StructTag(`json:"contactEmail"`),
		field.Time("paid_until").
			Optional().
			Nillable().
			StructTag(`json:"-"`),
		field.JSON("settings", map[string]interface{}{}).
			Optional(),
		field.Bool("free_trial").
			Default(false).
			StructTag(`json:"freeTrial"`),
		field.UUID("parent_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"parentId"`),
	}
}

// Edges of the BusinessUnit.
func (BusinessUnit) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("next", BusinessUnit.Type).
			Unique().
			From("prev").
			Unique().
			Field("parent_id").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			StructTag(`json:"parent_id"`),
		edge.To("organizations", Organization.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// Indexes of the BusinessUnit.
func (BusinessUnit) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").
			Unique().
			Annotations(
				entsql.DefaultExpr("lower(name)"),
			),
		index.Fields("entity_key").
			Unique(),
	}
}
