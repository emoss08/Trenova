package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Organization holds the schema definition for the Organization entity.
type Organization struct {
	ent.Schema
}

// Fields of the Organization.
func (Organization) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("business_unit_id", uuid.UUID{}).
			StructTag(`json:"businessUnitId"`),
		field.String("name").
			MaxLen(100),
		field.String("scac_code").
			MaxLen(4).
			StructTag(`json:"scacCode"`),
		field.String("dot_number").
			MaxLen(12).
			StructTag(`json:"dotNumber"`),
		field.String("logo_url").
			Optional().
			Nillable().
			StructTag(`json:"logoUrl"`),
		field.Enum("org_type").
			Values("A", "B", "X").
			Default("A").
			StructTag(`json:"orgType"`),
		field.Enum("timezone").
			Values("AmericaLosAngeles", "AmericaDenver", "AmericaChicago", "AmericaNewYork").
			Default("AmericaLosAngeles"),
	}
}

// Edges of the Organization.
func (Organization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("business_unit", BusinessUnit.Type).Ref("organizations").
			Field("business_unit_id").
			Required().
			Unique(),
		edge.To("accounting_control", AccountingControl.Type).
			StorageKey(edge.Column("organization_id")).
			Unique(),
		edge.To("billing_control", BillingControl.Type).
			Unique(),
		edge.To("dispatch_control", DispatchControl.Type).
			Unique(),
	}
}

func (Organization) Indexes() []ent.Index {
	return []ent.Index{
		// Each organization inside a business unit must have a unique ScacCode.
		index.Fields("business_unit_id", "scac_code").
			Unique(),
	}
}

// Mixin of the Organization.
func (Organization) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}
