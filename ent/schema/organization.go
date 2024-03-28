package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
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
			StorageKey(edge.Column("organization_id")).
			Unique(),
		edge.To("dispatch_control", DispatchControl.Type).
			StorageKey(edge.Column("organization_id")).
			Unique(),
		edge.To("feasibility_tool_control", FeasibilityToolControl.Type).
			StorageKey(edge.Column("organization_id")).
			Unique(),
		edge.To("invoice_control", InvoiceControl.Type).
			StorageKey(edge.Column("organization_id")).
			Unique(),
		edge.To("route_control", RouteControl.Type).
			StorageKey(edge.Column("organization_id")).
			Unique(),
		edge.To("shipment_control", ShipmentControl.Type).
			StorageKey(edge.Column("organization_id")).
			Unique(),
		edge.To("email_control", EmailControl.Type).
			StorageKey(edge.Column("organization_id")).
			Unique(),
		edge.To("google_api", GoogleApi.Type).
			StorageKey(edge.Column("organization_id")).
			Unique(),
	}
}

// Mixin of the Organization.
func (Organization) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

// // Policy defines the privacy policy of the Organization.
// func (Organization) Policy() ent.Policy {
// 	return privacy.Policy{
// 		Mutation: privacy.MutationPolicy{
// 			rule.AllowIfAdmin(),
// 		},
// 	}
// }
